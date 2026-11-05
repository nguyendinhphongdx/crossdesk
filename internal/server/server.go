// Package server runs on the machine with the physical keyboard/mouse. When
// the cursor is pushed past the configured screen edge it swallows local input
// and streams it to the connected client until the cursor comes back.
package server

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"sharedevice/internal/capture"
	"sharedevice/internal/proto"
)

type Config struct {
	Listen   string
	Password string
	Side     string  // where the client screen is: left, right, top, bottom
	Speed    float64 // pointer speed multiplier on the client
	Verbose  bool
}

// Platform hooks (replaced in tests).
type Platform struct {
	Freeze func(bool)
	Warp   func(x, y float64)
	Bounds func() capture.Bounds
}

type client struct {
	conn   *proto.Conn
	out    chan proto.Msg
	w, h   int32
	name   string
	closed bool
}

type Server struct {
	cfg Config
	pf  Platform

	mu          sync.Mutex
	cl          *client
	remote      bool
	vx, vy      float64
	local       capture.Bounds
	localKeys   map[uint16]bool // keys held down on the server before switching
	buttonsHeld int
}

func New(cfg Config, pf Platform) (*Server, error) {
	switch cfg.Side {
	case "left", "right", "top", "bottom":
	default:
		return nil, errors.New("side must be left, right, top or bottom")
	}
	if cfg.Speed <= 0 {
		cfg.Speed = 1
	}
	return &Server{cfg: cfg, pf: pf, localKeys: map[uint16]bool{}}, nil
}

// Handle processes one captured event. It returns true if the event must be
// swallowed locally.
func (s *Server) Handle(ev capture.Event) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.remote {
		switch ev.Kind {
		case capture.Key:
			if ev.Down {
				s.localKeys[ev.Key] = true
			} else {
				delete(s.localKeys, ev.Key)
			}
		case capture.Button:
			if ev.Down {
				s.buttonsHeld++
			} else if s.buttonsHeld > 0 {
				s.buttonsHeld--
			}
		case capture.Move:
			// Don't switch while dragging: the button-up would land on the wrong machine.
			if s.cl != nil && s.buttonsHeld == 0 && s.atEdge(ev) {
				s.enter(ev)
				return true
			}
		}
		return false
	}

	switch ev.Kind {
	case capture.Move:
		s.vx += ev.DX * s.cfg.Speed
		s.vy += ev.DY * s.cfg.Speed
		if s.leaving() {
			s.leave(true)
			return true
		}
		s.clamp()
		s.send(proto.Msg{Type: proto.MouseMove, X: int32(s.vx), Y: int32(s.vy)})
	case capture.Button:
		s.send(proto.Msg{Type: proto.MouseButton, Button: byte(ev.Button), Down: ev.Down})
	case capture.Scroll:
		s.send(proto.Msg{Type: proto.Wheel, X: ev.ScrollX, Y: ev.ScrollY})
	case capture.Key:
		// A key pressed before switching must be released locally, or it sticks.
		if !ev.Down && s.localKeys[ev.Key] {
			delete(s.localKeys, ev.Key)
			return false
		}
		s.send(proto.Msg{Type: proto.Key, Key: ev.Key, Down: ev.Down})
	}
	return true
}

func (s *Server) atEdge(ev capture.Event) bool {
	b := s.pf.Bounds()
	switch s.cfg.Side {
	case "right":
		return ev.DX > 0 && ev.X >= b.MaxX-1
	case "left":
		return ev.DX < 0 && ev.X <= b.MinX
	case "bottom":
		return ev.DY > 0 && ev.Y >= b.MaxY-1
	case "top":
		return ev.DY < 0 && ev.Y <= b.MinY
	}
	return false
}

func ratio(v, lo, hi float64) float64 {
	if hi <= lo {
		return 0.5
	}
	r := (v - lo) / (hi - lo)
	return min(max(r, 0), 1)
}

func (s *Server) enter(ev capture.Event) {
	b := s.pf.Bounds()
	s.local = b
	w, h := float64(s.cl.w), float64(s.cl.h)
	switch s.cfg.Side {
	case "right":
		s.vx, s.vy = 0, ratio(ev.Y, b.MinY, b.MaxY)*(h-1)
	case "left":
		s.vx, s.vy = w-1, ratio(ev.Y, b.MinY, b.MaxY)*(h-1)
	case "bottom":
		s.vx, s.vy = ratio(ev.X, b.MinX, b.MaxX)*(w-1), 0
	case "top":
		s.vx, s.vy = ratio(ev.X, b.MinX, b.MaxX)*(w-1), h-1
	}
	s.remote = true
	s.pf.Freeze(true)
	s.send(proto.Msg{Type: proto.Enter, X: int32(s.vx), Y: int32(s.vy)})
	s.send(proto.Msg{Type: proto.MouseMove, X: int32(s.vx), Y: int32(s.vy)})
	log.Printf("-> switched to %s", s.cl.name)
}

func (s *Server) leaving() bool {
	switch s.cfg.Side {
	case "right":
		return s.vx < 0
	case "left":
		return s.vx > float64(s.cl.w-1)
	case "bottom":
		return s.vy < 0
	case "top":
		return s.vy > float64(s.cl.h-1)
	}
	return false
}

func (s *Server) clamp() {
	s.vx = min(max(s.vx, 0), float64(s.cl.w-1))
	s.vy = min(max(s.vy, 0), float64(s.cl.h-1))
}

// leave returns control to the local machine. Caller holds s.mu.
func (s *Server) leave(notify bool) {
	if !s.remote {
		return
	}
	s.remote = false
	b := s.local
	var x, y float64
	if s.cl != nil {
		w, h := float64(s.cl.w), float64(s.cl.h)
		switch s.cfg.Side {
		case "right":
			x, y = b.MaxX-2, b.MinY+ratio(s.vy, 0, h-1)*(b.MaxY-b.MinY-1)
		case "left":
			x, y = b.MinX+1, b.MinY+ratio(s.vy, 0, h-1)*(b.MaxY-b.MinY-1)
		case "bottom":
			x, y = b.MinX+ratio(s.vx, 0, w-1)*(b.MaxX-b.MinX-1), b.MaxY-2
		case "top":
			x, y = b.MinX+ratio(s.vx, 0, w-1)*(b.MaxX-b.MinX-1), b.MinY+1
		}
	}
	s.pf.Freeze(false)
	if s.cl != nil {
		s.pf.Warp(x, y)
	}
	if notify {
		s.send(proto.Msg{Type: proto.Leave})
	}
	log.Printf("<- back to local")
}

// send queues a message for the client without blocking the event tap.
func (s *Server) send(m proto.Msg) {
	if s.cl == nil || s.cl.closed {
		return
	}
	select {
	case s.cl.out <- m:
	default:
		if s.cfg.Verbose {
			log.Printf("output queue full, dropping %v", m.Type)
		}
	}
}

// Serve accepts clients forever. Only one client is active at a time; a new
// connection replaces the old one.
func (s *Server) Serve() error {
	ln, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		return err
	}
	log.Printf("listening on %s (client is on the %s)", ln.Addr(), s.cfg.Side)
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(c)
	}
}

func (s *Server) handleConn(c net.Conn) {
	if tc, ok := c.(*net.TCPConn); ok {
		tc.SetNoDelay(true)
	}
	conn, err := proto.ServerHandshake(c, s.cfg.Password)
	if err != nil {
		log.Printf("%s: handshake: %v", c.RemoteAddr(), err)
		c.Close()
		return
	}
	hello, err := conn.Recv(10 * time.Second)
	if err != nil || hello.Type != proto.Hello || hello.W <= 0 || hello.H <= 0 {
		log.Printf("%s: rejected: %v", c.RemoteAddr(), errOr(err, "bad hello"))
		conn.Close()
		return
	}
	if err := conn.Send(proto.Msg{Type: proto.Hello}); err != nil {
		conn.Close()
		return
	}
	cl := &client{conn: conn, out: make(chan proto.Msg, 4096), w: hello.W, h: hello.H, name: hello.Name}
	log.Printf("client %q connected from %s (%dx%d)", cl.name, c.RemoteAddr(), cl.w, cl.h)

	s.mu.Lock()
	if old := s.cl; old != nil {
		s.leave(false)
		s.closeClient(old)
	}
	s.cl = cl
	s.mu.Unlock()

	done := make(chan struct{})
	go func() { // writer
		for m := range cl.out {
			if err := conn.Send(m); err != nil {
				conn.Close()
				break
			}
		}
		for range cl.out { // drain until closed
		}
	}()
	go func() { // keepalive
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				s.mu.Lock()
				if s.cl == cl {
					s.send(proto.Msg{Type: proto.Ping})
				}
				s.mu.Unlock()
			}
		}
	}()
	for {
		if _, err := conn.Recv(8 * time.Second); err != nil {
			log.Printf("client %q disconnected: %v", cl.name, err)
			break
		}
	}
	close(done)
	conn.Close()
	s.mu.Lock()
	if s.cl == cl {
		s.leave(false)
		s.closeClient(cl)
		s.cl = nil
	}
	s.mu.Unlock()
}

// closeClient closes the client's queue. Caller holds s.mu (send also runs
// under s.mu, so no send can race with the close).
func (s *Server) closeClient(cl *client) {
	if !cl.closed {
		cl.closed = true
		close(cl.out)
		cl.conn.Close()
	}
}

func errOr(err error, msg string) any {
	if err != nil {
		return err
	}
	return msg
}
