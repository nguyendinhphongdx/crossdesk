//go:build darwin || linux

package autostart

import (
	"os"
	"strings"
	"testing"
)

func TestToggle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	if Enabled() {
		t.Fatal("should start disabled")
	}
	if err := Set(true); err != nil {
		t.Fatal(err)
	}
	if !Enabled() {
		t.Fatal("expected enabled")
	}
	path := plistOrDesktop()
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), HiddenFlag) {
		t.Fatalf("entry lacks %s:\n%s", HiddenFlag, b)
	}
	if err := Set(false); err != nil || Enabled() {
		t.Fatal("expected disabled", err)
	}
	if err := Set(false); err != nil {
		t.Fatal("disabling twice must not fail:", err)
	}
}
