// Package server runs on the host: the machine whose physical keyboard/mouse
// is shared. Screens are placed on a grid layout; pushing the cursor past an
// edge moves it to the neighbouring screen (another client or back home).
package server

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"crossdesk/internal/capture"
	"crossdesk/internal/proto"
)

// Cell is a position on the layout grid.
type Cell struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Config struct {
	SelfID   string
	Password string
	Speed    float64 // pointer speed multiplier on clients
	Verbose  bool
}

// Platform hooks (replaced in tests).
type Platform struct {
	Freeze func(bool)
	Warp   func(x, y float64)
	Bounds func() capture.Bounds
}

type client struct {
	id     string
	name   string
	conn   *proto.Conn
	out    chan proto.Msg
	w, h   int32
	closed bool
}

type dir int

const (
	right dir = iota
	left
	down
	up
)

var dirDelta = map[dir]Cell{right: {1, 0}, left: {-1, 0}, down: {0, 1}, up: {0, -1}}

type Server struct {
	cfg Config
	pf  Platform

	mu          sync.Mutex
	active      bool
	layout      map[string]Cell
	clients     map[string]*client
	cur         *client // nil = cursor is on the host
	vx, vy      float64
	local       capture.Bounds
	localKeys   map[uint16]bool // keys held on the host before switching away
	buttonsHeld int
}

func New(cfg Config, pf Platform) *Server {
	if cfg.Speed <= 0 {
		cfg.Speed = 1
	}
	return &Server{cfg: cfg, pf: pf, layout: map[string]Cell{}, clients: map[string]*client{}, localKeys: map[uint16]bool{}}
}

// SetPassword changes the password for future connections.
func (s *Server) SetPassword(p string) {
	s.mu.Lock()
	s.cfg.Password = p
	s.mu.Unlock()
}

func (s *Server) SetSpeed(v float64) {
	s.mu.Lock()
	if v > 0 {
		s.cfg.Speed = v
	}
	s.mu.Unlock()
}

// SetActive turns host mode on or off. Turning it off returns the cursor home
// and drops all clients.
func (s *Server) SetActive(on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == on {
		return
	}
	s.active = on
	if !on {
		s.abandon()
		for id, c := range s.clients {
			s.closeClient(c)
			delete(s.clients, id)
		}
		log.Printf("host mode off")
	} else {
		log.Printf("host mode on")
	}
}

func (s *Server) SetLayout(l map[string]Cell) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.layout = make(map[string]Cell, len(l))
	for k, v := range l {
		s.layout[k] = v
	}
}

// Connected returns the IDs of connected clients.
func (s *Server) Connected() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.clients))
	for id := range s.clients {
		ids = append(ids, id)
	}
	return ids
}

// neighbor returns the device placed next to `from` in direction d.
func (s *Server) neighbor(from string, d dir) string {
	c, ok := s.layout[from]
	if !ok {
		if from != s.cfg.SelfID {
			return ""
		}
		c = Cell{} // host defaults to origin
	}
	t := Cell{c.X + dirDelta[d].X, c.Y + dirDelta[d].Y}
	for id, cell := range s.layout {
		if cell == t && id != from {
			return id
		}
	}
	return ""
}

func ratio(v, lo, hi float64) float64 {
	if hi <= lo {
		return 0.5
	}
	return min(max((v-lo)/(hi-lo), 0), 1)
}

// Handle processes one captured event and returns true if it must be
// swallowed locally.
func (s *Server) Handle(ev capture.Event) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return false
	}

	if s.cur == nil {
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
			// Never switch mid-drag: the button-up would land on the wrong machine.
			if s.buttonsHeld > 0 {
				return false
			}
			b := s.pf.Bounds()
			var d dir
			var r float64
			switch {
			case ev.DX > 0 && ev.X >= b.MaxX-1:
				d, r = right, ratio(ev.Y, b.MinY, b.MaxY)
			case ev.DX < 0 && ev.X <= b.MinX:
				d, r = left, ratio(ev.Y, b.MinY, b.MaxY)
			case ev.DY > 0 && ev.Y >= b.MaxY-1:
				d, r = down, ratio(ev.X, b.MinX, b.MaxX)
			case ev.DY < 0 && ev.Y <= b.MinY:
				d, r = up, ratio(ev.X, b.MinX, b.MaxX)
			default:
				return false
			}
			if c := s.clients[s.neighbor(s.cfg.SelfID, d)]; c != nil {
				s.local = b
				s.pf.Freeze(true)
				s.enter(c, d, r)
				return true
			}
		}
		return false
	}

	switch ev.Kind {
	case capture.Move:
		s.vx += ev.DX * s.cfg.Speed
		s.vy += ev.DY * s.cfg.Speed
		w, h := float64(s.cur.w-1), float64(s.cur.h-1)
		var d dir
		var r float64
		out := true
		switch {
		case s.vx > w:
			d, r = right, ratio(s.vy, 0, h)
		case s.vx < 0:
			d, r = left, ratio(s.vy, 0, h)
		case s.vy > h:
			d, r = down, ratio(s.vx, 0, w)
		case s.vy < 0:
			d, r = up, ratio(s.vx, 0, w)
		default:
			out = false
		}
		if out {
			n := s.neighbor(s.cur.id, d)
			if n == s.cfg.SelfID {
				s.goHome(d, r, true)
				return true
			}
			if c := s.clients[n]; c != nil {
				s.send(s.cur, proto.Msg{Type: proto.Leave})
				s.enter(c, d, r)
				return true
			}
		}
		s.vx = min(max(s.vx, 0), w)
		s.vy = min(max(s.vy, 0), h)
		s.send(s.cur, proto.Msg{Type: proto.MouseMove, X: int32(s.vx), Y: int32(s.vy)})
	case capture.Button:
		s.send(s.cur, proto.Msg{Type: proto.MouseButton, Button: byte(ev.Button), Down: ev.Down})
	case capture.Scroll:
		s.send(s.cur, proto.Msg{Type: proto.Wheel, X: ev.ScrollX, Y: ev.ScrollY})
	case capture.Key:
		// A key pressed before switching must be released locally, or it sticks.
		if !ev.Down && s.localKeys[ev.Key] {
			delete(s.localKeys, ev.Key)
			return false
		}
		s.send(s.cur, proto.Msg{Type: proto.Key, Key: ev.Key, Down: ev.Down})
	}
	return true
}

// enter moves the virtual cursor onto client c, arriving while travelling in
// direction d at relative position r along the edge.
func (s *Server) enter(c *client, d dir, r float64) {
	w, h := float64(c.w-1), float64(c.h-1)
	switch d {
	case right:
		s.vx, s.vy = 0, r*h
	case left:
		s.vx, s.vy = w, r*h
	case down:
		s.vx, s.vy = r*w, 0
	case up:
		s.vx, s.vy = r*w, h
	}
	s.cur = c
	s.send(c, proto.Msg{Type: proto.Enter, X: int32(s.vx), Y: int32(s.vy)})
	s.send(c, proto.Msg{Type: proto.MouseMove, X: int32(s.vx), Y: int32(s.vy)})
	log.Printf("-> %s", c.name)
}

// goHome returns the cursor to the host. Caller holds s.mu.
func (s *Server) goHome(d dir, r float64, notify bool) {
	if s.cur == nil {
		return
	}
	if notify {
		s.send(s.cur, proto.Msg{Type: proto.Leave})
	}
	s.cur = nil
	b := s.local
	var x, y float64
	switch d {
	case right:
		x, y = b.MinX+1, b.MinY+r*(b.MaxY-b.MinY-1)
	case left:
		x, y = b.MaxX-2, b.MinY+r*(b.MaxY-b.MinY-1)
	case down:
		x, y = b.MinX+r*(b.MaxX-b.MinX-1), b.MinY+1
	case up:
		x, y = b.MinX+r*(b.MaxX-b.MinX-1), b.MaxY-2
	}
	s.pf.Freeze(false)
	s.pf.Warp(x, y)
	log.Printf("<- home")
}

// send queues a message without blocking the input hook.
func (s *Server) send(c *client, m proto.Msg) {
	if c == nil || c.closed {
		return
	}
	select {
	case c.out <- m:
	default:
		if s.cfg.Verbose {
			log.Printf("queue full for %s, dropping %v", c.name, m.Type)
		}
	}
}

// Serve accepts client connections on ln until it is closed.
func (s *Server) Serve(ln net.Listener) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(c)
	}
}

func (s *Server) handleConn(c net.Conn) {
	s.mu.Lock()
	active, pw := s.active, s.cfg.Password
	s.mu.Unlock()
	if !active || pw == "" {
		c.Close()
		return
	}
	if tc, ok := c.(*net.TCPConn); ok {
		tc.SetNoDelay(true)
	}
	conn, err := proto.ServerHandshake(c, pw)
	if err != nil {
		c.Close()
		return
	}
	hello, err := conn.Recv(10 * time.Second)
	if err != nil || hello.Type != proto.Hello || hello.W <= 0 || hello.H <= 0 {
		log.Printf("%s rejected: %v", c.RemoteAddr(), err)
		conn.Close()
		return
	}
	id, name, _ := strings.Cut(hello.Name, "\n")
	if err := conn.Send(proto.Msg{Type: proto.Hello}); err != nil {
		conn.Close()
		return
	}
	cl := &client{id: id, name: name, conn: conn, out: make(chan proto.Msg, 4096), w: hello.W, h: hello.H}

	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		conn.Close()
		return
	}
	if old := s.clients[id]; old != nil {
		if s.cur == old {
			s.abandon()
		}
		s.closeClient(old)
	}
	s.clients[id] = cl
	s.mu.Unlock()
	log.Printf("client %q connected from %s (%dx%d)", name, c.RemoteAddr(), cl.w, cl.h)

	done := make(chan struct{})
	go func() {
		for m := range cl.out {
			if conn.Send(m) != nil {
				conn.Close()
				break
			}
		}
		for range cl.out {
		}
	}()
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				s.mu.Lock()
				s.send(cl, proto.Msg{Type: proto.Ping})
				s.mu.Unlock()
			}
		}
	}()
	for {
		if _, err := conn.Recv(8 * time.Second); err != nil {
			log.Printf("client %q disconnected: %v", name, err)
			break
		}
	}
	close(done)
	s.mu.Lock()
	if s.clients[id] == cl {
		if s.cur == cl {
			s.abandon()
		}
		delete(s.clients, id)
	}
	s.closeClient(cl)
	s.mu.Unlock()
}

// closeClient closes the queue. Caller holds s.mu, as do all senders.
func (s *Server) closeClient(c *client) {
	if !c.closed {
		c.closed = true
		close(c.out)
		c.conn.Close()
	}
}

// abandon gives the cursor back to the host without moving it, used when the
// active client vanishes. Caller holds s.mu.
func (s *Server) abandon() {
	if s.cur != nil {
		s.cur = nil
		s.pf.Freeze(false)
		log.Printf("<- home (client gone)")
	}
}
