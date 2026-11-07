package proto

import "errors"

const (
	// ClipChunkSize keeps every encrypted frame under the 64 KiB frame limit.
	ClipChunkSize = 48 << 10
	// MaxClip is the largest clipboard payload that is synced.
	MaxClip = 10 << 20
)

// ClipMessages splits clipboard data into Begin, Chunk..., End messages.
func ClipMessages(kind byte, data []byte) []Msg {
	out := []Msg{{Type: ClipBegin, Button: kind, X: int32(len(data))}}
	for off := 0; off < len(data); off += ClipChunkSize {
		end := min(off+ClipChunkSize, len(data))
		out = append(out, Msg{Type: ClipChunk, Name: string(data[off:end])})
	}
	return append(out, Msg{Type: ClipEnd})
}

// ClipAssembler rebuilds a clipboard transfer from its messages.
type ClipAssembler struct {
	active bool
	kind   byte
	want   int
	buf    []byte
}

var errClip = errors.New("invalid clipboard transfer")

// Feed consumes one clipboard message. It returns done=true with the data
// once ClipEnd arrives. Non-clipboard messages are ignored.
func (a *ClipAssembler) Feed(m Msg) (kind byte, data []byte, done bool, err error) {
	switch m.Type {
	case ClipBegin:
		if m.X < 0 || m.X > MaxClip {
			a.active = false
			return 0, nil, false, errClip
		}
		a.active, a.kind, a.want = true, m.Button, int(m.X)
		a.buf = make([]byte, 0, a.want)
	case ClipChunk:
		if !a.active || len(a.buf)+len(m.Name) > a.want {
			a.active = false
			return 0, nil, false, errClip
		}
		a.buf = append(a.buf, m.Name...)
	case ClipEnd:
		if !a.active || len(a.buf) != a.want {
			a.active = false
			return 0, nil, false, errClip
		}
		a.active = false
		data, a.buf = a.buf, nil
		return a.kind, data, true, nil
	}
	return 0, nil, false, nil
}
