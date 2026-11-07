package clipboard

import (
	"errors"
	"runtime"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

var (
	user32                      = syscall.NewLazyDLL("user32.dll")
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	pOpenClipboard              = user32.NewProc("OpenClipboard")
	pCloseClipboard             = user32.NewProc("CloseClipboard")
	pEmptyClipboard             = user32.NewProc("EmptyClipboard")
	pGetClipboardData           = user32.NewProc("GetClipboardData")
	pSetClipboardData           = user32.NewProc("SetClipboardData")
	pIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	pRegisterClipboardFormat    = user32.NewProc("RegisterClipboardFormatW")
	pGlobalAlloc                = kernel32.NewProc("GlobalAlloc")
	pGlobalFree                 = kernel32.NewProc("GlobalFree")
	pGlobalLock                 = kernel32.NewProc("GlobalLock")
	pGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
	pGlobalSize                 = kernel32.NewProc("GlobalSize")
)

const (
	cfUnicodeText = 13
	cfDIB         = 8
	gmemMoveable  = 0x0002
)

var cfPNG = func() uintptr {
	name, _ := syscall.UTF16PtrFromString("PNG")
	r, _, _ := pRegisterClipboardFormat.Call(uintptr(unsafe.Pointer(name)))
	return r
}()

// memAt views size bytes of global memory returned by GlobalLock.
// The handle stays locked while the slice is used.
func memAt(p uintptr, size int) []byte {
	return unsafe.Slice(*(**byte)(unsafe.Pointer(&p)), size)
}

// open retries because other apps hold the clipboard briefly.
func open() error {
	for i := 0; i < 20; i++ {
		if r, _, _ := pOpenClipboard.Call(0); r != 0 {
			return nil
		}
		time.Sleep(15 * time.Millisecond)
	}
	return errors.New("clipboard is busy")
}

func available(f uintptr) bool {
	r, _, _ := pIsClipboardFormatAvailable.Call(f)
	return r != 0
}

func getBytes(f uintptr) []byte {
	h, _, _ := pGetClipboardData.Call(f)
	if h == 0 {
		return nil
	}
	size, _, _ := pGlobalSize.Call(h)
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		return nil
	}
	defer pGlobalUnlock.Call(h)
	return append([]byte(nil), memAt(p, int(size))...)
}

func read() (Content, bool) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if open() != nil {
		return Content{}, false
	}
	defer pCloseClipboard.Call()
	if available(cfUnicodeText) {
		raw := getBytes(cfUnicodeText)
		u := make([]uint16, len(raw)/2)
		for i := range u {
			u[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		for i, c := range u {
			if c == 0 {
				u = u[:i]
				break
			}
		}
		if len(u) > 0 {
			return Content{Kind: Text, Data: []byte(string(utf16.Decode(u)))}, true
		}
	}
	if cfPNG != 0 && available(cfPNG) {
		if b := getBytes(cfPNG); len(b) > 0 {
			return Content{Kind: PNG, Data: b}, true
		}
	}
	if available(cfDIB) {
		if p, err := dibToPNG(getBytes(cfDIB)); err == nil {
			return Content{Kind: PNG, Data: p}, true
		}
	}
	return Content{}, false
}

func setBytes(f uintptr, b []byte) error {
	h, _, _ := pGlobalAlloc.Call(gmemMoveable, uintptr(len(b)))
	if h == 0 {
		return errors.New("GlobalAlloc failed")
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		pGlobalFree.Call(h)
		return errors.New("GlobalLock failed")
	}
	copy(memAt(p, len(b)), b)
	pGlobalUnlock.Call(h)
	if r, _, _ := pSetClipboardData.Call(f, h); r == 0 {
		pGlobalFree.Call(h)
		return errors.New("SetClipboardData failed")
	}
	return nil // the clipboard now owns h
}

func write(c Content) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := open(); err != nil {
		return err
	}
	defer pCloseClipboard.Call()
	pEmptyClipboard.Call()
	if c.Kind == Text {
		u := utf16.Encode([]rune(string(c.Data)))
		b := make([]byte, 2*len(u)+2)
		for i, v := range u {
			b[2*i], b[2*i+1] = byte(v), byte(v>>8)
		}
		return setBytes(cfUnicodeText, b)
	}
	if cfPNG != 0 {
		setBytes(cfPNG, c.Data)
	}
	dib, err := pngToDIB(c.Data)
	if err != nil {
		return err
	}
	return setBytes(cfDIB, dib)
}

func problem() string { return "" }
