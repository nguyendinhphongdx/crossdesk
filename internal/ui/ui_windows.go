package ui

import (
	_ "embed"
	"encoding/binary"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
)

//go:embed assets/crossdesk.ico
var icoFile []byte

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	shell32                   = syscall.NewLazyDLL("shell32.dll")
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	pRegisterClassEx          = user32.NewProc("RegisterClassExW")
	pCreateWindowEx           = user32.NewProc("CreateWindowExW")
	pDefWindowProc            = user32.NewProc("DefWindowProcW")
	pGetMessage               = user32.NewProc("GetMessageW")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessage          = user32.NewProc("DispatchMessageW")
	pCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	pAppendMenu               = user32.NewProc("AppendMenuW")
	pTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	pDestroyMenu              = user32.NewProc("DestroyMenu")
	pGetCursorPos             = user32.NewProc("GetCursorPos")
	pSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	pPostMessage              = user32.NewProc("PostMessageW")
	pShowWindow               = user32.NewProc("ShowWindow")
	pSendMessage              = user32.NewProc("SendMessageW")
	pRegisterWindowMessage    = user32.NewProc("RegisterWindowMessageW")
	pCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	pShellNotifyIcon          = shell32.NewProc("Shell_NotifyIconW")
	pGetModuleHandle          = kernel32.NewProc("GetModuleHandleW")
)

const (
	wmNull         = 0x0000
	wmSetIcon      = 0x0080
	wmLButtonUp    = 0x0202
	wmRButtonUp    = 0x0205
	wmTray         = 0x8001 // WM_APP + 1
	nimAdd         = 0
	nimDelete      = 2
	nifMessage     = 0x1
	nifIcon        = 0x2
	nifTip         = 0x4
	mfString       = 0x0
	mfSeparator    = 0x800
	tpmReturnCmd   = 0x100
	tpmRightAlign  = 0x8
	tpmBottomAlign = 0x20
	swRestore      = 9
	hwndMessage    = ^uintptr(2) // (HWND)-3

	cmdOpen = 1
	cmdQuit = 2
)

type point struct{ X, Y int32 }

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	_       uint32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type notifyIconData struct {
	Size            uint32
	Wnd             uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Version         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GUIDItem        [16]byte
	BalloonIcon     uintptr
}

var (
	opts       Options
	trayWnd    uintptr
	taskbarMsg uintptr
	nid        notifyIconData

	winMu   sync.Mutex
	current webview2.WebView
	openReq = make(chan struct{}, 1)
)

// iconFromICO creates an HICON from the embedded .ico, choosing the entry
// closest to size px (entries may hold PNG data, which Windows accepts).
func iconFromICO(size int) uintptr {
	if len(icoFile) < 6 {
		return 0
	}
	n := int(binary.LittleEndian.Uint16(icoFile[4:]))
	best, bestDiff := -1, 1<<30
	for i := 0; i < n && 6+16*(i+1) <= len(icoFile); i++ {
		w := int(icoFile[6+16*i])
		if w == 0 {
			w = 256
		}
		d := w - size
		if d < 0 {
			d = -d * 4 // prefer scaling down over scaling up
		}
		if d < bestDiff {
			best, bestDiff = i, d
		}
	}
	if best < 0 {
		return 0
	}
	e := icoFile[6+16*best:]
	ln := int(binary.LittleEndian.Uint32(e[8:]))
	off := int(binary.LittleEndian.Uint32(e[12:]))
	if off+ln > len(icoFile) {
		return 0
	}
	h, _, _ := pCreateIconFromResourceEx.Call(uintptr(unsafe.Pointer(&icoFile[off])), uintptr(ln), 1, 0x00030000, uintptr(size), uintptr(size), 0)
	return h
}

func utf16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }

func addTrayIcon() {
	nid = notifyIconData{Wnd: trayWnd, ID: 1, Flags: nifMessage | nifIcon | nifTip, CallbackMessage: wmTray, Icon: iconFromICO(16)}
	nid.Size = uint32(unsafe.Sizeof(nid))
	tip, _ := syscall.UTF16FromString("Crossdesk")
	copy(nid.Tip[:], tip)
	pShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
}

func showMenu() {
	m, _, _ := pCreatePopupMenu.Call()
	pAppendMenu.Call(m, mfString, cmdOpen, uintptr(unsafe.Pointer(utf16("Mở Crossdesk"))))
	pAppendMenu.Call(m, mfSeparator, 0, 0)
	pAppendMenu.Call(m, mfString, cmdQuit, uintptr(unsafe.Pointer(utf16("Thoát Crossdesk"))))
	var p point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	pSetForegroundWindow.Call(trayWnd)
	cmd, _, _ := pTrackPopupMenu.Call(m, tpmReturnCmd|tpmRightAlign|tpmBottomAlign, uintptr(p.X), uintptr(p.Y), 0, trayWnd, 0)
	pPostMessage.Call(trayWnd, wmNull, 0, 0)
	pDestroyMenu.Call(m)
	switch cmd {
	case cmdOpen:
		Show()
	case cmdQuit:
		quit()
	}
}

func quit() {
	pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	if opts.OnQuit != nil {
		go opts.OnQuit()
	}
}

func trayProc(hwnd, m, wp, lp uintptr) uintptr {
	switch {
	case m == wmTray:
		switch lp & 0xFFFF {
		case wmLButtonUp:
			Show()
		case wmRButtonUp:
			showMenu()
		}
		return 0
	case taskbarMsg != 0 && m == taskbarMsg: // Explorer restarted: re-add the icon
		addTrayIcon()
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, m, wp, lp)
	return r
}

// windowLoop owns the WebView2 window on its own UI thread. Closing the
// window ends Run(); the app keeps living in the notification area.
func windowLoop() {
	runtime.LockOSThread()
	for range openReq {
		dataDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "crossdesk", "webview2")
		w := webview2.NewWithOptions(webview2.WebViewOptions{
			DataPath:  dataDir,
			AutoFocus: true,
			WindowOptions: webview2.WindowOptions{
				Title: "Crossdesk", Width: 1120, Height: 800, Center: true,
			},
		})
		if w == nil {
			log.Print("WebView2 runtime not available; opening the browser instead")
			OpenBrowser(opts.URL)
			continue
		}
		hwnd := uintptr(w.Window())
		if big := iconFromICO(32); big != 0 {
			pSendMessage.Call(hwnd, wmSetIcon, 1, big)
		}
		if small := iconFromICO(16); small != 0 {
			pSendMessage.Call(hwnd, wmSetIcon, 0, small)
		}
		w.SetSize(720, 520, webview2.HintMin)
		w.Navigate(opts.URL)
		winMu.Lock()
		current = w
		winMu.Unlock()
		w.Run()
		winMu.Lock()
		current = nil
		winMu.Unlock()
	}
}

// Run shows the notification-area icon and blocks forever. Call it from main().
func Run(o Options) {
	opts = o
	runtime.LockOSThread()
	go windowLoop()

	inst, _, _ := pGetModuleHandle.Call(0)
	cls := utf16("CrossdeskTray")
	wc := wndClassEx{WndProc: syscall.NewCallback(trayProc), Instance: inst, ClassName: cls}
	wc.Size = uint32(unsafe.Sizeof(wc))
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	trayWnd, _, _ = pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(utf16("Crossdesk"))), 0, 0, 0, 0, 0, hwndMessage, 0, inst, 0)
	taskbarMsg, _, _ = pRegisterWindowMessage.Call(uintptr(unsafe.Pointer(utf16("TaskbarCreated"))))
	addTrayIcon()

	if o.ShowWindow {
		Show()
	}
	var m msg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// Show opens the window, or brings it to the front if it is already open.
func Show() {
	winMu.Lock()
	w := current
	winMu.Unlock()
	if w != nil {
		w.Dispatch(func() {
			hwnd := uintptr(w.Window())
			pShowWindow.Call(hwnd, swRestore)
			pSetForegroundWindow.Call(hwnd)
		})
		return
	}
	select {
	case openReq <- struct{}{}:
	default:
	}
}
