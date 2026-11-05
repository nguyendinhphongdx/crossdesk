package agent

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"sharedevice/internal/server"
)

// Shared is the group-wide configuration, replicated between all devices that
// share the password. The copy with the highest (Epoch, Author) wins.
type Shared struct {
	Epoch  uint64                 `json:"epoch"`
	Author string                 `json:"author"`
	Host   string                 `json:"host"` // device ID of the host, "" = nobody
	Layout map[string]server.Cell `json:"layout"`
}

func (s Shared) newerThan(o Shared) bool {
	if s.Epoch != o.Epoch {
		return s.Epoch > o.Epoch
	}
	return s.Author > o.Author
}

func (s Shared) clone() Shared {
	c := s
	c.Layout = make(map[string]server.Cell, len(s.Layout))
	for k, v := range s.Layout {
		c.Layout[k] = v
	}
	return c
}

// Config is what this device persists to disk.
type Config struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Password    string  `json:"password"`
	SwapCmdCtrl bool    `json:"swapCmdCtrl"`
	Speed       float64 `json:"speed"`
	Shared      Shared  `json:"shared"`
}

func ConfigDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = "."
	}
	return filepath.Join(d, "share-device")
}

func configPath() string { return filepath.Join(ConfigDir(), "config.json") }

func randomID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func loadConfig() *Config {
	c := &Config{SwapCmdCtrl: true, Speed: 1.5}
	if b, err := os.ReadFile(configPath()); err == nil {
		json.Unmarshal(b, c)
	}
	if c.ID == "" {
		c.ID = randomID()
	}
	if c.Name == "" {
		c.Name, _ = os.Hostname()
	}
	if c.Speed <= 0 {
		c.Speed = 1.5
	}
	if c.Shared.Layout == nil {
		c.Shared.Layout = map[string]server.Cell{}
	}
	return c
}

func (c *Config) save() error {
	if err := os.MkdirAll(ConfigDir(), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, configPath())
}
