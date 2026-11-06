package agent

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/sha256"
	"encoding/json"
	"net"
	"time"
)

// Beacon is broadcast on UDP by every device about once a second.
// Anyone can see that a device exists; only devices with the same password can
// verify the MAC and therefore trust its role and shared configuration.
type Beacon struct {
	App       string  `json:"app"`
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	OS        string  `json:"os"`
	Port      int     `json:"port"`
	W         int32   `json:"w"`
	H         int32   `json:"h"`
	Role      string  `json:"role"` // host, client, idle
	Connected bool    `json:"connected"`
	Shared    *Shared `json:"shared,omitempty"`
	Time      int64   `json:"t"`
	MAC       []byte  `json:"mac,omitempty"`
}

const beaconApp = "crossdesk/1"

func groupKey(password string) []byte {
	if password == "" {
		return nil
	}
	k, _ := pbkdf2.Key(sha256.New, password, []byte("crossdesk group v1"), 100_000, 32)
	return k
}

func (b Beacon) sign(key []byte) Beacon {
	b.MAC = nil
	if key == nil {
		return b
	}
	raw, _ := json.Marshal(b)
	m := hmac.New(sha256.New, key)
	m.Write(raw)
	b.MAC = m.Sum(nil)
	return b
}

func (b Beacon) verify(key []byte) bool {
	if key == nil || len(b.MAC) == 0 {
		return false
	}
	got := b.MAC
	b.MAC = nil
	raw, _ := json.Marshal(b)
	m := hmac.New(sha256.New, key)
	m.Write(raw)
	return hmac.Equal(got, m.Sum(nil))
}

// broadcastAddrs returns the limited broadcast address plus every IPv4
// interface's directed broadcast address (macOS often needs the latter).
func broadcastAddrs(port int) []*net.UDPAddr {
	out := []*net.UDPAddr{{IP: net.IPv4bcast, Port: port}}
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagBroadcast == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok || ipn.IP.To4() == nil {
				continue
			}
			ip, mask := ipn.IP.To4(), ipn.Mask
			if len(mask) == 16 {
				mask = mask[12:]
			}
			bc := make(net.IP, 4)
			for i := range 4 {
				bc[i] = ip[i] | ^mask[i]
			}
			out = append(out, &net.UDPAddr{IP: bc, Port: port})
		}
	}
	return out
}

type peer struct {
	Beacon
	IP       net.IP
	LastSeen time.Time
	Trusted  bool
}

func (p *peer) online() bool { return time.Since(p.LastSeen) < 6*time.Second }
