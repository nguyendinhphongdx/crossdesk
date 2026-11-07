// Package client connects to the host and replays the received input locally.
package client

import (
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"crossdesk/internal/clipboard"
	"crossdesk/internal/inject"
	"crossdesk/internal/keymap"
	"crossdesk/internal/proto"
)

type Config struct {
	Server      string // host:port
	Password    string
	ID          string
	Name        string
	SwapCmdCtrl bool
	Verbose     bool
	// Clipboard sharing (optional).
	Clip        clipboard.IO
	ClipEnabled func() bool
}

// Client keeps a connection to one host until stopped.
type Client struct {
	cfg       Config
	inj       inject.Injector
	stop      chan struct{}
	done      chan struct{}
	connected atomic.Bool
	lastErr   atomic.Value // string
}

// Start connects in the background, reconnecting until Stop is called.
func Start(cfg Config, inj inject.Injector) *Client {
	c := &Client{cfg: cfg, inj: inj, stop: make(chan struct{}), done: make(chan struct{})}
	go c.loop()
	return c
}

func (c *Client) Connected() bool { return c.connected.Load() }

func (c *Client) LastError() string {
	s, _ := c.lastErr.Load().(string)
	return s
}

// Stop disconnects and waits for the connection to wind down.
func (c *Client) Stop() {
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
	<-c.done
}

func (c *Client) loop() {
	defer close(c.done)
	for {
		err := c.runOnce()
		c.connected.Store(false)
		select {
		case <-c.stop:
			return
		default:
		}
		if err != nil {
			c.lastErr.Store(err.Error())
			log.Printf("client: %v; retrying", err)
		}
		select {
		case <-c.stop:
			return
		case <-time.After(2 * time.Second):
		}
	}
}

type session struct {
	cfg     Config
	inj     inject.Injector
	keys    map[uint16]bool
	buttons map[int]bool
	conn    *proto.Conn
	clip    proto.ClipAssembler
	clipMu  sync.Mutex
	synced  [32]byte // clipboard content both sides already share
}

func (c *Client) runOnce() error {
	nc, err := net.DialTimeout("tcp", c.cfg.Server, 5*time.Second)
	if err != nil {
		return err
	}
	if tc, ok := nc.(*net.TCPConn); ok {
		tc.SetNoDelay(true)
	}
	// Closing the socket is how Stop interrupts a blocking Recv.
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-c.stop:
			nc.Close()
		case <-stopped:
		}
	}()

	conn, err := proto.ClientHandshake(nc, c.cfg.Password)
	if err != nil {
		nc.Close()
		return err
	}
	defer conn.Close()

	scr := c.inj.Screen()
	s := &session{cfg: c.cfg, inj: c.inj, keys: map[uint16]bool{}, buttons: map[int]bool{}, conn: conn}
	defer s.releaseAll()

	if err := conn.Send(proto.Msg{Type: proto.Hello, X: scr.X, Y: scr.Y, W: scr.W, H: scr.H, Name: c.cfg.ID + "\n" + c.cfg.Name}); err != nil {
		return err
	}
	ack, err := conn.Recv(10 * time.Second)
	if err != nil {
		if errors.Is(err, proto.ErrAuth) {
			return err
		}
		return errors.New("host closed the connection (wrong password or not hosting)")
	}
	if ack.Type != proto.Hello {
		return errors.New("unexpected handshake reply")
	}
	c.connected.Store(true)
	c.lastErr.Store("")
	log.Printf("connected to host %s (screen %dx%d)", c.cfg.Server, scr.W, scr.H)

	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stopped:
				return
			case <-t.C:
				if conn.Send(proto.Msg{Type: proto.Ping}) != nil {
					conn.Close()
					return
				}
			}
		}
	}()

	for {
		m, err := conn.Recv(8 * time.Second)
		if err != nil {
			return err
		}
		s.apply(m)
	}
}

func (s *session) clipOn() bool {
	return s.cfg.Clip != nil && (s.cfg.ClipEnabled == nil || s.cfg.ClipEnabled())
}

// sendClip answers a ClipRequest when the local clipboard changed.
func (s *session) sendClip() {
	if !s.clipOn() {
		return
	}
	content, ok := s.cfg.Clip.Read()
	if !ok || len(content.Data) > proto.MaxClip {
		return
	}
	h := content.Hash()
	s.clipMu.Lock()
	defer s.clipMu.Unlock()
	if h == s.synced {
		return
	}
	s.synced = h
	for _, m := range proto.ClipMessages(byte(content.Kind), content.Data) {
		if s.conn.Send(m) != nil {
			return
		}
	}
}

func (s *session) apply(m proto.Msg) {
	switch m.Type {
	case proto.ClipRequest:
		go s.sendClip()
		return
	case proto.ClipBegin, proto.ClipChunk, proto.ClipEnd:
		kind, data, done, err := s.clip.Feed(m)
		if err != nil {
			log.Printf("clipboard: %v", err)
		}
		if done && s.clipOn() {
			content := clipboard.Content{Kind: clipboard.Kind(kind), Data: data}
			s.clipMu.Lock()
			s.synced = content.Hash()
			s.clipMu.Unlock()
			if err := s.cfg.Clip.Write(content); err != nil {
				log.Printf("clipboard: %v", err)
			}
		}
		return
	}
	switch m.Type {
	case proto.Enter:
		s.inj.Screen() // refresh in case the resolution changed
		s.inj.MoveTo(m.X, m.Y)
	case proto.Leave:
		s.releaseAll()
	case proto.MouseMove:
		s.inj.MoveTo(m.X, m.Y)
	case proto.MouseButton:
		s.buttons[int(m.Button)] = m.Down
		s.inj.Button(int(m.Button), m.Down)
	case proto.Wheel:
		s.inj.Wheel(m.X, m.Y)
	case proto.Key:
		k := m.Key
		if s.cfg.SwapCmdCtrl {
			k = keymap.SwapCmdCtrl(k)
		}
		if m.Down {
			s.keys[k] = true
		} else {
			delete(s.keys, k)
		}
		if s.cfg.Verbose {
			log.Printf("key 0x%02X down=%v", k, m.Down)
		}
		s.inj.Key(k, m.Down)
	}
}

// releaseAll lifts every key/button still held so nothing sticks when the
// cursor leaves or the connection drops.
func (s *session) releaseAll() {
	for k := range s.keys {
		s.inj.Key(k, false)
	}
	clear(s.keys)
	for b, down := range s.buttons {
		if down {
			s.inj.Button(b, false)
		}
	}
	clear(s.buttons)
}
