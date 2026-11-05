package inject

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>

static void mainBounds(double *x, double *y, double *w, double *h) {
	CGRect r = CGDisplayBounds(CGMainDisplayID());
	*x = r.origin.x; *y = r.origin.y; *w = r.size.width; *h = r.size.height;
}

static void postMouse(int type, double x, double y, int button, unsigned long long flags, int clicks) {
	CGEventRef e = CGEventCreateMouseEvent(NULL, (CGEventType)type, CGPointMake(x, y), (CGMouseButton)button);
	if (!e) return;
	if (clicks > 0) CGEventSetIntegerValueField(e, kCGMouseEventClickState, clicks);
	CGEventSetFlags(e, (CGEventFlags)flags);
	CGEventPost(kCGHIDEventTap, e);
	CFRelease(e);
}

static void postScroll(int dy, int dx) {
	CGEventRef e = CGEventCreateScrollWheelEvent(NULL, kCGScrollEventUnitPixel, 2, dy, dx);
	if (!e) return;
	CGEventPost(kCGHIDEventTap, e);
	CFRelease(e);
}

static void postKey(int keycode, int down, unsigned long long flags, int isModifier) {
	CGEventRef e = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keycode, down ? true : false);
	if (!e) return;
	if (isModifier) CGEventSetType(e, kCGEventFlagsChanged);
	CGEventSetFlags(e, (CGEventFlags)flags);
	CGEventPost(kCGHIDEventTap, e);
	CFRelease(e);
}
*/
import "C"

import (
	"time"

	"sharedevice/internal/keymap"
)

// Flag masks: device-independent | device-dependent (left/right) bits.
var modFlags = map[uint16]uint64{
	keymap.HIDLeftCtrl:   0x40000 | 0x0001,
	keymap.HIDLeftShift:  0x20000 | 0x0002,
	keymap.HIDRightShift: 0x20000 | 0x0004,
	keymap.HIDLeftGUI:    0x100000 | 0x0008,
	keymap.HIDRightGUI:   0x100000 | 0x0010,
	keymap.HIDLeftAlt:    0x80000 | 0x0020,
	keymap.HIDRightAlt:   0x80000 | 0x0040,
	keymap.HIDRightCtrl:  0x40000 | 0x2000,
}

type darwin struct {
	ox, oy    float64
	x, y      float64
	held      map[uint16]bool
	buttons   [5]bool
	lastClick time.Time
	lastBtn   int
	clicks    int
}

func New() (Injector, error) {
	d := &darwin{held: map[uint16]bool{}}
	d.Screen()
	return d, nil
}

func (d *darwin) Screen() Screen {
	var x, y, w, h C.double
	C.mainBounds(&x, &y, &w, &h)
	d.ox, d.oy = float64(x), float64(y)
	return Screen{X: int32(x), Y: int32(y), W: int32(w), H: int32(h)}
}

func (d *darwin) flags() uint64 {
	var f uint64 = 0x100 // NX_NONCOALSESCEDMASK-ish "non-coalesced" bit macOS sets on real events
	for k := range d.held {
		f |= modFlags[k]
	}
	return f
}

// CGEventType constants.
const (
	leftDown, leftUp, rightDown, rightUp, moved, leftDragged, rightDragged = 1, 2, 3, 4, 5, 6, 7
	otherDown, otherUp, otherDragged                                       = 25, 26, 27
)

func (d *darwin) MoveTo(x, y int32) {
	d.x, d.y = d.ox+float64(x), d.oy+float64(y)
	typ, btn := moved, 0
	switch {
	case d.buttons[0]:
		typ = leftDragged
	case d.buttons[1]:
		typ, btn = rightDragged, 1
	case d.buttons[2] || d.buttons[3] || d.buttons[4]:
		typ, btn = otherDragged, 2
	}
	C.postMouse(C.int(typ), C.double(d.x), C.double(d.y), C.int(btn), C.ulonglong(d.flags()), 0)
}

func (d *darwin) Button(b int, down bool) {
	if b < 0 || b > 4 {
		return
	}
	d.buttons[b] = down
	var typ int
	switch b {
	case 0:
		typ = pickInt(down, leftDown, leftUp)
	case 1:
		typ = pickInt(down, rightDown, rightUp)
	default:
		typ = pickInt(down, otherDown, otherUp)
	}
	// macOS needs an explicit click count for double/triple clicks.
	if down {
		if b == d.lastBtn && time.Since(d.lastClick) < 400*time.Millisecond {
			d.clicks++
		} else {
			d.clicks = 1
		}
		d.lastClick, d.lastBtn = time.Now(), b
	}
	C.postMouse(C.int(typ), C.double(d.x), C.double(d.y), C.int(b), C.ulonglong(d.flags()), C.int(d.clicks))
}

func pickInt(c bool, a, b int) int {
	if c {
		return a
	}
	return b
}

func (d *darwin) Wheel(dx, dy int32) {
	// 120 units = one notch ≈ 40 px.
	C.postScroll(C.int(dy/3), C.int(-dx/3))
}

func (d *darwin) Key(hid uint16, down bool) {
	code, ok := keymap.HIDToMac(hid)
	if !ok {
		return
	}
	isMod := keymap.IsModifier(hid)
	if isMod {
		if down {
			d.held[hid] = true
		} else {
			delete(d.held, hid)
		}
	}
	m := 0
	if isMod {
		m = 1
	}
	dn := 0
	if down {
		dn = 1
	}
	C.postKey(C.int(code), C.int(dn), C.ulonglong(d.flags()), C.int(m))
}
