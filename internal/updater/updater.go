// Package updater keeps Crossdesk up to date from GitHub Releases: it checks
// for a newer release, downloads the build for this OS/CPU, verifies it
// against the release's SHA256SUMS.txt, swaps it in place and restarts.
package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	Repo          = "nguyendinhphongdx/crossdesk"
	checkInterval = 6 * time.Hour
	maxDownload   = 300 << 20
)

// RestartEnv is set for the relaunched process so it waits for the old one
// to release its ports instead of thinking another copy is already running.
// An environment variable (not a flag) so any version can be started safely.
const RestartEnv = "CROSSDESK_AFTER_UPDATE"

func restartEnv() []string { return append(os.Environ(), RestartEnv+"=1") }

// AfterUpdate reports (and clears) the restart marker.
func AfterUpdate() bool {
	v := os.Getenv(RestartEnv) == "1"
	os.Unsetenv(RestartEnv)
	return v
}

type Status struct {
	Current    string    `json:"current"`
	Latest     string    `json:"latest,omitempty"`
	Available  bool      `json:"available"`
	Notes      string    `json:"notes,omitempty"`
	URL        string    `json:"url,omitempty"`
	State      string    `json:"state"` // idle, checking, downloading, installing, error
	Progress   int       `json:"progress"`
	Error      string    `json:"error,omitempty"`
	CheckedAt  time.Time `json:"checkedAt,omitzero"`
	Auto       bool      `json:"auto"`
	CanInstall bool      `json:"canInstall"`
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type release struct {
	Tag     string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Body    string  `json:"body"`
	Draft   bool    `json:"draft"`
	Pre     bool    `json:"prerelease"`
	Assets  []asset `json:"assets"`
}

type Updater struct {
	mu      sync.Mutex
	st      Status
	rel     *release
	dir     string
	restart func()
	client  *http.Client
	apiBase string // overridable in tests
}

type settings struct {
	Auto bool `json:"auto"`
}

// New creates an updater. restart is called after the new version has been
// put in place and launched; it must shut this process down.
func New(current, configDir string, restart func()) *Updater {
	u := &Updater{
		dir:     filepath.Join(configDir, "update"),
		restart: restart,
		client:  &http.Client{Timeout: 10 * time.Minute},
		apiBase: "https://api.github.com",
	}
	u.st = Status{Current: current, State: "idle", Auto: true}
	if b, err := os.ReadFile(filepath.Join(configDir, "update.json")); err == nil {
		var s settings
		if json.Unmarshal(b, &s) == nil {
			u.st.Auto = s.Auto
		}
	}
	return u
}

func (u *Updater) settingsPath() string { return filepath.Join(filepath.Dir(u.dir), "update.json") }

func (u *Updater) Status() Status {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.st
}

func (u *Updater) SetAuto(on bool) error {
	u.mu.Lock()
	u.st.Auto = on
	u.mu.Unlock()
	b, _ := json.Marshal(settings{Auto: on})
	if err := os.MkdirAll(filepath.Dir(u.settingsPath()), 0o700); err != nil {
		return err
	}
	return os.WriteFile(u.settingsPath(), b, 0o600)
}

// Start checks shortly after launch and then periodically, if enabled.
func (u *Updater) Start() {
	go func() {
		time.Sleep(15 * time.Second)
		for {
			if u.Status().Auto {
				if err := u.Check(); err != nil {
					log.Printf("update check: %v", err)
				}
			}
			time.Sleep(checkInterval)
		}
	}()
}

func (u *Updater) set(f func(*Status)) {
	u.mu.Lock()
	f(&u.st)
	u.mu.Unlock()
}

// Check asks GitHub for the latest release.
func (u *Updater) Check() error {
	u.mu.Lock()
	if u.st.State == "downloading" || u.st.State == "installing" {
		u.mu.Unlock()
		return nil
	}
	u.st.State, u.st.Error = "checking", ""
	u.mu.Unlock()

	rel, err := u.fetchLatest()
	if err != nil {
		u.set(func(s *Status) { s.State, s.Error = "error", "Không kiểm tra được bản mới: "+err.Error() })
		return err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.rel = rel
	u.st.State = "idle"
	u.st.CheckedAt = time.Now()
	u.st.Latest = rel.Tag
	u.st.URL = rel.HTMLURL
	u.st.Notes = trimNotes(rel.Body)
	u.st.Available = Newer(rel.Tag, u.st.Current)
	_, ok := findAsset(rel, AssetName())
	u.st.CanInstall = u.st.Available && ok && supported()
	if u.st.Available {
		log.Printf("update available: %s -> %s", u.st.Current, rel.Tag)
	}
	return nil
}

func (u *Updater) fetchLatest() (*release, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", u.apiBase+"/repos/"+Repo+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "crossdesk-updater")
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub trả về %s", resp.Status)
	}
	var rel release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.Tag == "" {
		return nil, errors.New("release không hợp lệ")
	}
	return &rel, nil
}

func trimNotes(s string) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > 1500 {
		s = string(r[:1500]) + "…"
	}
	return s
}

// AssetName is the release file for this OS/CPU.
func AssetName() string {
	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	switch runtime.GOOS {
	case "darwin":
		return "Crossdesk-macOS-universal.zip"
	case "windows":
		return "Crossdesk-Windows-" + arch + ".exe"
	case "linux":
		return "Crossdesk-Linux-" + arch + ".tar.gz"
	}
	return ""
}

func findAsset(rel *release, name string) (asset, bool) {
	for _, a := range rel.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return asset{}, false
}

// Install downloads, verifies and applies the latest release in the
// background, then restarts the app. Progress is reported via Status.
func (u *Updater) Install() error {
	u.mu.Lock()
	if !u.st.CanInstall || u.rel == nil {
		u.mu.Unlock()
		return errors.New("không có bản cập nhật để cài")
	}
	if u.st.State == "downloading" || u.st.State == "installing" {
		u.mu.Unlock()
		return nil
	}
	rel := u.rel
	u.st.State, u.st.Progress, u.st.Error = "downloading", 0, ""
	u.mu.Unlock()

	go func() {
		if err := u.install(rel); err != nil {
			log.Printf("update failed: %v", err)
			u.set(func(s *Status) { s.State, s.Error = "error", err.Error() })
		}
	}()
	return nil
}

func (u *Updater) install(rel *release) error {
	name := AssetName()
	a, ok := findAsset(rel, name)
	if !ok {
		return fmt.Errorf("bản %s không có file %s", rel.Tag, name)
	}
	sums, ok := findAsset(rel, "SHA256SUMS.txt")
	if !ok {
		return errors.New("bản phát hành thiếu SHA256SUMS.txt")
	}
	want, err := u.expectedSum(sums.URL, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(u.dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(u.dir, name)
	got, err := u.download(a, path)
	if err != nil {
		return fmt.Errorf("tải bản mới: %w", err)
	}
	if !strings.EqualFold(got, want) {
		os.Remove(path)
		return errors.New("checksum không khớp — đã huỷ cập nhật")
	}
	u.set(func(s *Status) { s.State, s.Progress = "installing", 100 })
	log.Printf("installing %s", rel.Tag)
	relaunch, err := apply(path)
	if err != nil {
		return err
	}
	os.Remove(path)
	if err := relaunch(); err != nil {
		return fmt.Errorf("đã cài bản mới nhưng không mở lại được, hãy mở Crossdesk thủ công: %w", err)
	}
	log.Printf("restarting into %s", rel.Tag)
	if u.restart != nil {
		u.restart()
	}
	return nil
}

// Only follow links that GitHub hands out for release assets.
func checkHost(raw string) error {
	p, err := url.Parse(raw)
	if err != nil || p.Scheme != "https" || (p.Host != "github.com" && !strings.HasSuffix(p.Host, ".githubusercontent.com")) {
		return fmt.Errorf("đường dẫn tải không hợp lệ: %s", raw)
	}
	return nil
}

func (u *Updater) get(raw string) (*http.Response, error) {
	if u.apiBase == "https://api.github.com" {
		if err := checkHost(raw); err != nil {
			return nil, err
		}
	}
	req, _ := http.NewRequest("GET", raw, nil)
	req.Header.Set("User-Agent", "crossdesk-updater")
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s: %s", raw, resp.Status)
	}
	return resp, nil
}

func (u *Updater) expectedSum(sumsURL, name string) (string, error) {
	resp, err := u.get(sumsURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return parseSums(io.LimitReader(resp.Body, 1<<20), name)
}

// parseSums finds name in sha256sum output ("<hex>  <file>").
func parseSums(r io.Reader, name string) (string, error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name && len(f[0]) == 64 {
			return f[0], nil
		}
	}
	return "", fmt.Errorf("SHA256SUMS.txt không có %s", name)
}

type progressWriter struct {
	u     *Updater
	total int64
	n     int64
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.n += int64(len(b))
	if p.total > 0 {
		pct := int(p.n * 100 / p.total)
		p.u.set(func(s *Status) { s.Progress = min(pct, 99) })
	}
	return len(b), nil
}

func (u *Updater) download(a asset, path string) (string, error) {
	resp, err := u.get(a.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	total := resp.ContentLength
	if total <= 0 {
		total = a.Size
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h, &progressWriter{u: u, total: total}), io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return "", err
	}
	if n > maxDownload {
		return "", errors.New("file quá lớn")
	}
	return hex.EncodeToString(h.Sum(nil)), f.Sync()
}

// Newer reports whether tag a is a newer version than b. Unparseable
// versions (e.g. "dev") count as 0.0.0.
func Newer(a, b string) bool {
	va, pa := parse(a)
	vb, pb := parse(b)
	for i := range va {
		if va[i] != vb[i] {
			return va[i] > vb[i]
		}
	}
	// Same numbers: a release beats a pre-release of it.
	return pa == "" && pb != ""
}

func parse(v string) ([3]int, string) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, pre, _ := strings.Cut(v, "-")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, "invalid"
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, "invalid"
		}
		out[i] = n
	}
	return out, pre
}

// executable is a variable so tests can point it at a fake install.
var executable = func() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return p, nil
}
