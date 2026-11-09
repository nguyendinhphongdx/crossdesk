package agent

import (
	"net"
	"strconv"
	"testing"
	"time"

	"crossdesk/internal/capture"
	"crossdesk/internal/server"
)

// testAgent is an Agent without UDP, capture or disk side effects.
func testAgent(t *testing.T, id, pw string) *Agent {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := &Agent{cfg: &Config{ID: id, Name: id, Password: pw, Shared: Shared{Layout: map[string]server.Cell{}}},
		key: groupKey(pw), peers: map[string]*peer{}, inj: fakeInj{}, changed: make(chan struct{}, 1)}
	a.captureRunning = true // never start the real input hook in tests
	a.srv = server.New(server.Config{SelfID: id, Password: pw, Speed: 1}, server.Platform{
		Freeze: func(bool) {}, Warp: func(float64, float64) {},
		Bounds: func() capture.Bounds { return capture.Bounds{MaxX: 1440, MaxY: 900} },
	})
	return a
}

func listen(t *testing.T) net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("cannot listen:", err)
	}
	t.Cleanup(func() { ln.Close() })
	return ln
}

func port(ln net.Listener) int { return ln.Addr().(*net.TCPAddr).Port }

// tick runs what Agent.Run does each second, but fast.
func tick(t *testing.T, as ...*Agent) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	for _, a := range as {
		go func(a *Agent) {
			for {
				select {
				case <-stop:
					return
				case <-time.After(100 * time.Millisecond):
					a.reconcile()
					a.sendBeacon()
				}
			}
		}(a)
	}
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out: " + what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func hasConnected(a *Agent, id string) bool {
	for _, c := range a.srv.Connected() {
		if c == id {
			return true
		}
	}
	return false
}

// pc can only dial out (like a company PC whose firewall blocks incoming
// connections); mac accepts. pc opens a manual link to mac.
func TestManualLinkOneWayNetwork(t *testing.T) {
	pc := testAgent(t, "pc", "pw")
	mac := testAgent(t, "mac", "pw")

	// mac accepts links and input sessions; pc accepts nothing at all.
	macLinks, macInput := listen(t), listen(t)
	mac.initLinks(port(macLinks))
	mac.inputPort = port(macInput)
	go mac.listenLinks(macLinks)
	go mac.srv.Serve(macInput)
	pc.initLinks(1) // pc never listens; port unused
	pc.cfg.ManualPeers = []string{"127.0.0.1:" + strconv.Itoa(port(macLinks))}
	go pc.dialLinks()
	tick(t, pc, mac)

	eventually(t, "both sides see each other over the link", func() bool {
		pc.mu.Lock()
		p := pc.peers["mac"]
		pc.mu.Unlock()
		mac.mu.Lock()
		m := mac.peers["pc"]
		mac.mu.Unlock()
		return p != nil && p.Trusted && m != nil && m.Trusted
	})
	if st := pc.Status(); len(st.Links) != 1 || st.Links[0].State != "connected" || st.Links[0].Peer != "mac" {
		t.Fatalf("link status: %+v", st.Links)
	}

	// 1) The PC is host: mac cannot dial it, so the PC opens the session.
	if err := pc.SetHost("pc"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "mac controlled by pc over a reverse session", func() bool { return hasConnected(pc, "mac") })
	mac.mu.Lock()
	dialed := mac.cli != nil
	mac.mu.Unlock()
	if dialed {
		t.Fatal("mac must not try to dial a host it can only reach through the host's link")
	}
	eventually(t, "mac reports connected", func() bool { return mac.Status().Self.Connected })

	// 2) The Mac is host: the PC dials the Mac's input port as usual.
	if err := mac.SetHost("mac"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "pc controlled by mac", func() bool { return hasConnected(mac, "pc") })
	eventually(t, "pc reports connected", func() bool { return pc.Status().Self.Connected })
}

func TestManualLinkWrongPassword(t *testing.T) {
	pc := testAgent(t, "pc", "right")
	mac := testAgent(t, "mac", "other")
	macLinks := listen(t)
	mac.initLinks(port(macLinks))
	go mac.listenLinks(macLinks)
	pc.initLinks(1)
	addr := "127.0.0.1:" + strconv.Itoa(port(macLinks))
	pc.cfg.ManualPeers = []string{addr}
	go pc.dialLinks()
	tick(t, pc, mac)
	eventually(t, "wrong password reported", func() bool {
		st := pc.Status()
		return len(st.Links) == 1 && st.Links[0].State == "error"
	})
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if p := pc.peers["mac"]; p != nil && p.Trusted {
		t.Fatal("must not trust a peer with another password")
	}
}

func TestNormalizeAddr(t *testing.T) {
	for in, want := range map[string]string{
		"192.168.0.88":       "192.168.0.88:24803",
		" 192.168.0.88:9999": "192.168.0.88:9999",
		"mac.local":          "mac.local:24803",
	} {
		if got, err := normalizeAddr(in); err != nil || got != want {
			t.Errorf("%q -> %q, %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "http://x", "1.2.3.4:0", "a b"} {
		if _, err := normalizeAddr(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
