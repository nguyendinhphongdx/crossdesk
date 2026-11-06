//go:build linux

package linuxinput

import (
	"os"
	"testing"
	"time"
)

// findDevice waits for a uinput device to show up as /dev/input/event*.
func findDevice(t *testing.T, name string) *Physical {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, path := range EventPaths() {
			if p, err := openRaw(path); err == nil {
				if p.Name == name {
					return p
				}
				p.Close()
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("device %q never appeared", name)
	return nil
}

func TestUinputRoundTrip(t *testing.T) {
	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY, 0)
	if err != nil {
		t.Skip("cannot open /dev/uinput (needs root or the input group):", err)
	}
	f.Close()
	kbd, err := NewKeyboard([]uint16{KeyA, 29})
	if err != nil {
		t.Fatal(err)
	}
	defer kbd.Close()
	ptr, err := NewPointer(1920, 1080)
	if err != nil {
		t.Fatal(err)
	}
	defer ptr.Close()

	pk := findDevice(t, DevicePrefix+" keyboard")
	defer pk.Close()
	if pk.Kind&Keyboard == 0 {
		t.Fatalf("keyboard classified as %v", pk.Kind)
	}
	pp := findDevice(t, DevicePrefix+" pointer")
	defer pp.Close()

	// Our own devices must be ignored by the capturer.
	if _, err := OpenPhysical(pk.Path); err == nil {
		t.Fatal("OpenPhysical must skip share-device's own devices")
	}

	if err := kbd.Emit([3]int32{EvKey, KeyA, 1}); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 24*16)
	evs, err := pk.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) < 2 || evs[0] != (Event{EvKey, KeyA, 1}) || evs[len(evs)-1].Type != EvSyn {
		t.Fatalf("keyboard events: %+v", evs)
	}

	if err := ptr.Emit([3]int32{EvAbs, AbsX, 960}, [3]int32{EvAbs, AbsY, 540}, [3]int32{EvRel, RelWheelHiRes, -120}); err != nil {
		t.Fatal(err)
	}
	evs, err = pp.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	want := []Event{{EvAbs, AbsX, 960}, {EvAbs, AbsY, 540}, {EvRel, RelWheelHiRes, -120}}
	for i, w := range want {
		if i >= len(evs) || evs[i] != w {
			t.Fatalf("pointer events: %+v", evs)
		}
	}

	// Grab must succeed on a real evdev node.
	if err := pk.Grab(true); err != nil {
		t.Fatal("grab:", err)
	}
	pk.Grab(false)
}

func TestParseWxH(t *testing.T) {
	if w, h, ok := parseWxH("2560x1440"); !ok || w != 2560 || h != 1440 {
		t.Fatal(w, h, ok)
	}
	if _, _, ok := parseWxH("garbage"); ok {
		t.Fatal("should fail")
	}
}
