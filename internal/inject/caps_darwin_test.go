package inject

import (
	"os"
	"testing"
)

// Toggles the real Caps Lock twice (restoring it), so it only runs when asked:
// CROSSDESK_REAL_CAPS=1.
func TestCapsLockToggle(t *testing.T) {
	if os.Getenv("CROSSDESK_REAL_CAPS") != "1" {
		t.Skip("set CROSSDESK_REAL_CAPS=1 to toggle the real Caps Lock")
	}
	inj, _ := New()
	d := inj.(*darwin)
	before := d.caps
	d.Key(0x39, true)
	d.Key(0x39, false)
	if d.caps == before {
		t.Fatal("Caps Lock did not toggle")
	}
	if d.flags()&flagAlphaShift == 0 == d.caps {
		t.Fatal("event flags must follow Caps Lock")
	}
	d.Key(0x39, true) // restore
	d.Key(0x39, false)
	if d.caps != before {
		t.Fatal("Caps Lock not restored")
	}
}
