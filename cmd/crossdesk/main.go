// Crossdesk: share one keyboard and mouse/trackpad between computers on a LAN.
// Run it on every machine. It lives in the menu bar / notification area and
// opens a window where you pick the host and arrange the screens.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"crossdesk/internal/agent"
	"crossdesk/internal/ui"
	"crossdesk/internal/updater"
	"crossdesk/internal/web"
)

const uiPort = 24801

// version is set at build time: -ldflags "-X main.version=v1.2.3".
var version = "dev"

type uiInfo struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
}

func uiInfoPath() string { return filepath.Join(agent.ConfigDir(), "ui.json") }

func main() {
	hidden := flag.Bool("hidden", false, "start in the background (menu bar / tray icon only)")
	noBrowser := flag.Bool("no-browser", false, "same as -hidden")
	flag.Parse()
	afterUpdate := updater.AfterUpdate()
	updater.Cleanup()

	os.MkdirAll(agent.ConfigDir(), 0o700)
	ring := &web.LogRing{}
	writers := []io.Writer{ring}
	if f, err := os.OpenFile(filepath.Join(agent.ConfigDir(), "crossdesk.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600); err == nil {
		writers = append(writers, f)
	}
	if fi, err := os.Stderr.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		writers = append(writers, os.Stderr)
	}
	log.SetOutput(io.MultiWriter(writers...))
	log.SetFlags(log.Ltime)

	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(uiPort))
	// After an update the previous version may still be shutting down.
	for i := 0; err != nil && afterUpdate && i < 40; i++ {
		time.Sleep(250 * time.Millisecond)
		ln, err = net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(uiPort))
	}
	if err != nil {
		// Already running: ask it to bring its window to the front.
		if b, err := os.ReadFile(uiInfoPath()); err == nil {
			var info uiInfo
			if json.Unmarshal(b, &info) == nil {
				if !*hidden && !*noBrowser {
					askToShow(info)
				}
				return
			}
		}
		log.Fatalf("UI port %d is busy: %v", uiPort, err)
	}

	tok := make([]byte, 16)
	rand.Read(tok)
	info := uiInfo{Port: uiPort, Token: hex.EncodeToString(tok)}
	if b, err := json.Marshal(info); err == nil {
		os.WriteFile(uiInfoPath(), b, 0o600)
	}

	log.Printf("Crossdesk %s (%s/%s)", version, runtime.GOOS, runtime.GOARCH)
	a, err := agent.New(version)
	if err != nil {
		log.Fatal(err)
	}
	quit := func() {
		log.Print("quitting")
		a.Shutdown()
		time.Sleep(300 * time.Millisecond)
		os.Exit(0)
	}
	up := updater.New(version, agent.ConfigDir(), quit)
	up.Start()
	ws := &web.Server{Agent: a, Token: info.Token, Logs: ring, Quit: quit, Show: ui.Show, Updater: up}
	go func() {
		srv := &http.Server{Handler: ws.Handler(strconv.Itoa(uiPort)), ReadHeaderTimeout: 5 * time.Second}
		log.Fatal(srv.Serve(ln))
	}()

	log.Printf("control page: http://127.0.0.1:%d/", uiPort)
	go func() { log.Fatal(a.Run()) }()
	// The native shell owns the main thread until the user quits.
	ui.Run(ui.Options{URL: pageURL(info), ShowWindow: !*hidden && !*noBrowser, OnQuit: quit})
}

// askToShow tells the running instance to show its window.
func askToShow(info uiInfo) {
	req, _ := http.NewRequest("POST", "http://127.0.0.1:"+strconv.Itoa(info.Port)+"/api/show", nil)
	req.Header.Set("X-Token", info.Token)
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return
		}
	}
	ui.OpenBrowser(pageURL(info))
}

// The token travels in the URL fragment, which browsers never send to servers.
func pageURL(i uiInfo) string {
	return "http://127.0.0.1:" + strconv.Itoa(i.Port) + "/#token=" + i.Token
}
