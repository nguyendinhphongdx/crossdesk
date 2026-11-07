package updater

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestApplyLinuxAndRelaunch(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "bin", "crossdesk")
	os.MkdirAll(filepath.Dir(exe), 0o755)
	os.WriteFile(exe, []byte("old"), 0o755)
	marker := filepath.Join(dir, "launched")
	script := []byte("#!/bin/sh\necho \"$" + RestartEnv + "\" > '" + marker + "'\n")

	tp := filepath.Join(dir, "u.tar.gz")
	f, _ := os.Create(tp)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "Crossdesk-Linux-x64/crossdesk", Mode: 0o755, Size: int64(len(script)), Typeflag: tar.TypeReg})
	tw.Write(script)
	tw.Close()
	gz.Close()
	f.Close()

	orig := executable
	executable = func() (string, error) { return exe, nil }
	defer func() { executable = orig }()

	relaunch, err := apply(tp)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); !strings.Contains(string(b), RestartEnv) {
		t.Fatal("binary not replaced")
	}
	if err := relaunch(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if b, err := os.ReadFile(marker); err == nil && strings.TrimSpace(string(b)) == "1" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("relaunched binary did not see the restart marker")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
