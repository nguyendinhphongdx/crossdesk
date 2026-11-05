package capture

import (
	"errors"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"sharedevice/internal/keymap"
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	pSetWindowsHookEx   = user32.NewProc("SetWindowsHookExW")
	pCallNextHookEx     = user32.NewProc("CallNextHookEx")
	pGetMessage         = user32.NewProc("GetMessageW")
	pGetCursorPos       = user32.NewProc("GetCursorPos")
	pSetCursorPos       = user32.NewProc("SetCursorPos")
	pGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	pSetDPIAware        = user32.NewProc("SetProcessDPIAware")
	pSetDpiAwarenessCtx = user32.NewProc("SetProcessDpiAwarenessContext")
	pGetModuleHandle    = kernel32.NewProc("GetModuleHandleW")
)

const (
	whKeyboardLL = 13
	whMouseLL    = 14

	wmMouseMove   = 0x200
	wmLButtonDown = 0x201
	wmLButtonUp   = 0x202
	wmRButtonDown = 0x204
	wmRButtonUp   = 0x205
	wmMButtonDown = 0x207
	wmMButtonUp   = 0x208
	wmMouseWheel  = 0x20A
	wmXButtonDown = 0x20B
	wmXButtonUp   = 0x20C
	wmMouseHWheel = 0x20E

	wmKeyDown    = 0x100
	wmKeyUp      = 0x101
	wmSysKeyDown = 0x104
	wmSysKeyUp   = 0x105

	llmhfInjected = 0x01
	llkhfExtended = 0x01
	llkhfInjected = 0x10
)

type point struct{ X, Y int32 }

type msllHook struct {
	Pt        point
	MouseData uint32
	Flags     uint32
	Time      uint32
	Extra     uintptr
}

type kbdllHook struct {
	Vk, Scan, Flags, Time uint32
	Extra                 uintptr
}

var (
	handler Handler
	mu      sync.Mutex
	frozen  bool
	last    point // previous cursor position (the anchor while frozen)
)

func init() {
	// Real pixels, not DPI-scaled ones.
	if pSetDpiAwarenessCtx.Find() == nil {
		pSetDpiAwarenessCtx.Call(^uintptr(3)) // PER_MONITOR_AWARE_V2 (-4)
	} else {
		pSetDPIAware.Call()
	}
}

func Trusted(bool) bool { return true }

func cursorPos() point {
	var p point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p
}

// Run installs low-level mouse and keyboard hooks and pumps messages forever.
func Run(h Handler) error {
	handler = h
	runtime.LockOSThread()
	mu.Lock()
	last = cursorPos()
	mu.Unlock()
	mod, _, _ := pGetModuleHandle.Call(0)
	mh, _, err1 := pSetWindowsHookEx.Call(whMouseLL, syscall.NewCallback(mouseProc), mod, 0)
	kh, _, err2 := pSetWindowsHookEx.Call(whKeyboardLL, syscall.NewCallback(keyboardProc), mod, 0)
	if mh == 0 || kh == 0 {
		return errors.New("SetWindowsHookEx failed: " + errors.Join(err1, err2).Error())
	}
	var msg [64]byte
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&msg[0])), 0, 0, 0)
		if int32(r) <= 0 {
			return errors.New("message loop ended")
		}
	}
}

func next(nCode, wParam, lParam uintptr) uintptr {
	r, _, _ := pCallNextHookEx.Call(0, nCode, wParam, lParam)
	return r
}

func mouseProc(nCode, wParam, lParam uintptr) uintptr {
	if int32(nCode) == 0 {
		info := *(**msllHook)(unsafe.Pointer(&lParam))
		if info.Flags&llmhfInjected == 0 && handleMouse(wParam, info) {
			return 1
		}
	}
	return next(nCode, wParam, lParam)
}

func handleMouse(msg uintptr, info *msllHook) bool {
	ev := Event{X: float64(info.Pt.X), Y: float64(info.Pt.Y)}
	switch msg {
	case wmMouseMove:
		mu.Lock()
		ev.Kind = Move
		ev.DX, ev.DY = float64(info.Pt.X-last.X), float64(info.Pt.Y-last.Y)
		if !frozen {
			last = info.Pt
		}
		mu.Unlock()
		if ev.DX == 0 && ev.DY == 0 {
			return false
		}
	case wmLButtonDown, wmLButtonUp:
		ev.Kind, ev.Button, ev.Down = Button, 0, msg == wmLButtonDown
	case wmRButtonDown, wmRButtonUp:
		ev.Kind, ev.Button, ev.Down = Button, 1, msg == wmRButtonDown
	case wmMButtonDown, wmMButtonUp:
		ev.Kind, ev.Button, ev.Down = Button, 2, msg == wmMButtonDown
	case wmXButtonDown, wmXButtonUp:
		ev.Kind, ev.Down = Button, msg == wmXButtonDown
		ev.Button = 2 + int(info.MouseData>>16) // XBUTTON1 -> 3 (back), XBUTTON2 -> 4
	case wmMouseWheel:
		ev.Kind, ev.ScrollY = Scroll, int32(int16(info.MouseData>>16))
	case wmMouseHWheel:
		ev.Kind, ev.ScrollX = Scroll, int32(int16(info.MouseData>>16))
	default:
		return false
	}
	return handler(ev)
}

var vkToHID = map[uint32]uint16{0xAD: keymap.HIDMute, 0xAE: keymap.HIDVolDown, 0xAF: keymap.HIDVolUp}

func keyboardProc(nCode, wParam, lParam uintptr) uintptr {
	if int32(nCode) == 0 {
		info := *(**kbdllHook)(unsafe.Pointer(&lParam))
		if info.Flags&llkhfInjected == 0 {
			hid, ok := vkToHID[info.Vk]
			if !ok {
				sc := uint16(info.Scan)
				if info.Flags&llkhfExtended != 0 {
					sc |= 0xE000
				}
				hid, ok = keymap.WinToHID(sc)
			}
			if ok {
				down := wParam == wmKeyDown || wParam == wmSysKeyDown
				if handler(Event{Kind: Key, Key: hid, Down: down}) {
					return 1
				}
			}
		}
	}
	return next(nCode, wParam, lParam)
}

// Freeze pins the cursor: while frozen, moves are swallowed so the cursor stays
// where it is, and deltas are measured against that anchor.
func Freeze(f bool) {
	mu.Lock()
	defer mu.Unlock()
	frozen = f
	last = cursorPos()
}

func Warp(x, y float64) {
	pSetCursorPos.Call(uintptr(int32(x)), uintptr(int32(y)))
	mu.Lock()
	last = point{int32(x), int32(y)}
	mu.Unlock()
}

func metric(i uintptr) float64 {
	r, _, _ := pGetSystemMetrics.Call(i)
	return float64(int32(r))
}

func ScreenBounds() Bounds {
	x, y := metric(76), metric(77)
	return Bounds{MinX: x, MinY: y, MaxX: x + metric(78), MaxY: y + metric(79)}
}
