package capture

import (
	"errors"
	"log"
	"sync"
	"time"

	"sharedevice/internal/keymap"
	li "sharedevice/internal/linuxinput"
)

// Linux capture reads evdev devices directly, so it works under X11 and
// Wayland alike. Wayland does not let apps read the global cursor position,
// so the position is dead-reckoned from relative motion and clamped to the
// screen: pushing against an edge re-synchronises it. To avoid switching by
// mistake when the estimate drifts, the cursor must be pushed `pushThrough`
// pixels past the edge before the event is reported as "at the edge".
// While the cursor is on another machine, devices are grabbed (EVIOCGRAB) and
// any event that must still reach this desktop is replayed through uinput.

type linuxCapture struct {
	mu      sync.Mutex
	devs    map[string]*physState
	grabbed bool
	w, h    float64
	x, y    float64
	overX   float64 // accumulated push beyond the left/right edge
	overY   float64
	ptr     *li.Device // to warp the cursor and resync
	kbd     *li.Device // to replay keys while grabbed
}

type physState struct {
	*li.Physical
	fingers  int
	touching bool
	lastX    int32
	lastY    int32
	hasLastX bool
	hasLastY bool
}

var lc = &linuxCapture{devs: map[string]*physState{}}
var handler Handler

func Trusted(bool) bool {
	inj, cap := li.CheckAccess()
	return inj && cap
}

func Run(h Handler) error {
	handler = h
	lc.mu.Lock()
	w, hh := li.ScreenSize()
	lc.w, lc.h = float64(w), float64(hh)
	var err error
	if lc.ptr, err = li.NewPointer(w, hh); err != nil {
		lc.mu.Unlock()
		return errors.New("cannot open /dev/uinput: " + err.Error())
	}
	if lc.kbd, err = li.NewKeyboard(keymap.LinuxKeyCodes()); err != nil {
		lc.mu.Unlock()
		return errors.New("cannot open /dev/uinput: " + err.Error())
	}
	// Start from a known position.
	lc.x, lc.y = lc.w/2, lc.h/2
	lc.ptr.Emit([3]int32{li.EvAbs, li.AbsX, int32(lc.x)}, [3]int32{li.EvAbs, li.AbsY, int32(lc.y)})
	lc.mu.Unlock()

	for {
		lc.scan()
		time.Sleep(3 * time.Second)
	}
}

// scan opens newly plugged-in devices.
func (c *linuxCapture) scan() {
	for _, path := range li.EventPaths() {
		c.mu.Lock()
		_, known := c.devs[path]
		c.mu.Unlock()
		if known {
			continue
		}
		p, err := li.OpenPhysical(path)
		if err != nil {
			continue
		}
		st := &physState{Physical: p}
		c.mu.Lock()
		c.devs[path] = st
		if c.grabbed {
			p.Grab(true)
		}
		c.mu.Unlock()
		log.Printf("capturing %s (%s)", p.Name, path)
		go c.read(st)
	}
}

func (c *linuxCapture) read(d *physState) {
	buf := make([]byte, 24*64)
	for {
		evs, err := d.Read(buf)
		if err != nil {
			c.mu.Lock()
			delete(c.devs, d.Path)
			c.mu.Unlock()
			d.Close()
			return
		}
		c.process(d, evs)
	}
}

// process turns one batch of evdev events into capture events.
func (c *linuxCapture) process(d *physState, evs []li.Event) {
	var dx, dy float64
	var wheelX, wheelY int32
	for _, e := range evs {
		switch e.Type {
		case li.EvRel:
			switch e.Code {
			case li.RelX:
				dx += float64(e.Value)
			case li.RelY:
				dy += float64(e.Value)
			case li.RelWheelHiRes:
				wheelY += e.Value
			case li.RelHWheelHiRes:
				wheelX += e.Value
			case li.RelWheel:
				if !d.HiRes {
					wheelY += e.Value * 120
				}
			case li.RelHWheel:
				if !d.HiRes {
					wheelX += e.Value * 120
				}
			}
		case li.EvAbs:
			if d.Kind&li.Touchpad == 0 || !d.touching {
				continue
			}
			scale := touchScale(d.AbsRes, d.AbsSpan)
			switch e.Code {
			case li.AbsX:
				if d.hasLastX {
					dx += float64(e.Value-d.lastX) * scale
				}
				d.lastX, d.hasLastX = e.Value, true
			case li.AbsY:
				if d.hasLastY {
					dy += float64(e.Value-d.lastY) * scale
				}
				d.lastY, d.hasLastY = e.Value, true
			}
		case li.EvKey:
			switch e.Code {
			case li.BtnTouch:
				d.touching = e.Value != 0
				d.hasLastX, d.hasLastY = false, false
			case li.BtnToolFinger:
				if e.Value != 0 {
					d.fingers = 1
				}
			case li.BtnToolDoubletap:
				if e.Value != 0 {
					d.fingers = 2
				} else {
					d.fingers = 1
				}
			case li.BtnLeft, li.BtnRight, li.BtnMiddle, li.BtnSide, li.BtnExtra:
				if e.Value == 2 {
					continue
				}
				c.emit(Event{Kind: Button, Button: int(e.Code - li.BtnLeft), Down: e.Value != 0}, e)
			default:
				hid, ok := keymap.LinuxToHID(e.Code)
				if !ok {
					continue
				}
				// value 2 = autorepeat: forward as another key down.
				c.emit(Event{Kind: Key, Key: hid, Down: e.Value != 0}, e)
			}
		}
	}
	// Two-finger touchpad motion scrolls instead of moving.
	if d.Kind&li.Touchpad != 0 && d.fingers >= 2 {
		wheelY += int32(dy * 4)
		wheelX -= int32(dx * 4)
		dx, dy = 0, 0
	}
	if wheelX != 0 || wheelY != 0 {
		handler(Event{Kind: Scroll, ScrollX: wheelX, ScrollY: wheelY})
	}
	if dx != 0 || dy != 0 {
		c.move(dx, dy)
	}
}

// touchScale converts touchpad units to approximate screen pixels.
func touchScale(res, span [2]int32) float64 {
	if res[0] > 0 {
		return 5.0 / float64(res[0]) // ~5 px per mm
	}
	if span[0] > 0 {
		return 1500.0 / float64(span[0])
	}
	return 1
}

// emit sends a key/button event to the handler; if it is not swallowed while
// devices are grabbed, it is replayed locally.
func (c *linuxCapture) emit(ev Event, raw li.Event) {
	swallow := handler(ev)
	c.mu.Lock()
	grabbed := c.grabbed
	c.mu.Unlock()
	if !swallow && grabbed {
		if ev.Kind == Key {
			c.kbd.Emit([3]int32{li.EvKey, int32(raw.Code), raw.Value})
		} else {
			c.ptr.Emit([3]int32{li.EvKey, int32(raw.Code), raw.Value})
		}
	}
}

func (c *linuxCapture) move(dx, dy float64) {
	c.mu.Lock()
	ev := Event{Kind: Move, DX: dx, DY: dy}
	if !c.grabbed {
		nx, ny := c.x+dx, c.y+dy
		c.overX = edgePush(nx, c.w, c.overX, dx)
		c.overY = edgePush(ny, c.h, c.overY, dy)
		c.x = min(max(nx, 0), c.w-1)
		c.y = min(max(ny, 0), c.h-1)
		ev.X, ev.Y = c.reportX(), c.reportY()
	}
	c.mu.Unlock()
	handler(ev)
}

// reportX/Y keep the reported position one pixel inside the screen until
// the user has pushed far enough past the edge.
func (c *linuxCapture) reportX() float64 {
	if c.overX < pushThrough {
		return min(max(c.x, 1), c.w-2)
	}
	return c.x
}

func (c *linuxCapture) reportY() float64 {
	if c.overY < pushThrough {
		return min(max(c.y, 1), c.h-2)
	}
	return c.y
}

func Freeze(f bool) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.grabbed == f {
		return
	}
	lc.grabbed = f
	for _, d := range lc.devs {
		if err := d.Grab(f); err != nil {
			log.Printf("grab %s: %v", d.Name, err)
		}
	}
	lc.overX, lc.overY = 0, 0
}

func Warp(x, y float64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.x, lc.y = x, y
	if lc.ptr != nil {
		lc.ptr.Emit([3]int32{li.EvAbs, li.AbsX, int32(x)}, [3]int32{li.EvAbs, li.AbsY, int32(y)})
	}
}

func ScreenBounds() Bounds {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if lc.w == 0 {
		w, h := li.ScreenSize()
		return Bounds{MaxX: float64(w), MaxY: float64(h)}
	}
	return Bounds{MaxX: lc.w, MaxY: lc.h}
}
