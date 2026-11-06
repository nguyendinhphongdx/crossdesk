package client

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"crossdesk/internal/capture"
	"crossdesk/internal/inject"
	"crossdesk/internal/keymap"
	"crossdesk/internal/server"
)

type recorder struct {
	mu  sync.Mutex
	log []string
}

func (r *recorder) add(s string)              { r.mu.Lock(); r.log = append(r.log, s); r.mu.Unlock() }
func (r *recorder) Screen() inject.Screen     { return inject.Screen{W: 1920, H: 1080} }
func (r *recorder) MoveTo(x, y int32)         { r.add(fmt.Sprintf("move %d,%d", x, y)) }
func (r *recorder) Button(b int, down bool)   { r.add(fmt.Sprintf("button %d %v", b, down)) }
func (r *recorder) Wheel(dx, dy int32)        { r.add(fmt.Sprintf("wheel %d,%d", dx, dy)) }
func (r *recorder) Key(hid uint16, down bool) { r.add(fmt.Sprintf("key %02X %v", hid, down)) }
func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.log...)
}

func startHost(t *testing.T, pw string) (*server.Server, string) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot listen:", err)
	}
	t.Cleanup(func() { ln.Close() })
	srv := server.New(server.Config{SelfID: "host", Password: pw, Speed: 1}, server.Platform{
		Freeze: func(bool) {}, Warp: func(float64, float64) {},
		Bounds: func() capture.Bounds { return capture.Bounds{MaxX: 1440, MaxY: 900} },
	})
	srv.SetLayout(map[string]server.Cell{"host": {X: 0, Y: 0}, "pc": {X: 1, Y: 0}})
	srv.SetActive(true)
	go srv.Serve(ln)
	return srv, ln.Addr().String()
}

func TestEndToEnd(t *testing.T) {
	srv, addr := startHost(t, "pw")
	rec := &recorder{}
	c := Start(Config{Server: addr, Password: "pw", ID: "pc", Name: "test", SwapCmdCtrl: true}, rec)
	defer c.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for !srv.Handle(capture.Event{Kind: capture.Move, X: 1439.5, Y: 450, DX: 2}) {
		if time.Now().After(deadline) {
			t.Fatal("client never connected")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !c.Connected() {
		t.Fatal("client should report connected")
	}
	srv.Handle(capture.Event{Kind: capture.Move, DX: 10, DY: 5})
	srv.Handle(capture.Event{Kind: capture.Key, Key: keymap.HIDLeftGUI, Down: true})
	srv.Handle(capture.Event{Kind: capture.Key, Key: 0x06, Down: true})
	srv.Handle(capture.Event{Kind: capture.Key, Key: 0x06, Down: false})
	srv.Handle(capture.Event{Kind: capture.Button, Button: 0, Down: true})
	srv.Handle(capture.Event{Kind: capture.Button, Button: 0, Down: false})
	srv.Handle(capture.Event{Kind: capture.Scroll, ScrollY: -120})
	srv.Handle(capture.Event{Kind: capture.Move, DX: -50})

	want := []string{
		"move 0,539", "move 0,539", "move 10,544",
		"key E0 true", "key 06 true", "key 06 false",
		"button 0 true", "button 0 false",
		"wheel 0,-120",
		"key E0 false",
	}
	deadline = time.Now().Add(3 * time.Second)
	for fmt.Sprint(rec.snapshot()) != fmt.Sprint(want) {
		if time.Now().After(deadline) {
			t.Fatalf("\n got  %v\n want %v", rec.snapshot(), want)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Host steps down: client must notice and disconnect.
	srv.SetActive(false)
	deadline = time.Now().Add(3 * time.Second)
	for c.Connected() {
		if time.Now().After(deadline) {
			t.Fatal("client still connected after host stepped down")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWrongPassword(t *testing.T) {
	_, addr := startHost(t, "right")
	c := Start(Config{Server: addr, Password: "wrong", ID: "pc"}, &recorder{})
	defer c.Stop()
	deadline := time.Now().Add(5 * time.Second)
	for c.LastError() == "" {
		if time.Now().After(deadline) {
			t.Fatal("no error reported")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if c.Connected() {
		t.Fatal("must not connect")
	}
	t.Log("error:", c.LastError())
}
