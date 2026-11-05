// Package capture grabs global mouse/keyboard input on the machine that owns
// the physical devices, and lets the caller decide per event whether it should
// be swallowed (forwarded to a remote machine) or delivered locally.
package capture

type Kind int

const (
	Move Kind = iota
	Button
	Key
	Scroll
)

type Event struct {
	Kind Kind
	// Cursor location in global screen coordinates (points on macOS).
	X, Y float64
	// Relative motion for Move events (still reported while the cursor is frozen).
	DX, DY float64
	// Button: 0 left, 1 right, 2 middle, 3 back, 4 forward.
	Button int
	Down   bool
	// Key: USB HID usage code.
	Key uint16
	// Scroll in 1/120 notch units. Positive Y = up, positive X = right.
	ScrollX, ScrollY int32
}

// Handler is called synchronously for every input event. Return true to
// swallow the event so the local machine never sees it. It must be fast.
type Handler func(Event) bool

type Bounds struct{ MinX, MinY, MaxX, MaxY float64 }
