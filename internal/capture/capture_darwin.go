package capture

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>

extern int goTapCallback(int type, int keycode, unsigned long long flags,
	double x, double y, double dx, double dy, int button,
	long long lineY, long long lineX, long long pointY, long long pointX, int continuous);

static CFMachPortRef gTap = NULL;
static CFRunLoopSourceRef gSrc = NULL;
static CFRunLoopRef gLoop = NULL;
static CGEventMask gMask = 0;

static CGEventRef tapCB(CGEventTapProxy proxy, CGEventType type, CGEventRef ev, void *ref) {
	if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
		if (gTap) CGEventTapEnable(gTap, true);
		return ev;
	}
	CGPoint p = CGEventGetLocation(ev);
	int swallow = goTapCallback((int)type,
		(int)CGEventGetIntegerValueField(ev, kCGKeyboardEventKeycode),
		(unsigned long long)CGEventGetFlags(ev),
		p.x, p.y,
		CGEventGetDoubleValueField(ev, kCGMouseEventDeltaX),
		CGEventGetDoubleValueField(ev, kCGMouseEventDeltaY),
		(int)CGEventGetIntegerValueField(ev, kCGMouseEventButtonNumber),
		CGEventGetIntegerValueField(ev, kCGScrollWheelEventDeltaAxis1),
		CGEventGetIntegerValueField(ev, kCGScrollWheelEventDeltaAxis2),
		CGEventGetIntegerValueField(ev, kCGScrollWheelEventPointDeltaAxis1),
		CGEventGetIntegerValueField(ev, kCGScrollWheelEventPointDeltaAxis2),
		(int)CGEventGetIntegerValueField(ev, kCGScrollWheelEventIsContinuous));
	return swallow ? NULL : ev;
}

static int runTap(void) {
	CGEventMask mask =
		CGEventMaskBit(kCGEventMouseMoved) |
		CGEventMaskBit(kCGEventLeftMouseDown) | CGEventMaskBit(kCGEventLeftMouseUp) |
		CGEventMaskBit(kCGEventRightMouseDown) | CGEventMaskBit(kCGEventRightMouseUp) |
		CGEventMaskBit(kCGEventOtherMouseDown) | CGEventMaskBit(kCGEventOtherMouseUp) |
		CGEventMaskBit(kCGEventLeftMouseDragged) | CGEventMaskBit(kCGEventRightMouseDragged) |
		CGEventMaskBit(kCGEventOtherMouseDragged) |
		CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventKeyUp) |
		CGEventMaskBit(kCGEventFlagsChanged) | CGEventMaskBit(kCGEventScrollWheel);
	gMask = mask;
	gTap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap,
		kCGEventTapOptionDefault, mask, tapCB, NULL);
	if (!gTap) return -1;
	gSrc = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, gTap, 0);
	gLoop = CFRunLoopGetCurrent();
	CFRunLoopAddSource(gLoop, gSrc, kCFRunLoopCommonModes);
	CGEventTapEnable(gTap, true);
	CFRunLoopRun();
	return 0;
}

// Taps inserted at the head run newest-first. Keyboard tools that also use
// event taps (OpenKey, EVKey for Mac...) may have been started after us;
// re-inserting our tap puts it first again, so keys for the other machine
// are taken before a local Vietnamese input method rewrites them.
static void reinsertTap(void) {
	if (!gLoop) return;
	CFRunLoopPerformBlock(gLoop, kCFRunLoopCommonModes, ^{
		CFMachPortRef t = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap,
			kCGEventTapOptionDefault, gMask, tapCB, NULL);
		if (!t) return;
		CFRunLoopSourceRef s = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, t, 0);
		CFRunLoopAddSource(gLoop, s, kCFRunLoopCommonModes);
		CGEventTapEnable(t, true);
		if (gTap) {
			CGEventTapEnable(gTap, false);
			CFRunLoopRemoveSource(gLoop, gSrc, kCFRunLoopCommonModes);
			CFMachPortInvalidate(gTap);
			CFRelease(gSrc);
			CFRelease(gTap);
		}
		gTap = t;
		gSrc = s;
	});
	CFRunLoopWakeUp(gLoop);
}

static int isTrusted(int prompt) {
	const void *keys[] = { kAXTrustedCheckOptionPrompt };
	const void *vals[] = { prompt ? kCFBooleanTrue : kCFBooleanFalse };
	CFDictionaryRef opts = CFDictionaryCreate(NULL, keys, vals, 1,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	Boolean ok = AXIsProcessTrustedWithOptions(opts);
	CFRelease(opts);
	return ok ? 1 : 0;
}

static void freezeCursor(int freeze) {
	CGAssociateMouseAndMouseCursorPosition(freeze ? false : true);
}

static void warpCursor(double x, double y) {
	CGWarpMouseCursorPosition(CGPointMake(x, y));
}

static void displayBounds(double *minX, double *minY, double *maxX, double *maxY) {
	CGDirectDisplayID ids[16];
	uint32_t n = 0;
	CGGetActiveDisplayList(16, ids, &n);
	*minX = *minY = 1e9; *maxX = *maxY = -1e9;
	for (uint32_t i = 0; i < n; i++) {
		CGRect r = CGDisplayBounds(ids[i]);
		if (r.origin.x < *minX) *minX = r.origin.x;
		if (r.origin.y < *minY) *minY = r.origin.y;
		if (r.origin.x + r.size.width > *maxX) *maxX = r.origin.x + r.size.width;
		if (r.origin.y + r.size.height > *maxY) *maxY = r.origin.y + r.size.height;
	}
}
*/
import "C"

import (
	"errors"
	"runtime"

	"crossdesk/internal/keymap"
)

var handler Handler

// CGEventType values.
const (
	evLeftDown     = 1
	evLeftUp       = 2
	evRightDown    = 3
	evRightUp      = 4
	evMoved        = 5
	evLeftDragged  = 6
	evRightDragged = 7
	evKeyDown      = 10
	evKeyUp        = 11
	evFlagsChanged = 12
	evScroll       = 22
	evOtherDown    = 25
	evOtherUp      = 26
	evOtherDragged = 27
)

// Device-dependent modifier bits (NX_DEVICE*KEYMASK) that tell left from right.
var modifierBit = map[uint16]uint64{
	0x3B: 0x0001, // left ctrl
	0x38: 0x0002, // left shift
	0x3C: 0x0004, // right shift
	0x37: 0x0008, // left cmd
	0x36: 0x0010, // right cmd
	0x3A: 0x0020, // left option
	0x3D: 0x0040, // right option
	0x3E: 0x2000, // right ctrl
}

// Device-independent fallbacks if a synthetic event has no device bits.
var modifierGeneric = map[uint16]uint64{
	0x3B: 0x40000, 0x3E: 0x40000, // ctrl
	0x38: 0x20000, 0x3C: 0x20000, // shift
	0x37: 0x100000, 0x36: 0x100000, // cmd
	0x3A: 0x80000, 0x3D: 0x80000, // option
}

//export goTapCallback
func goTapCallback(typ C.int, keycode C.int, flags C.ulonglong, x, y, dx, dy C.double,
	button C.int, lineY, lineX, pointY, pointX C.longlong, continuous C.int) C.int {
	if handler == nil {
		return 0
	}
	ev := Event{X: float64(x), Y: float64(y)}
	switch typ {
	case evMoved, evLeftDragged, evRightDragged, evOtherDragged:
		ev.Kind = Move
		ev.DX, ev.DY = float64(dx), float64(dy)
	case evLeftDown, evLeftUp:
		ev.Kind, ev.Button, ev.Down = Button, 0, typ == evLeftDown
	case evRightDown, evRightUp:
		ev.Kind, ev.Button, ev.Down = Button, 1, typ == evRightDown
	case evOtherDown, evOtherUp:
		ev.Kind, ev.Button, ev.Down = Button, int(button), typ == evOtherDown
	case evScroll:
		ev.Kind = Scroll
		if continuous != 0 {
			// Trackpad / Magic Mouse: pixel deltas. ~40px of swipe per notch feels natural.
			ev.ScrollY = int32(pointY) * 3
			ev.ScrollX = -int32(pointX) * 3
		} else {
			ev.ScrollY = int32(lineY) * 120
			ev.ScrollX = -int32(lineX) * 120
		}
		if ev.ScrollX == 0 && ev.ScrollY == 0 {
			return 0
		}
	case evKeyDown, evKeyUp:
		hid, ok := keymap.MacToHID(uint16(keycode))
		if !ok {
			return 0
		}
		ev.Kind, ev.Key, ev.Down = Key, hid, typ == evKeyDown
	case evFlagsChanged:
		code := uint16(keycode)
		hid, ok := keymap.MacToHID(code)
		if !ok {
			return 0 // e.g. Fn
		}
		ev.Kind, ev.Key = Key, hid
		if code == 0x39 { // CapsLock: one event per press; emit a full tap.
			ev.Down = true
			r := handler(ev)
			ev.Down = false
			r = handler(ev) || r
			return boolToC(r)
		}
		f := uint64(flags)
		if f&0xFFFF != 0 {
			ev.Down = f&modifierBit[code] != 0
		} else {
			ev.Down = f&modifierGeneric[code] != 0
		}
	default:
		return 0
	}
	return boolToC(handler(ev))
}

func boolToC(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// Trusted reports whether the process has the Accessibility permission. With
// prompt=true macOS shows the "grant access" dialog.
func Trusted(prompt bool) bool {
	p := 0
	if prompt {
		p = 1
	}
	return C.isTrusted(C.int(p)) == 1
}

// Run installs the global event tap and blocks forever.
func Run(h Handler) error {
	handler = h
	runtime.LockOSThread()
	if C.runTap() != 0 {
		return errors.New("cannot create event tap: grant Accessibility (and Input Monitoring) permission to this terminal/app in System Settings > Privacy & Security, then restart it")
	}
	return nil
}

// Freeze stops the visible cursor from moving while still reporting deltas.
func Freeze(f bool) {
	if f {
		C.reinsertTap()
		C.freezeCursor(1)
	} else {
		C.freezeCursor(0)
	}
}

func Warp(x, y float64) { C.warpCursor(C.double(x), C.double(y)) }

func ScreenBounds() Bounds {
	var b Bounds
	C.displayBounds((*C.double)(&b.MinX), (*C.double)(&b.MinY), (*C.double)(&b.MaxX), (*C.double)(&b.MaxY))
	return b
}
