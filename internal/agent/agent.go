// Package agent is the long-running app on every device: it discovers peers,
// keeps the shared group configuration in sync, and switches this device
// between host (sharing its keyboard/mouse), client and idle.
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"sharedevice/internal/capture"
	"sharedevice/internal/client"
	"sharedevice/internal/inject"
	"sharedevice/internal/server"
)

const (
	InputPort     = 24800
	DiscoveryPort = 24802
)

type Agent struct {
	version string
	mu      sync.Mutex
	cfg     *Config
	key     []byte
	peers   map[string]*peer
	inj     inject.Injector
	srv     *server.Server

	captureRunning bool
	captureErr     string

	cli       *client.Client
	cliTarget string

	udp     *net.UDPConn
	changed chan struct{}
}

func New(version string) (*Agent, error) {
	cfg := loadConfig()
	if err := cfg.save(); err != nil {
		log.Printf("cannot save config: %v", err)
	}
	inj, err := inject.New()
	if err != nil {
		return nil, err
	}
	a := &Agent{version: version, cfg: cfg, key: groupKey(cfg.Password), peers: map[string]*peer{}, inj: inj, changed: make(chan struct{}, 1)}
	a.srv = server.New(server.Config{SelfID: cfg.ID, Password: cfg.Password, Speed: cfg.Speed},
		server.Platform{Freeze: capture.Freeze, Warp: capture.Warp, Bounds: capture.ScreenBounds})
	a.srv.SetLayout(cfg.Shared.Layout)
	return a, nil
}

// Run starts networking and blocks.
func (a *Agent) Run() error {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(InputPort))
	if err != nil {
		return fmt.Errorf("port %d is busy (is another copy running?): %w", InputPort, err)
	}
	go a.srv.Serve(ln)

	udp, err := net.ListenUDP("udp4", &net.UDPAddr{Port: DiscoveryPort})
	if err != nil {
		return fmt.Errorf("discovery port %d: %w", DiscoveryPort, err)
	}
	a.udp = udp
	go a.receiveBeacons()

	log.Printf("device %s (%s) ready", a.cfg.Name, a.cfg.ID)
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		a.reconcile()
		a.sendBeacon()
		select {
		case <-t.C:
		case <-a.changed:
		}
	}
}

func (a *Agent) poke() {
	select {
	case a.changed <- struct{}{}:
	default:
	}
}

func (a *Agent) role() string {
	switch {
	case a.cfg.Shared.Host == a.cfg.ID:
		return "host"
	case a.cfg.Shared.Host != "":
		return "client"
	}
	return "idle"
}

func (a *Agent) sendBeacon() {
	a.mu.Lock()
	scr := a.inj.Screen()
	b := Beacon{App: beaconApp, ID: a.cfg.ID, Name: a.cfg.Name, OS: runtime.GOOS, Port: InputPort,
		W: scr.W, H: scr.H, Role: a.role(), Time: time.Now().Unix()}
	if a.cli != nil {
		b.Connected = a.cli.Connected()
	}
	if a.key != nil {
		sh := a.cfg.Shared.clone()
		b.Shared = &sh
	}
	b = b.sign(a.key)
	a.mu.Unlock()

	raw, _ := json.Marshal(b)
	for _, addr := range broadcastAddrs(DiscoveryPort) {
		a.udp.WriteToUDP(raw, addr)
	}
}

func (a *Agent) receiveBeacons() {
	buf := make([]byte, 64*1024)
	for {
		n, from, err := a.udp.ReadFromUDP(buf)
		if err != nil {
			log.Printf("discovery: %v", err)
			return
		}
		var b Beacon
		if json.Unmarshal(buf[:n], &b) != nil || b.App != beaconApp || b.ID == "" {
			continue
		}
		a.mu.Lock()
		if b.ID == a.cfg.ID {
			a.mu.Unlock()
			continue
		}
		trusted := b.verify(a.key)
		p := a.peers[b.ID]
		isNew := p == nil
		if isNew {
			p = &peer{}
			a.peers[b.ID] = p
		}
		p.Beacon, p.IP, p.LastSeen, p.Trusted = b, from.IP, time.Now(), trusted
		adopted := false
		if trusted && b.Shared != nil && b.Shared.newerThan(a.cfg.Shared) {
			a.cfg.Shared = b.Shared.clone()
			a.cfg.save()
			adopted = true
		}
		a.mu.Unlock()
		if isNew {
			log.Printf("found %s (%s, %s) trusted=%v", b.Name, b.OS, from.IP, trusted)
		}
		if adopted {
			log.Printf("adopted group config #%d from %s (host=%s)", b.Shared.Epoch, b.Name, b.Shared.Host)
			a.poke()
		}
	}
}

// reconcile makes this device's running role match the shared config.
func (a *Agent) reconcile() {
	a.mu.Lock()
	defer a.mu.Unlock()
	sh := &a.cfg.Shared
	a.srv.SetLayout(sh.Layout)

	if sh.Host == a.cfg.ID && a.cfg.Password != "" {
		a.stopClientLocked()
		a.autoPlaceLocked()
		a.srv.SetActive(true)
		a.startCaptureLocked()
		return
	}
	a.srv.SetActive(false)

	var target string
	if p := a.peers[sh.Host]; p != nil && p.Trusted && p.online() {
		target = net.JoinHostPort(p.IP.String(), strconv.Itoa(p.Port))
	}
	if target != a.cliTarget {
		a.stopClientLocked()
		if target != "" {
			swap := a.cfg.SwapCmdCtrl && (p0(a.peers[sh.Host]).OS == "darwin") != (runtime.GOOS == "darwin")
			a.cli = client.Start(client.Config{Server: target, Password: a.cfg.Password, ID: a.cfg.ID, Name: a.cfg.Name, SwapCmdCtrl: swap}, a.inj)
			a.cliTarget = target
			log.Printf("following host %s at %s (swap Cmd/Ctrl: %v)", a.peers[sh.Host].Name, target, swap)
		}
	}
}

func p0(p *peer) peer {
	if p == nil {
		return peer{}
	}
	return *p
}

func (a *Agent) stopClientLocked() {
	if a.cli != nil {
		c := a.cli
		a.cli, a.cliTarget = nil, ""
		go c.Stop()
	}
}

// autoPlaceLocked puts trusted devices that are not on the layout yet next to
// the host, so a fresh setup works without touching the layout editor.
func (a *Agent) autoPlaceLocked() {
	sh := &a.cfg.Shared
	changed := false
	if _, ok := sh.Layout[a.cfg.ID]; !ok {
		sh.Layout[a.cfg.ID] = server.Cell{}
		changed = true
	}
	home := sh.Layout[a.cfg.ID]
	used := map[server.Cell]bool{}
	for _, c := range sh.Layout {
		used[c] = true
	}
	ids := make([]string, 0, len(a.peers))
	for id, p := range a.peers {
		if _, placed := sh.Layout[id]; !placed && p.Trusted && p.online() {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, dx := range []int{1, -1, 2, -2, 3, -3} {
			c := server.Cell{X: home.X + dx, Y: home.Y}
			if !used[c] {
				sh.Layout[id], used[c], changed = c, true, true
				log.Printf("placed %s at %v", a.peers[id].Name, c)
				break
			}
		}
	}
	if changed {
		a.bumpLocked()
	}
}

func (a *Agent) bumpLocked() {
	sh := &a.cfg.Shared
	max := sh.Epoch
	for _, p := range a.peers {
		if p.Trusted && p.Shared != nil && p.Shared.Epoch > max {
			max = p.Shared.Epoch
		}
	}
	sh.Epoch, sh.Author = max+1, a.cfg.ID
	a.srv.SetLayout(sh.Layout)
	a.cfg.save()
}

func (a *Agent) startCaptureLocked() {
	if a.captureRunning {
		return
	}
	if !capture.Trusted(false) {
		a.captureErr = "missing-permission"
		return
	}
	a.captureRunning, a.captureErr = true, ""
	go func() {
		err := capture.Run(a.srv.Handle)
		a.mu.Lock()
		a.captureRunning = false
		if err != nil {
			a.captureErr = err.Error()
			log.Printf("capture: %v", err)
		}
		a.mu.Unlock()
	}()
}

// ---- API used by the UI ----

type DeviceInfo struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	OS        string       `json:"os"`
	IP        string       `json:"ip,omitempty"`
	Self      bool         `json:"self"`
	Online    bool         `json:"online"`
	Trusted   bool         `json:"trusted"`
	Role      string       `json:"role"`
	Connected bool         `json:"connected"`
	W         int32        `json:"w"`
	H         int32        `json:"h"`
	Cell      *server.Cell `json:"cell,omitempty"`
}

type Status struct {
	Self          DeviceInfo   `json:"self"`
	HasPassword   bool         `json:"hasPassword"`
	Host          string       `json:"host"`
	Devices       []DeviceInfo `json:"devices"`
	CaptureError  string       `json:"captureError,omitempty"`
	PermissionOK  bool         `json:"permissionOK"`
	Version       string       `json:"version"`
	ClientError   string       `json:"clientError,omitempty"`
	SwapCmdCtrl   bool         `json:"swapCmdCtrl"`
	Speed         float64      `json:"speed"`
	ConnectedToMe []string     `json:"connectedToMe"`
}

func (a *Agent) Status() Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	sh := a.cfg.Shared
	cell := func(id string) *server.Cell {
		if c, ok := sh.Layout[id]; ok {
			return &c
		}
		return nil
	}
	scr := a.inj.Screen()
	self := DeviceInfo{ID: a.cfg.ID, Name: a.cfg.Name, OS: runtime.GOOS, Self: true, Online: true, Trusted: a.key != nil,
		Role: a.role(), W: scr.W, H: scr.H, Cell: cell(a.cfg.ID)}
	if a.cli != nil {
		self.Connected = a.cli.Connected()
	}
	st := Status{Self: self, HasPassword: a.cfg.Password != "", Host: sh.Host, CaptureError: a.captureErr,
		SwapCmdCtrl: a.cfg.SwapCmdCtrl, Speed: a.cfg.Speed, ConnectedToMe: a.srv.Connected(),
		PermissionOK: capture.Trusted(false), Version: a.version}
	if a.cli != nil {
		st.ClientError = a.cli.LastError()
	}
	st.Devices = append(st.Devices, self)
	for id, p := range a.peers {
		role := "idle"
		if p.Trusted {
			role = p.Role
		}
		st.Devices = append(st.Devices, DeviceInfo{ID: id, Name: p.Name, OS: p.OS, IP: p.IP.String(), Online: p.online(),
			Trusted: p.Trusted, Role: role, Connected: p.Connected, W: p.W, H: p.H, Cell: cell(id)})
	}
	sort.Slice(st.Devices[1:], func(i, j int) bool { return st.Devices[1+i].Name < st.Devices[1+j].Name })
	return st
}

func (a *Agent) SetPassword(pw string) error {
	if len(pw) < 4 {
		return errors.New("mật khẩu cần ít nhất 4 ký tự")
	}
	a.mu.Lock()
	a.cfg.Password = pw
	a.key = groupKey(pw)
	a.srv.SetPassword(pw)
	for _, p := range a.peers {
		p.Trusted = false // re-verified on the next beacon
	}
	a.stopClientLocked()
	a.cfg.save()
	a.mu.Unlock()
	a.poke()
	return nil
}

func (a *Agent) SetName(name string) error {
	if name == "" || len(name) > 40 {
		return errors.New("tên không hợp lệ")
	}
	a.mu.Lock()
	a.cfg.Name = name
	a.cfg.save()
	a.mu.Unlock()
	a.poke()
	return nil
}

func (a *Agent) SetOptions(swap bool, speed float64) {
	a.mu.Lock()
	a.cfg.SwapCmdCtrl = swap
	if speed >= 0.25 && speed <= 5 {
		a.cfg.Speed = speed
		a.srv.SetSpeed(speed)
	}
	a.stopClientLocked() // reconnect with the new swap setting
	a.cfg.save()
	a.mu.Unlock()
	a.poke()
}

// SetHost makes device id the host for the whole group ("" = stop sharing).
func (a *Agent) SetHost(id string) error {
	a.mu.Lock()
	defer a.poke()
	defer a.mu.Unlock()
	if a.key == nil {
		return errors.New("hãy đặt mật khẩu nhóm trước")
	}
	if id != "" && id != a.cfg.ID {
		if p := a.peers[id]; p == nil || !p.Trusted {
			return errors.New("thiết bị này chưa cùng nhóm (khác mật khẩu?)")
		}
	}
	a.cfg.Shared.Host = id
	a.bumpLocked()
	return nil
}

func (a *Agent) SetLayout(l map[string]server.Cell) error {
	a.mu.Lock()
	defer a.poke()
	defer a.mu.Unlock()
	if a.key == nil {
		return errors.New("hãy đặt mật khẩu nhóm trước")
	}
	seen := map[server.Cell]string{}
	for id, c := range l {
		if other, dup := seen[c]; dup {
			return fmt.Errorf("hai thiết bị cùng một ô: %s, %s", other, id)
		}
		seen[c] = id
	}
	a.cfg.Shared.Layout = l
	a.bumpLocked()
	return nil
}

func (a *Agent) RequestPermission() bool { return capture.Trusted(true) }

// Shutdown returns the cursor home and disconnects before the app exits.
func (a *Agent) Shutdown() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.srv.SetActive(false)
	if a.cli != nil {
		a.cli.Stop()
		a.cli = nil
	}
}
