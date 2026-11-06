//go:build linux

// Package linuxinput talks to the Linux kernel input layer directly:
// /dev/uinput to create virtual devices and /dev/input/event* (evdev) to read
// physical ones. It works the same under X11 and Wayland and needs no cgo.
package linuxinput

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	EvSyn = 0x00
	EvKey = 0x01
	EvRel = 0x02
	EvAbs = 0x03

	SynReport = 0

	RelX           = 0x00
	RelY           = 0x01
	RelHWheel      = 0x06
	RelWheel       = 0x08
	RelWheelHiRes  = 0x0B
	RelHWheelHiRes = 0x0C

	AbsX = 0x00
	AbsY = 0x01

	BtnLeft          = 0x110
	BtnRight         = 0x111
	BtnMiddle        = 0x112
	BtnSide          = 0x113
	BtnExtra         = 0x114
	BtnToolFinger    = 0x145
	BtnTouch         = 0x14A
	BtnToolDoubletap = 0x14D

	KeyA = 30
)

// ioctl request encoding (asm-generic, used by amd64 and arm64).
func ioc(dir, typ, nr, size uintptr) uintptr { return dir<<30 | size<<16 | typ<<8 | nr }

const (
	iocWrite = 1
	iocRead  = 2
)

var (
	uiSetEvBit   = ioc(iocWrite, 'U', 100, 4)
	uiSetKeyBit  = ioc(iocWrite, 'U', 101, 4)
	uiSetRelBit  = ioc(iocWrite, 'U', 102, 4)
	uiSetAbsBit  = ioc(iocWrite, 'U', 103, 4)
	uiDevCreate  = ioc(0, 'U', 1, 0)
	uiDevDestroy = ioc(0, 'U', 2, 0)
	eviocGrab    = ioc(iocWrite, 'E', 0x90, 4)
)

func eviocGBit(ev, n uintptr) uintptr { return ioc(iocRead, 'E', 0x20+ev, n) }
func eviocGName(n uintptr) uintptr    { return ioc(iocRead, 'E', 0x06, n) }
func eviocGAbs(abs uintptr) uintptr   { return ioc(iocRead, 'E', 0x40+abs, 24) }

func ioctl(fd, req, arg uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, arg); e != 0 {
		return e
	}
	return nil
}

// inputEvent mirrors struct input_event on 64-bit kernels.
type inputEvent struct {
	Sec, Usec int64
	Type      uint16
	Code      uint16
	Value     int32
}

const eventSize = int(unsafe.Sizeof(inputEvent{}))

// ---------- virtual devices ----------

// Device is a uinput virtual device.
type Device struct {
	mu sync.Mutex
	f  *os.File
}

// uinput_user_dev (legacy setup API, supported by every kernel).
type userDev struct {
	Name      [80]byte
	Bustype   uint16
	Vendor    uint16
	Product   uint16
	Version   uint16
	FFEffects uint32
	AbsMax    [64]int32
	AbsMin    [64]int32
	AbsFuzz   [64]int32
	AbsFlat   [64]int32
}

// DevicePrefix marks our own virtual devices so the capturer ignores them.
const DevicePrefix = "crossdesk"

func create(name string, setup func(fd uintptr) error, absMax [2]int32) (*Device, error) {
	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	fd := f.Fd()
	if err := setup(fd); err != nil {
		f.Close()
		return nil, err
	}
	var ud userDev
	copy(ud.Name[:], DevicePrefix+" "+name)
	ud.Bustype, ud.Vendor, ud.Product, ud.Version = 0x06, 0x5344, 0x0001, 1 // BUS_VIRTUAL
	ud.AbsMax[AbsX], ud.AbsMax[AbsY] = absMax[0], absMax[1]
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, &ud)
	if _, err := f.Write(buf.Bytes()); err != nil {
		f.Close()
		return nil, err
	}
	if err := ioctl(fd, uiDevCreate, 0); err != nil {
		f.Close()
		return nil, err
	}
	time.Sleep(100 * time.Millisecond) // let udev/compositor pick it up
	return &Device{f: f}, nil
}

// NewKeyboard creates a virtual keyboard able to emit the given key codes.
func NewKeyboard(codes []uint16) (*Device, error) {
	return create("keyboard", func(fd uintptr) error {
		if err := ioctl(fd, uiSetEvBit, EvKey); err != nil {
			return err
		}
		for _, c := range codes {
			ioctl(fd, uiSetKeyBit, uintptr(c))
		}
		return nil
	}, [2]int32{})
}

// NewPointer creates a virtual absolute pointer (like a tablet) covering
// 0..w-1 x 0..h-1, with buttons and wheels.
func NewPointer(w, h int32) (*Device, error) {
	return create("pointer", func(fd uintptr) error {
		for _, ev := range []uintptr{EvKey, EvAbs, EvRel} {
			if err := ioctl(fd, uiSetEvBit, ev); err != nil {
				return err
			}
		}
		for _, b := range []uintptr{BtnLeft, BtnRight, BtnMiddle, BtnSide, BtnExtra} {
			ioctl(fd, uiSetKeyBit, b)
		}
		ioctl(fd, uiSetAbsBit, AbsX)
		ioctl(fd, uiSetAbsBit, AbsY)
		for _, r := range []uintptr{RelWheel, RelHWheel, RelWheelHiRes, RelHWheelHiRes} {
			ioctl(fd, uiSetRelBit, r)
		}
		return nil
	}, [2]int32{max(w-1, 1), max(h-1, 1)})
}

// Emit writes events followed by a SYN_REPORT.
func (d *Device) Emit(evs ...[3]int32) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	var buf bytes.Buffer
	now := time.Now()
	for _, e := range append(evs, [3]int32{EvSyn, SynReport, 0}) {
		binary.Write(&buf, binary.LittleEndian, inputEvent{Sec: now.Unix(), Usec: int64(now.Nanosecond() / 1000),
			Type: uint16(e[0]), Code: uint16(e[1]), Value: e[2]})
	}
	_, err := d.f.Write(buf.Bytes())
	return err
}

func (d *Device) Close() {
	ioctl(d.f.Fd(), uiDevDestroy, 0)
	d.f.Close()
}

// ---------- physical devices ----------

type Kind int

const (
	Keyboard Kind = 1 << iota
	Mouse
	Touchpad
)

type Event struct {
	Type, Code uint16
	Value      int32
}

// Physical is an opened /dev/input/event* device.
type Physical struct {
	Path    string
	Name    string
	Kind    Kind
	HiRes   bool     // emits REL_WHEEL_HI_RES
	AbsRes  [2]int32 // touchpad units per mm (0 = unknown)
	AbsSpan [2]int32
	f       *os.File
}

func testBit(bits []byte, n int) bool { return n/8 < len(bits) && bits[n/8]&(1<<(n%8)) != 0 }

func OpenPhysical(path string) (*Physical, error) {
	p, err := openRaw(path)
	if err != nil {
		return nil, err
	}
	if p.Kind == 0 || strings.HasPrefix(p.Name, DevicePrefix) {
		p.Close()
		return nil, errors.New("not an input device we use")
	}
	return p, nil
}

// openRaw opens and classifies any evdev node (including our own devices).
func openRaw(path string) (*Physical, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	fd := f.Fd()
	name := make([]byte, 256)
	ioctl(fd, eviocGName(256), uintptr(unsafe.Pointer(&name[0])))
	p := &Physical{Path: path, Name: strings.TrimRight(string(name), "\x00"), f: f}

	evBits := make([]byte, 4)
	keyBits := make([]byte, 96)
	relBits := make([]byte, 4)
	absBits := make([]byte, 8)
	ioctl(fd, eviocGBit(0, 4), uintptr(unsafe.Pointer(&evBits[0])))
	ioctl(fd, eviocGBit(EvKey, 96), uintptr(unsafe.Pointer(&keyBits[0])))
	ioctl(fd, eviocGBit(EvRel, 4), uintptr(unsafe.Pointer(&relBits[0])))
	ioctl(fd, eviocGBit(EvAbs, 8), uintptr(unsafe.Pointer(&absBits[0])))

	if testBit(evBits, EvKey) && testBit(keyBits, KeyA) {
		p.Kind |= Keyboard
	}
	if testBit(evBits, EvRel) && testBit(relBits, RelX) && testBit(keyBits, BtnLeft) {
		p.Kind |= Mouse
		p.HiRes = testBit(relBits, RelWheelHiRes)
	}
	if testBit(evBits, EvAbs) && testBit(absBits, AbsX) && testBit(keyBits, BtnToolFinger) {
		p.Kind |= Touchpad
		for i, axis := range []uintptr{AbsX, AbsY} {
			var info [6]int32 // value, min, max, fuzz, flat, resolution
			if ioctl(fd, eviocGAbs(axis), uintptr(unsafe.Pointer(&info[0]))) == nil {
				p.AbsRes[i], p.AbsSpan[i] = info[5], info[2]-info[1]
			}
		}
	}
	return p, nil
}

// Grab takes (or releases) exclusive access: while grabbed, the desktop
// receives nothing from this device.
func (p *Physical) Grab(on bool) error {
	v := 0
	if on {
		v = 1
	}
	return ioctl(p.f.Fd(), eviocGrab, uintptr(v))
}

// Read blocks for the next batch of events.
func (p *Physical) Read(buf []byte) ([]Event, error) {
	n, err := p.f.Read(buf)
	if err != nil {
		return nil, err
	}
	evs := make([]Event, 0, n/eventSize)
	for off := 0; off+eventSize <= n; off += eventSize {
		var e inputEvent
		binary.Read(bytes.NewReader(buf[off:off+eventSize]), binary.LittleEndian, &e)
		evs = append(evs, Event{Type: e.Type, Code: e.Code, Value: e.Value})
	}
	return evs, nil
}

func (p *Physical) Close() { p.f.Close() }

// EventPaths lists /dev/input/event* nodes.
func EventPaths() []string {
	m, _ := filepath.Glob("/dev/input/event*")
	return m
}

// ---------- environment ----------

// CheckAccess reports what is missing for full functionality.
func CheckAccess() (canInject, canCapture bool) {
	if f, err := os.OpenFile("/dev/uinput", os.O_WRONLY, 0); err == nil {
		f.Close()
		canInject = true
	}
	for _, p := range EventPaths() {
		if f, err := os.Open(p); err == nil {
			f.Close()
			canCapture = true
			break
		}
	}
	return
}

var xrandrScreen = regexp.MustCompile(`current (\d+) x (\d+)`)

// ScreenSize returns the desktop size in pixels: CROSSDESK_SCREEN=WxH,
// else xrandr (X11 / XWayland), else the first connected DRM output.
func ScreenSize() (int32, int32) {
	if v := os.Getenv("CROSSDESK_SCREEN"); v != "" {
		if w, h, ok := parseWxH(v); ok {
			return w, h
		}
	}
	if out, err := exec.Command("xrandr", "--current").Output(); err == nil {
		if m := xrandrScreen.FindSubmatch(out); m != nil {
			w, _ := strconv.Atoi(string(m[1]))
			h, _ := strconv.Atoi(string(m[2]))
			if w > 0 && h > 0 {
				return int32(w), int32(h)
			}
		}
	}
	outs, _ := filepath.Glob("/sys/class/drm/card*-*")
	for _, o := range outs {
		st, _ := os.ReadFile(filepath.Join(o, "status"))
		if strings.TrimSpace(string(st)) != "connected" {
			continue
		}
		modes, _ := os.ReadFile(filepath.Join(o, "modes"))
		if line, _, _ := strings.Cut(string(modes), "\n"); line != "" {
			if w, h, ok := parseWxH(line); ok {
				return w, h
			}
		}
	}
	return 1920, 1080
}

func parseWxH(s string) (int32, int32, bool) {
	a, b, ok := strings.Cut(strings.TrimSpace(s), "x")
	if !ok {
		return 0, 0, false
	}
	b, _, _ = strings.Cut(b, "i") // "1920x1080i"
	w, e1 := strconv.Atoi(a)
	h, e2 := strconv.Atoi(b)
	if e1 != nil || e2 != nil || w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return int32(w), int32(h), true
}
