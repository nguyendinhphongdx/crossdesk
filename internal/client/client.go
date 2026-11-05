// Package client connects to a server and replays the received input locally.
package client

import (
	"errors"
	"log"
	"net"
	"time"

	"sharedevice/internal/inject"
	"sharedevice/internal/keymap"
	"sharedevice/internal/proto"
)

type Config struct {
	Server      string
	Password    string
	Name        string
	SwapCmdCtrl bool
	Verbose     bool
}

// Run connects and reconnects forever.
func Run(cfg Config, inj inject.Injector) error {
	for {
		err := runOnce(cfg, inj)
		if errors.Is(err, proto.ErrAuth) {
			return errors.New("wrong password")
		}
		log.Printf("disconnected: %v; retrying in 2s", err)
		time.Sleep(2 * time.Second)
	}
}

type session struct {
	cfg     Config
	inj     inject.Injector
	scr     inject.Screen
	keys    map[uint16]bool
	buttons map[int]bool
}

func runOnce(cfg Config, inj inject.Injector) error {
	c, err := net.DialTimeout("tcp", cfg.Server, 5*time.Second)
	if err != nil {
		return err
	}
	if tc, ok := c.(*net.TCPConn); ok {
		tc.SetNoDelay(true)
	}
	conn, err := proto.ClientHandshake(c, cfg.Password)
	if err != nil {
		c.Close()
		return err
	}
	defer conn.Close()

	s := &session{cfg: cfg, inj: inj, scr: inj.Screen(), keys: map[uint16]bool{}, buttons: map[int]bool{}}
	defer s.releaseAll()

	if err := conn.Send(proto.Msg{Type: proto.Hello, X: s.scr.X, Y: s.scr.Y, W: s.scr.W, H: s.scr.H, Name: cfg.Name}); err != nil {
		return err
	}
	ack, err := conn.Recv(10 * time.Second)
	if err != nil {
		if errors.Is(err, proto.ErrAuth) {
			return err
		}
		// The server drops us silently on a bad password.
		return errors.New("server closed the connection (wrong password?): " + err.Error())
	}
	if ack.Type != proto.Hello {
		return errors.New("unexpected handshake reply")
	}
	log.Printf("connected to %s (screen %dx%d)", cfg.Server, s.scr.W, s.scr.H)

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stop:
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

func (s *session) apply(m proto.Msg) {
	switch m.Type {
	case proto.Enter:
		s.scr = s.inj.Screen() // resolution may have changed
		log.Printf("cursor entered")
		s.inj.MoveTo(m.X, m.Y)
	case proto.Leave:
		log.Printf("cursor left")
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

// releaseAll lifts every key/button still held, so nothing stays stuck when
// the cursor leaves or the connection drops.
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
