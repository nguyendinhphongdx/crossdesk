package agent

import (
	"testing"
	"time"

	"crossdesk/internal/inject"
	"crossdesk/internal/server"
)

func TestBeaconSignVerify(t *testing.T) {
	k := groupKey("secret")
	b := Beacon{App: beaconApp, ID: "a", Name: "mac", Role: "host",
		Shared: &Shared{Epoch: 3, Host: "a", Layout: map[string]server.Cell{"a": {X: 0}, "b": {X: 1}}}}.sign(k)
	if !b.verify(k) {
		t.Fatal("signed beacon must verify")
	}
	if b.verify(groupKey("other")) {
		t.Fatal("must not verify with another password")
	}
	b.Shared.Host = "evil"
	if b.verify(k) {
		t.Fatal("tampered beacon must not verify")
	}
	if (Beacon{ID: "x"}).sign(nil).verify(nil) {
		t.Fatal("no key, no trust")
	}
}

func TestSharedOrdering(t *testing.T) {
	a := Shared{Epoch: 5, Author: "a"}
	if !(Shared{Epoch: 6, Author: "a"}).newerThan(a) || (Shared{Epoch: 4, Author: "z"}).newerThan(a) {
		t.Fatal("epoch ordering")
	}
	if !(Shared{Epoch: 5, Author: "b"}).newerThan(a) {
		t.Fatal("tie broken by author")
	}
}

func TestBroadcastAddrs(t *testing.T) {
	if len(broadcastAddrs(DiscoveryPort)) == 0 {
		t.Fatal("expected at least the limited broadcast address")
	}
}

func TestStaleDevicesHiddenAndForgotten(t *testing.T) {
	a := &Agent{cfg: &Config{ID: "me", Name: "mac", Shared: Shared{Layout: map[string]server.Cell{}}},
		peers: map[string]*peer{}, inj: fakeInj{}, srv: server.New(server.Config{SelfID: "me"}, server.Platform{})}
	now := time.Now()
	a.peers["live"] = &peer{Beacon: Beacon{Name: "pc"}, LastSeen: now}
	a.peers["recent"] = &peer{Beacon: Beacon{Name: "pc-old"}, LastSeen: now.Add(-30 * time.Second)}
	a.peers["stale"] = &peer{Beacon: Beacon{Name: "pc-gone"}, LastSeen: now.Add(-2 * time.Minute)}
	a.peers["placed"] = &peer{Beacon: Beacon{Name: "pc-placed"}, LastSeen: now.Add(-2 * time.Minute)}
	a.peers["ancient"] = &peer{Beacon: Beacon{Name: "pc-ancient"}, LastSeen: now.Add(-time.Hour)}
	a.cfg.Shared.Layout["placed"] = server.Cell{X: 1}

	names := map[string]bool{}
	for _, d := range a.Status().Devices {
		names[d.Name] = true
	}
	for _, want := range []string{"mac", "pc", "pc-old", "pc-placed"} {
		if !names[want] {
			t.Errorf("%s should be listed", want)
		}
	}
	if names["pc-gone"] || names["pc-ancient"] {
		t.Error("devices offline for over a minute should be hidden")
	}
	a.reconcile()
	if _, ok := a.peers["ancient"]; ok {
		t.Error("devices offline for 10+ minutes should be forgotten")
	}
	if _, ok := a.peers["stale"]; !ok {
		t.Error("recently offline devices are kept in memory")
	}
}

type fakeInj struct{}

func (fakeInj) Screen() inject.Screen { return inject.Screen{W: 100, H: 100} }
func (fakeInj) MoveTo(x, y int32)     {}
func (fakeInj) Button(int, bool)      {}
func (fakeInj) Wheel(int32, int32)    {}
func (fakeInj) Key(uint16, bool)      {}
