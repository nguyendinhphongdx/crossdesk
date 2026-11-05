package agent

import (
	"testing"

	"sharedevice/internal/server"
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
