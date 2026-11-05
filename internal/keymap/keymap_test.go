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
