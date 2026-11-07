package clipboard

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "clipboard_darwin.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

func read() (Content, bool) {
	var p unsafe.Pointer
	var n C.long
	kind := C.cdClipboardRead(&p, &n)
	if kind == 0 {
		return Content{}, false
	}
	defer C.free(p)
	return Content{Kind: Kind(kind), Data: C.GoBytes(p, C.int(n))}, true
}

func write(c Content) error {
	if len(c.Data) == 0 {
		return nil
	}
	p := C.CBytes(c.Data)
	defer C.free(p)
	if C.cdClipboardWrite(C.int(c.Kind), p, C.long(len(c.Data))) != 0 {
		return errors.New("cannot write the clipboard")
	}
	return nil
}

func problem() string { return "" }
