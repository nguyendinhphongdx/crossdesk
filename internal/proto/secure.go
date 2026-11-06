package proto

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Wire format:
//   server -> client (plaintext): "XDSK1" + 16-byte random salt
//   then both sides send frames: uint16 length + AES-256-GCM ciphertext.
// Key = PBKDF2-SHA256(password, salt). Nonce = direction byte + 64-bit counter,
// so frames cannot be replayed, reordered or reflected. A fresh salt per session
// means a recorded session cannot be replayed later.

const magic = "XDSK1"

const (
	dirServer = 1
	dirClient = 2
)

type Conn struct {
	c       net.Conn
	r       *bufio.Reader
	aead    cipher.AEAD
	sendDir byte
	recvDir byte
	wmu     sync.Mutex
	sendCtr uint64
	recvCtr uint64
}

func deriveAEAD(password string, salt []byte) (cipher.AEAD, error) {
	key, err := pbkdf2.Key(sha256.New, password, salt, 200_000, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func ServerHandshake(c net.Conn, password string) (*Conn, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.Write(append([]byte(magic), salt...)); err != nil {
		return nil, err
	}
	c.SetWriteDeadline(time.Time{})
	aead, err := deriveAEAD(password, salt)
	if err != nil {
		return nil, err
	}
	return &Conn{c: c, r: bufio.NewReader(c), aead: aead, sendDir: dirServer, recvDir: dirClient}, nil
}

func ClientHandshake(c net.Conn, password string) (*Conn, error) {
	hdr := make([]byte, len(magic)+16)
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.ReadFull(c, hdr); err != nil {
		return nil, fmt.Errorf("reading handshake: %w", err)
	}
	c.SetReadDeadline(time.Time{})
	if string(hdr[:len(magic)]) != magic {
		return nil, errors.New("not a crossdesk server (bad magic)")
	}
	aead, err := deriveAEAD(password, hdr[len(magic):])
	if err != nil {
		return nil, err
	}
	return &Conn{c: c, r: bufio.NewReader(c), aead: aead, sendDir: dirClient, recvDir: dirServer}, nil
}

func nonce(dir byte, ctr uint64) []byte {
	n := make([]byte, 12)
	n[0] = dir
	binary.BigEndian.PutUint64(n[4:], ctr)
	return n
}

func (s *Conn) Send(m Msg) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	ct := s.aead.Seal(nil, nonce(s.sendDir, s.sendCtr), m.Marshal(), nil)
	s.sendCtr++
	frame := make([]byte, 2, 2+len(ct))
	binary.BigEndian.PutUint16(frame, uint16(len(ct)))
	frame = append(frame, ct...)
	s.c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := s.c.Write(frame)
	return err
}

// ErrAuth is returned when a frame fails to decrypt (wrong password or tampering).
var ErrAuth = errors.New("authentication failed (wrong password?)")

func (s *Conn) Recv(timeout time.Duration) (Msg, error) {
	if timeout > 0 {
		s.c.SetReadDeadline(time.Now().Add(timeout))
	}
	var lb [2]byte
	if _, err := io.ReadFull(s.r, lb[:]); err != nil {
		return Msg{}, err
	}
	ct := make([]byte, binary.BigEndian.Uint16(lb[:]))
	if _, err := io.ReadFull(s.r, ct); err != nil {
		return Msg{}, err
	}
	pt, err := s.aead.Open(nil, nonce(s.recvDir, s.recvCtr), ct, nil)
	if err != nil {
		return Msg{}, ErrAuth
	}
	s.recvCtr++
	return Unmarshal(pt)
}

func (s *Conn) Close() error         { return s.c.Close() }
func (s *Conn) RemoteAddr() net.Addr { return s.c.RemoteAddr() }
