package inject

import (
	"syscall"
	"unsafe"

	"crossdesk/internal/keymap"
)

var (
	user32                        = syscall.NewLazyDLL("user32.dll")
	procSendInput                 = user32.NewProc("SendInput")
	procGetSystemMetrics          = user32.NewProc("GetSystemMetrics")
	procSetProcessDPIAware        = user32.NewProc("SetProcessDPIAware")
	procSetProcessDpiAwarenessCtx = user32.NewProc("SetProcessDpiAwarenessContext")
)

const (
	inputMouse    = 0
	inputKeyboard = 1

	mouseMove        = 0x0001
	mouseLeftDown    = 0x0002
	mouseLeftUp      = 0x0004
	mouseRightDown   = 0x0008
	mouseRightUp     = 0x0010
	mouseMiddleDown  = 0x0020
	mouseMiddleUp    = 0x0040
	mouseXDown       = 0x0080
	mouseXUp         = 0x0100
	mouseWheel       = 0x0800
	mouseHWheel      = 0x1000
	mouseVirtualDesk = 0x4000
	mouseAbsolute    = 0x8000

	keyExtended = 0x0001
	keyUp       = 0x0002
	keyScancode = 0x0008

	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCXVirtualScreen = 78
	smCYVirtualScreen = 79
)

type mouseInput struct {
	Dx, Dy    int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type keybdInput struct {
	Vk, Scan  uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

// INPUT is a C union; mouseInput is its largest member, so mInput has the real size.
type mInput struct {
	Type uint32
	Mi   mouseInput
}

type kInput struct {
	Type uint32
	Ki   keybdInput
	_    [unsafe.Sizeof(mouseInput{}) - unsafe.Sizeof(keybdInput{})]byte
}

type windows struct{ scr Screen }

func New() (Injector, error) {
	// Per-monitor DPI awareness so coordinates are real pixels, not scaled ones.
	if procSetProcessDpiAwarenessCtx.Find() == nil {
		procSetProcessDpiAwarenessCtx.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
	} else {
		procSetProcessDPIAware.Call()
	}
	w := &windows{}
	w.scr = w.readScreen()
	return w, nil
}

func metric(i int) int32 {
	r, _, _ := procGetSystemMetrics.Call(uintptr(i))
	return int32(r)
}

func (w *windows) readScreen() Screen {
	return Screen{X: metric(smXVirtualScreen), Y: metric(smYVirtualScreen), W: metric(smCXVirtualScreen), H: metric(smCYVirtualScreen)}
}

func (w *windows) Screen() Screen {
	w.scr = w.readScreen()
	return w.scr
}

func sendMouse(mi mouseInput) {
	in := mInput{Type: inputMouse, Mi: mi}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func sendKey(ki keybdInput) {
	in := kInput{Type: inputKeyboard, Ki: ki}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func (w *windows) MoveTo(x, y int32) {
	s := w.scr
	if s.W <= 1 || s.H <= 1 {
		return
	}
	// Absolute coordinates are normalized to 0..65535 across the virtual desktop.
	nx := (int64(x)*65535 + int64(s.W-1)/2) / int64(s.W-1)
	ny := (int64(y)*65535 + int64(s.H-1)/2) / int64(s.H-1)
	sendMouse(mouseInput{Dx: int32(nx), Dy: int32(ny), Flags: mouseMove | mouseAbsolute | mouseVirtualDesk})
}

func (w *windows) Button(b int, down bool) {
	var mi mouseInput
	switch b {
	case 0:
		mi.Flags = pick(down, mouseLeftDown, mouseLeftUp)
	case 1:
		mi.Flags = pick(down, mouseRightDown, mouseRightUp)
	case 2:
		mi.Flags = pick(down, mouseMiddleDown, mouseMiddleUp)
	case 3, 4:
		mi.Flags = pick(down, mouseXDown, mouseXUp)
		mi.MouseData = uint32(b - 2) // XBUTTON1 = back, XBUTTON2 = forward
	default:
		return
	}
	sendMouse(mi)
}

func pick(c bool, a, b uint32) uint32 {
	if c {
		return a
	}
	return b
}

func (w *windows) Wheel(dx, dy int32) {
	if dy != 0 {
		sendMouse(mouseInput{Flags: mouseWheel, MouseData: uint32(dy)})
	}
	if dx != 0 {
		sendMouse(mouseInput{Flags: mouseHWheel, MouseData: uint32(dx)})
	}
}

// Media keys don't work as scancodes; send virtual-key codes instead.
var hidToVK = map[uint16]uint16{
	keymap.HIDMute:    0xAD,
	keymap.HIDVolDown: 0xAE,
	keymap.HIDVolUp:   0xAF,
}

func (w *windows) Key(hid uint16, down bool) {
	if vk, ok := hidToVK[hid]; ok {
		ki := keybdInput{Vk: vk, Flags: keyExtended}
		if !down {
			ki.Flags |= keyUp
		}
		sendKey(ki)
		return
	}
	sc, ok := keymap.HIDToWin(hid)
	if !ok {
		return
	}
	ki := keybdInput{Scan: sc & 0xFF, Flags: keyScancode}
	if sc&0xFF00 == 0xE000 {
		ki.Flags |= keyExtended
	}
	if !down {
		ki.Flags |= keyUp
	}
	sendKey(ki)
}
