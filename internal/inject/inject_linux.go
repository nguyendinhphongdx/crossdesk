package inject

import (
	"sync"

	"sharedevice/internal/keymap"
	li "sharedevice/internal/linuxinput"
)

// linux injects through two uinput devices: a keyboard and an absolute
// pointer, which X11 and Wayland compositors both accept.
type linux struct {
	mu      sync.Mutex
	kbd     *li.Device
	ptr     *li.Device
	w, h    int32
	initErr error
}

func New() (Injector, error) {
	l := &linux{}
	l.w, l.h = li.ScreenSize()
	return l, nil
}

// devices are created lazily so the app starts even without /dev/uinput
// access (the UI then explains how to grant it).
func (l *linux) ensure() bool {
	if l.kbd != nil && l.ptr != nil {
		return true
	}
	var err error
	if l.kbd == nil {
		if l.kbd, err = li.NewKeyboard(keymap.LinuxKeyCodes()); err != nil {
			l.initErr = err
			return false
		}
	}
	if l.ptr == nil {
		if l.ptr, err = li.NewPointer(l.w, l.h); err != nil {
			l.initErr = err
			return false
		}
	}
	return true
}

func (l *linux) Screen() Screen {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, h := li.ScreenSize()
	if (w != l.w || h != l.h) && l.ptr != nil {
		l.ptr.Close() // the pointer's range is fixed at creation
		l.ptr = nil
	}
	l.w, l.h = w, h
	return Screen{W: w, H: h}
}

func (l *linux) MoveTo(x, y int32) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ensure() {
		l.ptr.Emit([3]int32{li.EvAbs, li.AbsX, x}, [3]int32{li.EvAbs, li.AbsY, y})
	}
}

var buttonCodes = [...]int32{li.BtnLeft, li.BtnRight, li.BtnMiddle, li.BtnSide, li.BtnExtra}

func (l *linux) Button(b int, down bool) {
	if b < 0 || b >= len(buttonCodes) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ensure() {
		l.ptr.Emit([3]int32{li.EvKey, buttonCodes[b], boolVal(down)})
	}
}

func (l *linux) Wheel(dx, dy int32) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.ensure() {
		return
	}
	var evs [][3]int32
	if dy != 0 {
		evs = append(evs, [3]int32{li.EvRel, li.RelWheelHiRes, dy})
		if dy/120 != 0 {
			evs = append(evs, [3]int32{li.EvRel, li.RelWheel, dy / 120})
		}
	}
	if dx != 0 {
		evs = append(evs, [3]int32{li.EvRel, li.RelHWheelHiRes, dx})
		if dx/120 != 0 {
			evs = append(evs, [3]int32{li.EvRel, li.RelHWheel, dx / 120})
		}
	}
	if len(evs) > 0 {
		l.ptr.Emit(evs...)
	}
}

func (l *linux) Key(hid uint16, down bool) {
	code, ok := keymap.HIDToLinux(hid)
	if !ok {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ensure() {
		l.kbd.Emit([3]int32{li.EvKey, int32(code), boolVal(down)})
	}
}

func boolVal(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
