// Package inject synthesizes mouse and keyboard input on the client machine.
package inject

// Screen describes the area the cursor can move in, in the units the injector
// uses for MoveTo (pixels on Windows, points on macOS).
type Screen struct{ X, Y, W, H int32 }

type Injector interface {
	Screen() Screen
	// MoveTo moves to (x,y) relative to Screen origin.
	MoveTo(x, y int32)
	Button(button int, down bool)
	// Wheel scrolls in 1/120 notch units; positive dy = up, positive dx = right.
	Wheel(dx, dy int32)
	// Key presses or releases a key given as a USB HID usage code.
	Key(hid uint16, down bool)
}
