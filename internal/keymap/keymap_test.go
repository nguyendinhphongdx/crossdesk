package keymap

import "testing"

func TestMacHIDBijective(t *testing.T) {
	if len(hidToMac) != len(macToHID) {
		t.Fatalf("duplicate HID targets: %d mac codes map to %d HID codes", len(macToHID), len(hidToMac))
	}
}

func TestEveryMacKeyHasWindowsScancode(t *testing.T) {
	for mac, hid := range macToHID {
		if hid == HIDMute || hid == HIDVolUp || hid == HIDVolDown {
			continue // sent as virtual keys on Windows
		}
		if _, ok := hidToWin[hid]; !ok {
			t.Errorf("mac 0x%02X -> hid 0x%02X has no Windows scancode", mac, hid)
		}
	}
}

func TestWindowsRoundTrip(t *testing.T) {
	for hid, sc := range hidToWin {
		back, ok := WinToHID(sc)
		if !ok {
			t.Fatalf("scancode 0x%04X has no HID", sc)
		}
		if back != hid && !(hid == 0x32 && back == 0x31) {
			t.Errorf("hid 0x%02X -> sc 0x%04X -> hid 0x%02X", hid, sc, back)
		}
	}
	// NumLock reported with the extended flag by low-level hooks.
	if h, _ := WinToHID(0xE045); h != 0x53 {
		t.Errorf("extended NumLock fallback: got 0x%02X", h)
	}
}
