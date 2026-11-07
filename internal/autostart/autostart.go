// Package autostart registers Crossdesk to start (hidden) when the user logs in:
// a LaunchAgent on macOS, the HKCU Run key on Windows, an XDG autostart entry
// on Linux. Everything is per-user; no admin rights are needed.
package autostart

import (
	"os"
	"path/filepath"
)

// HiddenFlag is passed when started at login so only the tray icon appears.
const HiddenFlag = "-hidden"

func executable() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return p, nil
}
