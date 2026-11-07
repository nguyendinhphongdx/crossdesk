// Package proto defines the wire messages exchanged between server and client.
package proto

import (
	"encoding/binary"
	"errors"
)

type Type byte

const (
	// Hello: client -> server on connect (X,Y = screen origin, W,H = screen size in pixels, Name).
	// Server echoes a Hello back to acknowledge (proves the password matched).
	Hello Type = 1
	// Enter: server -> client, the cursor has entered the client's screen at (X,Y).
	Enter Type = 2
	// Leave: server -> client, the cursor went back to the server. Client releases held keys.
	Leave Type = 3
	// MouseMove: absolute position (X,Y) in client screen coordinates (0..W-1, 0..H-1).
	MouseMove Type = 4
	// MouseButton: Button (0 left, 1 right, 2 middle, 3 back, 4 forward), Down.
	MouseButton Type = 5
	// Wheel: X = horizontal, Y = vertical, in 1/120 notch units (Windows WHEEL_DELTA).
	// Positive Y scrolls up (content moves down), positive X scrolls right.
	Wheel Type = 6
	// Key: Key = USB HID usage code (layout independent, positional), Down.
	Key Type = 7
	// Ping: keepalive.
	Ping Type = 8

	// Clipboard sync. Older peers ignore these types.
	// ClipRequest: server -> client, "send me your clipboard if it changed".
	ClipRequest Type = 9
	// ClipBegin: Button = clipboard kind (1 text, 2 PNG), X = total byte size.
	ClipBegin Type = 10
	// ClipChunk: Name carries the next piece of data (up to ClipChunkSize bytes).
	ClipChunk Type = 11
	// ClipEnd: the transfer is complete.
	ClipEnd Type = 12
)

type Msg struct {
	Type   Type
	X, Y   int32
	W, H   int32
	Button byte
	Down   bool
	Key    uint16
	Name   string
}

const headerLen = 1 + 4*4 + 1 + 1 + 2

func (m Msg) Marshal() []byte {
	b := make([]byte, 0, headerLen+len(m.Name))
	b = append(b, byte(m.Type))
	b = binary.BigEndian.AppendUint32(b, uint32(m.X))
	b = binary.BigEndian.AppendUint32(b, uint32(m.Y))
	b = binary.BigEndian.AppendUint32(b, uint32(m.W))
	b = binary.BigEndian.AppendUint32(b, uint32(m.H))
	b = append(b, m.Button)
	if m.Down {
		b = append(b, 1)
	} else {
		b = append(b, 0)
	}
	b = binary.BigEndian.AppendUint16(b, m.Key)
	if m.Type != ClipChunk && len(m.Name) > 255 {
		m.Name = m.Name[:255]
	}
	return append(b, m.Name...)
}

func Unmarshal(b []byte) (Msg, error) {
	if len(b) < headerLen {
		return Msg{}, errors.New("short message")
	}
	var m Msg
	m.Type = Type(b[0])
	m.X = int32(binary.BigEndian.Uint32(b[1:]))
	m.Y = int32(binary.BigEndian.Uint32(b[5:]))
	m.W = int32(binary.BigEndian.Uint32(b[9:]))
	m.H = int32(binary.BigEndian.Uint32(b[13:]))
	m.Button = b[17]
	m.Down = b[18] != 0
	m.Key = binary.BigEndian.Uint16(b[19:])
	m.Name = string(b[headerLen:])
	return m, nil
}
