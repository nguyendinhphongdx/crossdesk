// Package clipboard reads and writes the system clipboard (text and images).
package clipboard

import "crypto/sha256"

type Kind byte

const (
	Text Kind = 1 // UTF-8
	PNG  Kind = 2
)

type Content struct {
	Kind Kind
	Data []byte
}

func (c Content) Hash() [32]byte {
	h := sha256.New()
	h.Write([]byte{byte(c.Kind)})
	h.Write(c.Data)
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// IO is the clipboard of this machine. Read returns ok=false when the
// clipboard holds nothing we can share.
type IO interface {
	Read() (Content, bool)
	Write(Content) error
}

// System is the real clipboard.
var System IO = system{}

type system struct{}

func (system) Read() (Content, bool) { return read() }
func (system) Write(c Content) error { return write(c) }

// Problem explains why clipboard sharing cannot work here ("" if it can).
func Problem() string { return problem() }
