package autostart

import (
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

const label = "local.crossdesk.app"

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist")
}

func Enabled() bool {
	_, err := os.Stat(plistPath())
	return err == nil
}

func Set(on bool) error {
	if !on {
		err := os.Remove(plistPath())
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	exe, err := executable()
	if err != nil {
		return err
	}
	// Opened straight from Downloads, macOS runs the app from a random
	// read-only copy that disappears; registering that path would break.
	if strings.Contains(exe, "/AppTranslocation/") {
		return errors.New("hãy kéo Crossdesk vào thư mục Applications rồi mở lại, sau đó mới bật tự khởi động")
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key><string>%s</string>
	<key>ProgramArguments</key>
	<array><string>%s</string><string>%s</string></array>
	<key>RunAtLoad</key><true/>
	<key>ProcessType</key><string>Interactive</string>
	<key>LimitLoadToSessionType</key><string>Aqua</string>
</dict>
</plist>
`, label, html.EscapeString(exe), HiddenFlag)
	if err := os.MkdirAll(filepath.Dir(plistPath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(plistPath(), []byte(plist), 0o644)
}
