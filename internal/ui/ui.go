// Package ui is the native shell around the control panel: a menu-bar (macOS)
// or notification-area (Windows) icon, and an app window that hosts the
// control page. On other systems it falls back to the default browser.
package ui

import (
	"log"
	"os/exec"
	"runtime"
)

// Options configures the shell.
type Options struct {
	URL        string // control page, including the #token fragment
	ShowWindow bool   // open the window right away (false when started at login)
	OnQuit     func() // called when the user picks Quit
}

// OpenBrowser opens url in the user's default browser.
func OpenBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("open %s manually (%v)", url, err)
	}
}
