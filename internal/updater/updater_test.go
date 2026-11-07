package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v0.2.1", "v0.2.0", true},
		{"v0.10.0", "v0.9.9", true},
		{"v0.2.0", "v0.2.0", false},
		{"v0.1.9", "v0.2.0", false},
		{"v0.2.0", "v0.2.0-dev", true},
		{"v0.2.0-beta.1", "v0.2.0", false},
		{"v0.2.0", "dev", true},
		{"garbage", "v0.1.0", false},
	}
	for _, c := range cases {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q,%q)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestParseSums(t *testing.T) {
	h := strings.Repeat("ab", 32)
	s := h + "  Crossdesk-Linux-x64.tar.gz\n" + strings.Repeat("cd", 32) + "  Crossdesk-macOS-universal.zip\n"
	got, err := parseSums(strings.NewReader(s), "Crossdesk-Linux-x64.tar.gz")
	if err != nil || got != h {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := parseSums(strings.NewReader(s), "missing"); err == nil {
		t.Fatal("expected error")
	}
}

func TestZipSlipRejected(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "evil.zip")
	f, _ := os.Create(zp)
	w := zip.NewWriter(f)
	e, _ := w.Create("../../escape.txt")
	e.Write([]byte("x"))
	w.Close()
	f.Close()
	dest := filepath.Join(dir, "out")
	os.Mkdir(dest, 0o755)
	if err := unzipTo(zp, dest); err != nil {
		// rejected is fine too; what matters is nothing escaped
	}
	if _, err := os.Stat(filepath.Join(dir, "escape.txt")); err == nil {
		t.Fatal("zip slip wrote outside dest")
	}
}

func TestUnzipKeepsModes(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "app.zip")
	f, _ := os.Create(zp)
	w := zip.NewWriter(f)
	h := &zip.FileHeader{Name: "Crossdesk.app/Contents/MacOS/crossdesk"}
	h.SetMode(0o755)
	e, _ := w.CreateHeader(h)
	e.Write([]byte("#!/bin/sh\n"))
	w.Close()
	f.Close()
	dest := filepath.Join(dir, "out")
	os.Mkdir(dest, 0o755)
	if err := unzipTo(zp, dest); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(filepath.Join(dest, "Crossdesk.app/Contents/MacOS/crossdesk"))
	if err != nil || st.Mode().Perm()&0o100 == 0 {
		t.Fatalf("binary not executable: %v %v", st, err)
	}
}

func TestExtractFromTarGz(t *testing.T) {
	dir := t.TempDir()
	tp := filepath.Join(dir, "x.tar.gz")
	f, _ := os.Create(tp)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	body := []byte("binary")
	tw.WriteHeader(&tar.Header{Name: "Crossdesk-Linux-x64/crossdesk", Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
	tw.Write(body)
	tw.Close()
	gz.Close()
	f.Close()
	out := filepath.Join(dir, "crossdesk.new")
	if err := extractFromTarGz(tp, "crossdesk", out); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(out); string(b) != "binary" {
		t.Fatalf("got %q", b)
	}
}

// End to end against a fake GitHub: check, download, checksum mismatch.
func TestCheckAndChecksum(t *testing.T) {
	payload := []byte("new build")
	sum := sha256.Sum256(payload)
	var srv *httptest.Server
	good := true
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			json.NewEncoder(w).Encode(release{Tag: "v9.9.9", HTMLURL: "x", Body: "notes", Assets: []asset{
				{Name: AssetName(), URL: srv.URL + "/a"},
				{Name: "SHA256SUMS.txt", URL: srv.URL + "/sums"},
			}})
		case r.URL.Path == "/a":
			w.Write(payload)
		case r.URL.Path == "/sums":
			h := hex.EncodeToString(sum[:])
			if !good {
				h = strings.Repeat("0", 64)
			}
			fmt.Fprintf(w, "%s  %s\n", h, AssetName())
		}
	}))
	defer srv.Close()

	u := New("v0.2.0", t.TempDir(), nil)
	u.apiBase = srv.URL
	if err := u.Check(); err != nil {
		t.Fatal(err)
	}
	st := u.Status()
	if !st.Available || st.Latest != "v9.9.9" || st.Notes != "notes" {
		t.Fatalf("status %+v", st)
	}
	// Verify the download + checksum path without applying.
	want, err := u.expectedSum(srv.URL+"/sums", AssetName())
	if err != nil {
		t.Fatal(err)
	}
	got, err := u.download(asset{URL: srv.URL + "/a"}, filepath.Join(t.TempDir(), "f"))
	if err != nil || got != want {
		t.Fatalf("sum got %s want %s err %v", got, want, err)
	}
	// A bad checksum must abort before anything is applied.
	good = false
	if err := u.install(u.rel); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum error, got %v", err)
	}
	_ = time.Second
}

func TestAutoSetting(t *testing.T) {
	dir := t.TempDir()
	u := New("v1.0.0", dir, nil)
	if !u.Status().Auto {
		t.Fatal("auto check should default on")
	}
	u.SetAuto(false)
	if New("v1.0.0", dir, nil).Status().Auto {
		t.Fatal("setting not persisted")
	}
}

func TestCheckHost(t *testing.T) {
	if checkHost("https://github.com/x/y/releases/download/v1/a.zip") != nil {
		t.Fatal("github should be allowed")
	}
	if checkHost("https://objects.githubusercontent.com/x") != nil {
		t.Fatal("asset CDN should be allowed")
	}
	for _, bad := range []string{"http://github.com/x", "https://evil.example/x", "https://github.com.evil.io/x"} {
		if checkHost(bad) == nil {
			t.Fatalf("%s should be rejected", bad)
		}
	}
}

func TestAfterUpdateEnv(t *testing.T) {
	t.Setenv(RestartEnv, "1")
	if !AfterUpdate() {
		t.Fatal("expected marker")
	}
	if AfterUpdate() {
		t.Fatal("marker must be cleared so child processes don't inherit it")
	}
}
