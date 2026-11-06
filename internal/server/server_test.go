package server

import (
	"testing"

	"crossdesk/internal/capture"
	"crossdesk/internal/proto"
)

type fakePF struct {
	frozen       bool
	warpX, warpY float64
}

func newTestServer(t *testing.T, layout map[string]Cell, clients ...*client) (*Server, *fakePF) {
	f := &fakePF{}
	s := New(Config{SelfID: "host", Speed: 1}, Platform{
		Freeze: func(b bool) { f.frozen = b },
		Warp:   func(x, y float64) { f.warpX, f.warpY = x, y },
		Bounds: func() capture.Bounds { return capture.Bounds{MaxX: 1440, MaxY: 900} },
	})
	s.SetLayout(layout)
	s.active = true
	for _, c := range clients {
		s.clients[c.id] = c
	}
	return s, f
}

func newClient(id string) *client {
	return &client{id: id, name: id, out: make(chan proto.Msg, 100), w: 1920, h: 1080}
}

func drain(c chan proto.Msg) []proto.Msg {
	var ms []proto.Msg
	for {
		select {
		case m := <-c:
			ms = append(ms, m)
		default:
			return ms
		}
	}
}

func TestEnterAndLeaveRight(t *testing.T) {
	win := newClient("win")
	s, f := newTestServer(t, map[string]Cell{"host": {0, 0}, "win": {1, 0}}, win)

	if s.Handle(capture.Event{Kind: capture.Move, X: 700, Y: 450, DX: 5}) {
		t.Fatal("should not swallow local move")
	}
	if !s.Handle(capture.Event{Kind: capture.Move, X: 1439.5, Y: 450, DX: 3}) {
		t.Fatal("edge move should be swallowed")
	}
	if s.cur != win || !f.frozen {
		t.Fatal("expected remote + frozen cursor")
	}
	ms := drain(win.out)
	if ms[0].Type != proto.Enter || ms[0].X != 0 || ms[0].Y != 539 {
		t.Fatalf("bad enter %+v", ms[0])
	}
	s.Handle(capture.Event{Kind: capture.Key, Key: 0x04, Down: true})
	if m := drain(win.out); len(m) != 1 || m[0].Type != proto.Key {
		t.Fatalf("expected key msg, got %+v", m)
	}
	s.Handle(capture.Event{Kind: capture.Move, DX: 100})
	s.Handle(capture.Event{Kind: capture.Move, DX: -150})
	if s.cur != nil || f.frozen {
		t.Fatal("expected back home")
	}
	if f.warpX != 1438 {
		t.Fatalf("warp x = %v", f.warpX)
	}
	if ms := drain(win.out); ms[len(ms)-1].Type != proto.Leave {
		t.Fatalf("expected Leave, got %+v", ms)
	}
}

func TestNoNeighborNoSwitch(t *testing.T) {
	win := newClient("win")
	s, _ := newTestServer(t, map[string]Cell{"host": {0, 0}, "win": {1, 0}}, win)
	if s.Handle(capture.Event{Kind: capture.Move, X: 0, Y: 450, DX: -3}) {
		t.Fatal("nothing on the left: must stay local")
	}
}

func TestClientAbove(t *testing.T) {
	win := newClient("win")
	s, f := newTestServer(t, map[string]Cell{"host": {0, 1}, "win": {0, 0}}, win)
	s.Handle(capture.Event{Kind: capture.Move, X: 720, Y: 0, DY: -2})
	if s.cur != win {
		t.Fatal("expected switch upward")
	}
	if m := drain(win.out)[0]; m.Y != 1079 || m.X != 959 {
		t.Fatalf("enter at %d,%d", m.X, m.Y)
	}
	s.Handle(capture.Event{Kind: capture.Move, DY: 5})
	if s.cur != nil || f.warpY != 1 {
		t.Fatalf("expected home at top edge, warpY=%v", f.warpY)
	}
}

func TestHopBetweenClients(t *testing.T) {
	a, b := newClient("a"), newClient("b")
	s, _ := newTestServer(t, map[string]Cell{"host": {0, 0}, "a": {1, 0}, "b": {2, 0}}, a, b)
	s.Handle(capture.Event{Kind: capture.Move, X: 1439.5, Y: 0, DX: 1})
	s.Handle(capture.Event{Kind: capture.Move, DX: 2000})
	if s.cur != b {
		t.Fatal("expected hop to b")
	}
	if ms := drain(a.out); ms[len(ms)-1].Type != proto.Leave {
		t.Fatal("a should get Leave")
	}
	if ms := drain(b.out); ms[0].Type != proto.Enter || ms[0].X != 0 {
		t.Fatalf("b enter %+v", ms[0])
	}
}

func TestNoSwitchWhileDragging(t *testing.T) {
	win := newClient("win")
	s, _ := newTestServer(t, map[string]Cell{"host": {0, 0}, "win": {1, 0}}, win)
	s.Handle(capture.Event{Kind: capture.Button, Button: 0, Down: true})
	if s.Handle(capture.Event{Kind: capture.Move, X: 1439.5, DX: 3}) || s.cur != nil {
		t.Fatal("must not switch while a button is held")
	}
}

func TestLocalKeyReleasedLocally(t *testing.T) {
	win := newClient("win")
	s, _ := newTestServer(t, map[string]Cell{"host": {0, 0}, "win": {1, 0}}, win)
	s.Handle(capture.Event{Kind: capture.Key, Key: 0xE3, Down: true})
	s.Handle(capture.Event{Kind: capture.Move, X: 1439.5, DX: 3})
	drain(win.out)
	if s.Handle(capture.Event{Kind: capture.Key, Key: 0xE3, Down: false}) {
		t.Fatal("release of a locally pressed key must reach the local machine")
	}
	if len(drain(win.out)) != 0 {
		t.Fatal("should not forward that release")
	}
}

func TestInactivePassesEverything(t *testing.T) {
	win := newClient("win")
	s, _ := newTestServer(t, map[string]Cell{"host": {0, 0}, "win": {1, 0}}, win)
	s.active = false
	if s.Handle(capture.Event{Kind: capture.Move, X: 1439.5, DX: 3}) {
		t.Fatal("inactive server must not swallow")
	}
}
