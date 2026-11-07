package ui

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework WebKit
#include <stdlib.h>
#include "shell_darwin.h"
*/
import "C"

import (
	_ "embed"
	"runtime"
	"unsafe"
)

//go:embed assets/menubar@2x.png
var menubarIcon []byte

var onQuit func()

func init() {
	// Cocoa must run on the process's main thread.
	runtime.LockOSThread()
}

//export goQuit
func goQuit() {
	if onQuit != nil {
		go onQuit()
	}
}

// Run starts the menu-bar app and blocks forever. Call it from main().
func Run(o Options) {
	onQuit = o.OnQuit
	url := C.CString(o.URL)
	icon := C.CBytes(menubarIcon)
	defer C.free(unsafe.Pointer(url))
	defer C.free(icon)
	show := 0
	if o.ShowWindow {
		show = 1
	}
	C.runShell(url, icon, C.int(len(menubarIcon)), C.int(show))
}

// Show brings the window to the front (safe from any goroutine).
func Show() { C.showWindowAsync() }
