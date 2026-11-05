// Share Device: share one keyboard and mouse/trackpad between computers on a LAN.
// Run it on every machine; it opens a control page in the browser where you
// pick the host and arrange the screens.
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
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"sharedevice/internal/agent"
	"sharedevice/internal/web"
)

const uiPort = 24801

type uiInfo struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
}

func uiInfoPath() string { return filepath.Join(agent.ConfigDir(), "ui.json") }

func main() {
	noBrowser := flag.Bool("no-browser", false, "don't open the control page on start")
	flag.Parse()

	os.MkdirAll(agent.ConfigDir(), 0o700)
	ring := &web.LogRing{}
	writers := []io.Writer{ring}
	if f, err := os.OpenFile(filepath.Join(agent.ConfigDir(), "share-device.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600); err == nil {
		writers = append(writers, f)
	}
	if fi, err := os.Stderr.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		writers = append(writers, os.Stderr)
	}
	log.SetOutput(io.MultiWriter(writers...))
	log.SetFlags(log.Ltime)

	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(uiPort))
	if err != nil {
		// Already running: just bring up its page.
		if b, err := os.ReadFile(uiInfoPath()); err == nil {
			var info uiInfo
			if json.Unmarshal(b, &info) == nil {
				openBrowser(pageURL(info))
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

	a, err := agent.New()
	if err != nil {
		log.Fatal(err)
	}
	quit := func() {
		log.Print("quitting")
		a.Shutdown()
		time.Sleep(300 * time.Millisecond)
		os.Exit(0)
	}
	ws := &web.Server{Agent: a, Token: info.Token, Logs: ring, Quit: quit}
	go func() {
		srv := &http.Server{Handler: ws.Handler(strconv.Itoa(uiPort)), ReadHeaderTimeout: 5 * time.Second}
		log.Fatal(srv.Serve(ln))
	}()

	url := pageURL(info)
	log.Printf("control page: http://127.0.0.1:%d/", uiPort)
	if !*noBrowser {
		openBrowser(url)
	}
	log.Fatal(a.Run())
}

// The token travels in the URL fragment, which browsers never send to servers.
func pageURL(i uiInfo) string {
	return "http://127.0.0.1:" + strconv.Itoa(i.Port) + "/#token=" + i.Token
}

func openBrowser(url string) {
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
		log.Printf("open the control page manually: %s (%v)", url, err)
	}
}
