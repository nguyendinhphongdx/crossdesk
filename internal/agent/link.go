package agent

// Manual links: an optional second way to connect, for networks where
// discovery broadcasts are blocked or one machine accepts no incoming
// connections (e.g. a company PC whose firewall cannot be changed).
//
// The machine that can only dial out keeps a TCP link to the other one
// (LinkPort). Over that link both sides exchange the same signed beacons as
// over UDP, so discovery, trust and the shared config work unchanged. Input
// sessions then go:
//   - dialer is client:  it dials the host's InputPort as usual (outbound);
//   - dialer is host:    it dials the client's LinkPort again with mode 'R'
//     and serves the session on that connection, so the client never has to
//     accept a connection on InputPort.

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"crossdesk/internal/client"
	"crossdesk/internal/clipboard"
	"crossdesk/internal/proto"
)

const (
	LinkPort    = 24803
	modeLink    = 'L' // beacons
	modeReverse = 'R' // the host dialed us to run the input session
	maxManual   = 8
)

type link struct {
	conn     *proto.Conn
	addr     string // host:port we dialed, or the remote address
	ip       net.IP
	outgoing bool
	peerID   string // learned from the first beacon
	dropped  bool   // closed on purpose (reconnect / removed), not an error
}

type LinkInfo struct {
	Addr  string `json:"addr"`
	State string `json:"state"` // connecting, connected, error
	Error string `json:"error,omitempty"`
	Peer  string `json:"peer,omitempty"`
}

// normalizeAddr accepts "host" or "host:port" and returns host:port.
func normalizeAddr(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "/ ") {
		return "", fmt.Errorf("địa chỉ không hợp lệ: %q", s)
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		host, port = s, strconv.Itoa(LinkPort)
	}
	if host == "" {
		return "", fmt.Errorf("địa chỉ không hợp lệ: %q", s)
	}
	if n, err := strconv.Atoi(port); err != nil || n <= 0 || n > 65535 {
		return "", fmt.Errorf("cổng không hợp lệ: %q", s)
	}
	return net.JoinHostPort(host, port), nil
}

// listenLinks accepts manual links from other machines.
func (a *Agent) listenLinks(ln net.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go a.acceptLink(c)
	}
}

func (a *Agent) acceptLink(c net.Conn) {
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	var mode [1]byte
	if _, err := c.Read(mode[:]); err != nil {
		c.Close()
		return
	}
	c.SetReadDeadline(time.Time{})
	a.mu.Lock()
	pw := a.cfg.Password
	a.mu.Unlock()
	if pw == "" {
		c.Close()
		return
	}
	switch mode[0] {
	case modeLink:
		conn, err := proto.ServerHandshake(c, pw)
		if err != nil {
			c.Close()
			return
		}
		ip := net.IP(nil)
		if ta, ok := c.RemoteAddr().(*net.TCPAddr); ok {
			ip = ta.IP
		}
		a.runLink(&link{conn: conn, addr: c.RemoteAddr().String(), ip: ip})
	case modeReverse:
		a.acceptReverse(c)
	default:
		c.Close()
	}
}

// acceptReverse runs our client session on a connection the host opened.
func (a *Agent) acceptReverse(c net.Conn) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.role() != "client" {
		c.Close()
		return
	}
	if a.revCli != nil {
		go a.revCli.Stop()
	}
	a.stopClientLocked()
	host := a.peers[a.cfg.Shared.Host]
	swap := a.cfg.SwapCmdCtrl && host != nil && (host.OS == "darwin") != (runtimeGOOS == "darwin")
	cli := client.StartConn(client.Config{Password: a.cfg.Password, ID: a.cfg.ID, Name: a.cfg.Name, SwapCmdCtrl: swap,
		Clip: clipboard.System, ClipEnabled: a.clipboardOn}, a.inj, c)
	a.revCli = cli
	log.Printf("host connected to us over a manual link (%s)", c.RemoteAddr())
	go func() {
		<-cli.Done()
		a.mu.Lock()
		if a.revCli == cli {
			a.revCli = nil
		}
		a.mu.Unlock()
	}()
}

// runLink registers l and reads beacons from it until it drops.
func (a *Agent) runLink(l *link) {
	a.mu.Lock()
	a.links[l] = struct{}{}
	a.mu.Unlock()
	if l.outgoing {
		a.setLinkState(l.addr, "connected", "")
	}
	err := a.readLink(l)
	l.conn.Close()
	a.mu.Lock()
	delete(a.links, l)
	a.mu.Unlock()
	a.mu.Lock()
	dropped := l.dropped
	a.mu.Unlock()
	if l.outgoing && !dropped {
		msg := "mất kết nối"
		if errors.Is(err, proto.ErrAuth) {
			msg = "sai mật khẩu nhóm"
		}
		a.setLinkState(l.addr, "error", msg)
	}
	a.poke()
}

func (a *Agent) readLink(l *link) error {
	for {
		m, err := l.conn.Recv(15 * time.Second)
		if err != nil {
			return err
		}
		if m.Type != proto.LinkBeacon {
			continue
		}
		var b Beacon
		if json.Unmarshal([]byte(m.Name), &b) != nil || b.App != beaconApp || b.ID == "" {
			continue
		}
		a.mu.Lock()
		if l.peerID != b.ID {
			l.peerID = b.ID
		}
		a.mu.Unlock()
		a.handleBeacon(b, l.ip)
	}
}

func (a *Agent) setLinkState(addr, state, errMsg string) {
	a.mu.Lock()
	a.linkState[addr] = LinkInfo{Addr: addr, State: state, Error: errMsg}
	a.mu.Unlock()
}

// dialLinks keeps a link open to every manual address.
func (a *Agent) dialLinks() {
	for {
		a.mu.Lock()
		addrs := append([]string(nil), a.cfg.ManualPeers...)
		a.mu.Unlock()
		for _, addr := range addrs {
			a.mu.Lock()
			busy := a.dialing[addr]
			for l := range a.links {
				if l.outgoing && l.addr == addr {
					busy = true
				}
			}
			if !busy {
				a.dialing[addr] = true
			}
			a.mu.Unlock()
			if !busy {
				go a.dialLink(addr)
			}
		}
		time.Sleep(3 * time.Second)
	}
}

func (a *Agent) dialLink(addr string) {
	defer func() {
		a.mu.Lock()
		delete(a.dialing, addr)
		a.mu.Unlock()
	}()
	a.mu.Lock()
	pw := a.cfg.Password
	a.mu.Unlock()
	if pw == "" {
		a.setLinkState(addr, "error", "chưa đặt mật khẩu nhóm")
		return
	}
	a.mu.Lock()
	if _, ok := a.linkState[addr]; !ok {
		a.linkState[addr] = LinkInfo{Addr: addr, State: "connecting"}
	}
	a.mu.Unlock()
	c, err := net.DialTimeout("tcp", addr, 4*time.Second)
	if err != nil {
		a.setLinkState(addr, "error", "không kết nối được tới "+addr+" — máy kia có đang chạy Crossdesk không?")
		return
	}
	if _, err := c.Write([]byte{modeLink}); err != nil {
		c.Close()
		return
	}
	conn, err := proto.ClientHandshake(c, pw)
	if err != nil {
		c.Close()
		a.setLinkState(addr, "error", err.Error())
		return
	}
	ip := net.IP(nil)
	if ta, ok := c.RemoteAddr().(*net.TCPAddr); ok {
		ip = ta.IP
	}
	l := &link{conn: conn, addr: addr, ip: ip, outgoing: true}
	a.mu.Lock()
	delete(a.dialing, addr) // the link itself now marks addr as busy
	a.mu.Unlock()
	go a.sendLinkBeacon(l, a.beaconBytes())
	a.runLink(l)
}

// sendLinkBeacon writes one beacon to a link without blocking the caller.
func (a *Agent) sendLinkBeacon(l *link, raw []byte) {
	if err := l.conn.Send(proto.Msg{Type: proto.LinkBeacon, Name: string(raw)}); err != nil {
		l.conn.Close()
	}
}

// incomingLinkFromLocked reports whether peer id reaches us through a link
// it opened — then it cannot be dialed and will dial us instead.
func (a *Agent) incomingLinkFromLocked(id string) bool {
	for l := range a.links {
		if !l.outgoing && l.peerID == id {
			return true
		}
	}
	return false
}

// reverseDialLocked starts input sessions to clients that only accept our
// outgoing links. Called by reconcile while we are host.
func (a *Agent) reverseDialLocked() {
	connected := map[string]bool{}
	for _, id := range a.srv.Connected() {
		connected[id] = true
	}
	for l := range a.links {
		id := l.peerID
		p := a.peers[id]
		if !l.outgoing || id == "" || p == nil || !p.Trusted || !p.online() || connected[id] || a.reversing[id] {
			continue
		}
		a.reversing[id] = true
		go a.reverseDial(l.addr, id)
	}
}

func (a *Agent) reverseDial(addr, id string) {
	defer func() {
		a.mu.Lock()
		delete(a.reversing, id)
		a.mu.Unlock()
	}()
	c, err := net.DialTimeout("tcp", addr, 4*time.Second)
	if err != nil {
		return
	}
	if _, err := c.Write([]byte{modeReverse}); err != nil {
		c.Close()
		return
	}
	a.srv.ServeConn(c) // blocks for the whole session
}

// Reconnect drops and re-establishes connections: one manual link (addr),
// or everything (addr == ""): manual links and the session with the host.
func (a *Agent) Reconnect(addr string) {
	a.mu.Lock()
	for l := range a.links {
		if l.outgoing && (addr == "" || l.addr == addr) {
			l.dropped = true
			l.conn.Close()
		}
	}
	for k := range a.linkState {
		if addr == "" || k == addr {
			a.linkState[k] = LinkInfo{Addr: k, State: "connecting"}
		}
	}
	if addr == "" {
		a.stopClientLocked() // reconcile starts it again on the next tick
		a.stopReverseLocked()
	}
	a.mu.Unlock()
	log.Printf("reconnecting %s", map[bool]string{true: "everything", false: addr}[addr == ""])
	a.poke()
}

// SetManualPeers replaces the list of manual addresses.
func (a *Agent) SetManualPeers(in []string) error {
	if len(in) > maxManual {
		return fmt.Errorf("tối đa %d địa chỉ", maxManual)
	}
	var out []string
	seen := map[string]bool{}
	for _, s := range in {
		addr, err := normalizeAddr(s)
		if err != nil {
			return err
		}
		if !seen[addr] {
			seen[addr] = true
			out = append(out, addr)
		}
	}
	a.mu.Lock()
	a.cfg.ManualPeers = out
	for l := range a.links {
		if l.outgoing && !seen[l.addr] {
			l.dropped = true
			l.conn.Close()
		}
	}
	for addr := range a.linkState {
		if !seen[addr] {
			delete(a.linkState, addr)
		}
	}
	a.cfg.save()
	a.mu.Unlock()
	return nil
}

// LocalIPs lists this machine's IPv4 addresses (to type on the other one).
func LocalIPs() []string {
	var out []string
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, ad := range addrs {
			if ipn, ok := ad.(*net.IPNet); ok && ipn.IP.To4() != nil && !ipn.IP.IsLinkLocalUnicast() {
				out = append(out, ipn.IP.String())
			}
		}
	}
	return out
}
