package updater

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeBundleZip(t *testing.T, zp, script string) {
	f, err := os.Create(zp)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	h := &zip.FileHeader{Name: "Crossdesk.app/Contents/MacOS/crossdesk"}
	h.SetMode(0o755)
	e, _ := w.CreateHeader(h)
	e.Write([]byte(script))
	pl, _ := w.Create("Crossdesk.app/Contents/Info.plist")
	pl.Write([]byte("<plist>new</plist>"))
	w.Close()
	f.Close()
}

func TestApplyBundleAndRelaunch(t *testing.T) {
	dir := t.TempDir()
	apps := filepath.Join(dir, "Applications")
	oldBin := filepath.Join(apps, "Crossdesk.app", "Contents", "MacOS", "crossdesk")
	os.MkdirAll(filepath.Dir(oldBin), 0o755)
	os.WriteFile(oldBin, []byte("old"), 0o755)

	marker := filepath.Join(dir, "launched")
	zp := filepath.Join(dir, "update.zip")
	writeBundleZip(t, zp, "#!/bin/sh\necho \"$"+RestartEnv+"\" > '"+marker+"'\n")

	orig := executable
	executable = func() (string, error) { return oldBin, nil }
	defer func() { executable = orig }()

	relaunch, err := apply(zp)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(apps, "Crossdesk.app", "Contents", "Info.plist")); string(b) != "<plist>new</plist>" {
		t.Fatal("bundle was not replaced")
	}
	if _, err := os.Stat(filepath.Join(apps, "Crossdesk.app.old")); err == nil {
		t.Fatal("old bundle left behind")
	}
	if left, _ := filepath.Glob(filepath.Join(apps, ".crossdesk-update-*")); len(left) != 0 {
		t.Fatal("temp dir left behind")
	}
	if err := relaunch(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		b, err := os.ReadFile(marker)
		if err == nil && strings.TrimSpace(string(b)) == "1" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("new version did not start with %s=1 (got %q, %v)", RestartEnv, b, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestApplyRefusesTranslocated(t *testing.T) {
	orig := executable
	executable = func() (string, error) {
		return "/private/var/folders/x/T/AppTranslocation/ABC/d/Crossdesk.app/Contents/MacOS/crossdesk", nil
	}
	defer func() { executable = orig }()
	if _, err := apply("unused.zip"); err == nil || !strings.Contains(err.Error(), "Applications") {
		t.Fatalf("expected translocation error, got %v", err)
	}
}
