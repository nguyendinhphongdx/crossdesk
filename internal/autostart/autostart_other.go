//go:build !darwin && !windows

package autostart

import (
	"os"
	"path/filepath"
	"strings"
)

func desktopPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", "crossdesk.desktop")
}

func Enabled() bool {
	_, err := os.Stat(desktopPath())
	return err == nil
}

func Set(on bool) error {
	if !on {
		err := os.Remove(desktopPath())
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	exe, err := executable()
	if err != nil {
		return err
	}
	quoted := `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
	entry := "[Desktop Entry]\nType=Application\nName=Crossdesk\nIcon=crossdesk\nExec=" + quoted + " " + HiddenFlag +
		"\nX-GNOME-Autostart-enabled=true\nTerminal=false\n"
	if err := os.MkdirAll(filepath.Dir(desktopPath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(desktopPath(), []byte(entry), 0o644)
}
