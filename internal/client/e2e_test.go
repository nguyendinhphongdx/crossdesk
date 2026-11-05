package client

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"sharedevice/internal/capture"
	"sharedevice/internal/inject"
	"sharedevice/internal/keymap"
	"sharedevice/internal/server"
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

func TestEndToEnd(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot listen:", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	srv, err := server.New(server.Config{Listen: addr, Password: "pw", Side: "right", Speed: 1}, server.Platform{
		Freeze: func(bool) {}, Warp: func(float64, float64) {},
		Bounds: func() capture.Bounds { return capture.Bounds{MaxX: 1440, MaxY: 900} },
	})
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve()

	rec := &recorder{}
	go Run(Config{Server: addr, Password: "pw", Name: "test", SwapCmdCtrl: true}, rec)

	// Wait until the client is connected: an edge move only switches with a client.
	deadline := time.Now().Add(5 * time.Second)
	for !srv.Handle(capture.Event{Kind: capture.Move, X: 1439.5, Y: 450, DX: 2}) {
		if time.Now().After(deadline) {
			t.Fatal("client never connected")
		}
		time.Sleep(50 * time.Millisecond)
	}
	srv.Handle(capture.Event{Kind: capture.Move, DX: 10, DY: 5})
	srv.Handle(capture.Event{Kind: capture.Key, Key: keymap.HIDLeftGUI, Down: true}) // Cmd
	srv.Handle(capture.Event{Kind: capture.Key, Key: 0x06, Down: true})              // C
	srv.Handle(capture.Event{Kind: capture.Key, Key: 0x06, Down: false})
	srv.Handle(capture.Event{Kind: capture.Button, Button: 0, Down: true})
	srv.Handle(capture.Event{Kind: capture.Button, Button: 0, Down: false})
	srv.Handle(capture.Event{Kind: capture.Scroll, ScrollY: -120})
	srv.Handle(capture.Event{Kind: capture.Move, DX: -50}) // back across the edge -> Leave

	want := []string{
		"move 0,539", "move 0,539", "move 10,544",
		"key E0 true", // Cmd swapped to Ctrl
		"key 06 true", "key 06 false",
		"button 0 true", "button 0 false",
		"wheel 0,-120",
		"key E0 false", // released on Leave because Cmd was still held
	}
	deadline = time.Now().Add(3 * time.Second)
	for {
		got := rec.snapshot()
		if fmt.Sprint(got) == fmt.Sprint(want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("\n got  %v\n want %v", got, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWrongPasswordStops(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot listen:", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	srv, _ := server.New(server.Config{Listen: addr, Password: "right", Side: "right"}, server.Platform{})
	go srv.Serve()
	time.Sleep(100 * time.Millisecond)
	done := make(chan error)
	go func() { done <- runOnce(Config{Server: addr, Password: "wrong"}, &recorder{}) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error")
		}
		t.Log("client error:", err)
	case <-time.After(5 * time.Second):
		t.Fatal("client hung with wrong password")
	}
}
