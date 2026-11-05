package server

import (
	"testing"

	"sharedevice/internal/capture"
	"sharedevice/internal/proto"
)

type fakePF struct {
	frozen bool
	warpX  float64
	warpY  float64
}

func newTestServer(t *testing.T, side string) (*Server, *fakePF, *client) {
	f := &fakePF{}
	s, err := New(Config{Side: side, Speed: 1}, Platform{
		Freeze: func(b bool) { f.frozen = b },
		Warp:   func(x, y float64) { f.warpX, f.warpY = x, y },
		Bounds: func() capture.Bounds { return capture.Bounds{MaxX: 1440, MaxY: 900} },
	})
	if err != nil {
		t.Fatal(err)
	}
	cl := &client{out: make(chan proto.Msg, 100), w: 1920, h: 1080, name: "win"}
	s.cl = cl
	return s, f, cl
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
	s, f, cl := newTestServer(t, "right")

	// Moving inside the screen stays local.
	if s.Handle(capture.Event{Kind: capture.Move, X: 700, Y: 450, DX: 5}) {
		t.Fatal("should not swallow local move")
	}
	// Hitting the right edge switches.
	if !s.Handle(capture.Event{Kind: capture.Move, X: 1439.5, Y: 450, DX: 3}) {
		t.Fatal("edge move should be swallowed")
	}
	if !s.remote || !f.frozen {
		t.Fatal("expected remote + frozen cursor")
	}
	ms := drain(cl.out)
	if ms[0].Type != proto.Enter || ms[0].X != 0 || ms[0].Y != 539 {
		t.Fatalf("bad enter %+v", ms[0])
	}
	// Keys go remote.
	s.Handle(capture.Event{Kind: capture.Key, Key: 0x04, Down: true})
	if m := drain(cl.out); len(m) != 1 || m[0].Type != proto.Key {
		t.Fatalf("expected key msg, got %+v", m)
	}
	// Move right then back left past the edge.
	s.Handle(capture.Event{Kind: capture.Move, DX: 100})
	s.Handle(capture.Event{Kind: capture.Move, DX: -150})
	if s.remote || f.frozen {
		t.Fatal("expected back to local")
	}
	if f.warpX != 1438 {
		t.Fatalf("warp x = %v", f.warpX)
	}
	ms = drain(cl.out)
	if ms[len(ms)-1].Type != proto.Leave {
		t.Fatalf("expected Leave, got %+v", ms)
	}
}

func TestNoSwitchWhileDragging(t *testing.T) {
	s, _, _ := newTestServer(t, "right")
	s.Handle(capture.Event{Kind: capture.Button, Button: 0, Down: true})
	if s.Handle(capture.Event{Kind: capture.Move, X: 1439.5, DX: 3}) || s.remote {
		t.Fatal("must not switch while a button is held")
	}
}

func TestLocalKeyReleasedLocally(t *testing.T) {
	s, _, cl := newTestServer(t, "right")
	s.Handle(capture.Event{Kind: capture.Key, Key: 0xE3, Down: true}) // Cmd held locally
	s.Handle(capture.Event{Kind: capture.Move, X: 1439.5, DX: 3})
	drain(cl.out)
	if s.Handle(capture.Event{Kind: capture.Key, Key: 0xE3, Down: false}) {
		t.Fatal("release of a locally pressed key must reach the local machine")
	}
	if len(drain(cl.out)) != 0 {
		t.Fatal("should not forward that release")
	}
}

func TestLeftSide(t *testing.T) {
	s, _, cl := newTestServer(t, "left")
	s.Handle(capture.Event{Kind: capture.Move, X: 0, Y: 0, DX: -2})
	if !s.remote {
		t.Fatal("expected switch on left edge")
	}
	if m := drain(cl.out)[0]; m.X != 1919 {
		t.Fatalf("enter x = %d", m.X)
	}
	s.Handle(capture.Event{Kind: capture.Move, DX: 5})
	if s.remote {
		t.Fatal("moving right past the client's right edge should return")
	}
}
