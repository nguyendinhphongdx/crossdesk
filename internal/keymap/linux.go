package keymap

// Linux evdev KEY_* codes. For the classic PC keys (non-extended set-1
// scancodes below 0x59) the evdev code equals the scancode; everything else
// is listed explicitly.
var hidToLinuxExtra = map[uint16]uint16{
	0x58: 96,                                                         // KP Enter
	0xE4: 97,                                                         // Right Ctrl
	0x54: 98,                                                         // KP /
	0x46: 99,                                                         // PrintScreen (SysRq)
	0xE6: 100,                                                        // Right Alt
	0x4A: 102,                                                        // Home
	0x52: 103,                                                        // Up
	0x4B: 104,                                                        // PageUp
	0x50: 105,                                                        // Left
	0x4F: 106,                                                        // Right
	0x4D: 107,                                                        // End
	0x51: 108,                                                        // Down
	0x4E: 109,                                                        // PageDown
	0x49: 110,                                                        // Insert
	0x4C: 111,                                                        // Delete
	0x7F: 113,                                                        // Mute
	0x81: 114,                                                        // Volume down
	0x80: 115,                                                        // Volume up
	0x67: 117,                                                        // KP =
	0x48: 119,                                                        // Pause
	0x85: 121,                                                        // KP ,
	0x90: 122,                                                        // Lang1 (Hangeul / Kana)
	0x91: 123,                                                        // Lang2 (Hanja / Eisu)
	0x89: 124,                                                        // Yen
	0xE3: 125,                                                        // Left Meta
	0xE7: 126,                                                        // Right Meta
	0x65: 127,                                                        // Menu (Compose)
	0x87: 89,                                                         // Ro
	0x68: 183, 0x69: 184, 0x6A: 185, 0x6B: 186, 0x6C: 187, 0x6D: 188, // F13-F18
	0x6E: 189, 0x6F: 190, 0x70: 191, 0x71: 192, 0x72: 193, 0x73: 194, // F19-F24
}

func HIDToLinux(hid uint16) (uint16, bool) {
	if k, ok := hidToLinuxExtra[hid]; ok {
		return k, true
	}
	if sc, ok := hidToWin[hid]; ok && sc < 0x59 {
		return sc, true
	}
	return 0, false
}

var linuxToHID = func() map[uint16]uint16 {
	m := map[uint16]uint16{}
	for hid := uint16(0); hid < 0x100; hid++ {
		if k, ok := HIDToLinux(hid); ok {
			if _, dup := m[k]; !dup {
				m[k] = hid
			}
		}
	}
	return m
}()

func LinuxToHID(code uint16) (uint16, bool) { h, ok := linuxToHID[code]; return h, ok }

// LinuxKeyCodes lists every evdev key code the virtual keyboard may emit.
func LinuxKeyCodes() []uint16 {
	out := make([]uint16, 0, len(linuxToHID))
	for k := range linuxToHID {
		out = append(out, k)
	}
	return out
}
