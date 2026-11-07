package clipboard

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Linux has no clipboard API outside the display server, so the standard
// tools are used: wl-clipboard on Wayland, xclip on X11.

func wayland() bool { return os.Getenv("WAYLAND_DISPLAY") != "" }

func tool() string {
	if wayland() {
		return "wl-paste"
	}
	return "xclip"
}

func problem() string {
	if _, err := exec.LookPath(tool()); err != nil {
		if wayland() {
			return "Cần cài wl-clipboard (ví dụ: sudo apt install wl-clipboard)"
		}
		return "Cần cài xclip (ví dụ: sudo apt install xclip)"
	}
	return ""
}

func run(stdin []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return out.Bytes(), err
	case <-time.After(3 * time.Second):
		cmd.Process.Kill()
		return nil, errors.New(name + " timed out")
	}
}

func read() (Content, bool) {
	if problem() != "" {
		return Content{}, false
	}
	var types []byte
	if wayland() {
		types, _ = run(nil, "wl-paste", "--list-types")
	} else {
		types, _ = run(nil, "xclip", "-selection", "clipboard", "-t", "TARGETS", "-o")
	}
	t := string(types)
	hasText := strings.Contains(t, "text/plain") || strings.Contains(t, "UTF8_STRING")
	if !hasText && strings.Contains(t, "image/png") {
		var b []byte
		if wayland() {
			b, _ = run(nil, "wl-paste", "--type", "image/png")
		} else {
			b, _ = run(nil, "xclip", "-selection", "clipboard", "-t", "image/png", "-o")
		}
		if len(b) > 0 {
			return Content{Kind: PNG, Data: b}, true
		}
		return Content{}, false
	}
	var b []byte
	if wayland() {
		b, _ = run(nil, "wl-paste", "--no-newline", "--type", "text/plain;charset=utf-8")
	} else {
		b, _ = run(nil, "xclip", "-selection", "clipboard", "-t", "UTF8_STRING", "-o")
	}
	if len(b) == 0 {
		return Content{}, false
	}
	return Content{Kind: Text, Data: b}, true
}

func write(c Content) error {
	if p := problem(); p != "" {
		return errors.New(p)
	}
	mime := "text/plain;charset=utf-8"
	xtype := "UTF8_STRING"
	if c.Kind == PNG {
		mime, xtype = "image/png", "image/png"
	}
	// Both tools fork into the background to keep serving the selection.
	if wayland() {
		_, err := run(c.Data, "wl-copy", "--type", mime)
		return err
	}
	_, err := run(c.Data, "xclip", "-selection", "clipboard", "-t", xtype, "-i")
	return err
}
