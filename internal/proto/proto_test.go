package proto

import (
	"net"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	m := Msg{Type: Key, X: -5, Y: 1080, W: 1920, H: 1080, Button: 3, Down: true, Key: 0xE3, Name: "win-pc"}
	got, err := Unmarshal(m.Marshal())
	if err != nil || got != m {
		t.Fatalf("got %+v err %v", got, err)
	}
}

func pipe(t *testing.T, serverPw, clientPw string) (*Conn, *Conn) {
	a, b := net.Pipe()
	type res struct {
		c   *Conn
		err error
	}
	ch := make(chan res)
	go func() { c, err := ServerHandshake(a, serverPw); ch <- res{c, err} }()
	cc, err := ClientHandshake(b, clientPw)
	if err != nil {
		t.Fatal(err)
	}
	r := <-ch
	if r.err != nil {
		t.Fatal(r.err)
	}
	return r.c, cc
}

func TestSecureConn(t *testing.T) {
	sc, cc := pipe(t, "pw", "pw")
	go func() {
		cc.Send(Msg{Type: Hello, W: 100, H: 50, Name: "x"})
		cc.Send(Msg{Type: Ping})
	}()
	for _, want := range []Type{Hello, Ping} {
		m, err := sc.Recv(0)
		if err != nil || m.Type != want {
			t.Fatalf("want %v got %+v err %v", want, m, err)
		}
	}
}

func TestWrongPassword(t *testing.T) {
	sc, cc := pipe(t, "right", "wrong")
	go cc.Send(Msg{Type: Hello})
	if _, err := sc.Recv(0); err != ErrAuth {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestClipRoundTripThroughConn(t *testing.T) {
	sc, cc := pipe(t, "pw", "pw")
	data := make([]byte, 3*ClipChunkSize+123)
	for i := range data {
		data[i] = byte(i * 7)
	}
	go func() {
		for _, m := range ClipMessages(2, data) {
			cc.Send(m)
		}
	}()
	var a ClipAssembler
	for {
		m, err := sc.Recv(0)
		if err != nil {
			t.Fatal(err)
		}
		kind, got, done, err := a.Feed(m)
		if err != nil {
			t.Fatal(err)
		}
		if done {
			if kind != 2 || string(got) != string(data) {
				t.Fatal("clipboard data corrupted")
			}
			return
		}
	}
}

func TestClipAssemblerRejectsBadInput(t *testing.T) {
	var a ClipAssembler
	if _, _, _, err := a.Feed(Msg{Type: ClipBegin, X: MaxClip + 1}); err == nil {
		t.Fatal("oversized transfer must be rejected")
	}
	a.Feed(Msg{Type: ClipBegin, Button: 1, X: 3})
	if _, _, _, err := a.Feed(Msg{Type: ClipChunk, Name: "toolong"}); err == nil {
		t.Fatal("overflowing chunk must be rejected")
	}
	if _, _, _, err := a.Feed(Msg{Type: ClipEnd}); err == nil {
		t.Fatal("end without begin must be rejected")
	}
}
