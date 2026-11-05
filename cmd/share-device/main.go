// share-device: share one keyboard and mouse/trackpad between computers on a LAN.
//
//	Mac (has the keyboard):   share-device server -password SECRET -side right
//	Windows:                  share-device client -server 192.168.1.10:24800 -password SECRET
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"

	"sharedevice/internal/capture"
	"sharedevice/internal/client"
	"sharedevice/internal/inject"
	"sharedevice/internal/server"
)

const defaultPort = "24800"

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  share-device server [flags]   run on the computer whose keyboard/mouse you want to share
  share-device client [flags]   run on the computer you want to control

Run "share-device server -h" or "share-device client -h" for flags.
`)
	os.Exit(2)
}

func main() {
	log.SetFlags(log.Ltime)
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "server":
		runServer(os.Args[2:])
	case "client":
		runClient(os.Args[2:])
	default:
		usage()
	}
}

func passwordOrEnv(p string) string {
	if p == "" {
		p = os.Getenv("SHARE_DEVICE_PASSWORD")
	}
	if p == "" {
		log.Fatal("a password is required (-password or SHARE_DEVICE_PASSWORD); keystrokes are encrypted with it")
	}
	return p
}

func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	listen := fs.String("listen", ":"+defaultPort, "address to listen on")
	password := fs.String("password", "", "shared password (or env SHARE_DEVICE_PASSWORD)")
	side := fs.String("side", "right", "which side of this screen the other computer is on: left, right, top, bottom")
	speed := fs.Float64("speed", 1.5, "pointer speed multiplier on the other computer")
	verbose := fs.Bool("v", false, "verbose logging")
	fs.Parse(args)

	cfg := server.Config{Listen: *listen, Password: passwordOrEnv(*password), Side: *side, Speed: *speed, Verbose: *verbose}
	srv, err := server.New(cfg, server.Platform{Freeze: capture.Freeze, Warp: capture.Warp, Bounds: capture.ScreenBounds})
	if err != nil {
		log.Fatal(err)
	}
	if runtime.GOOS == "darwin" && !capture.Trusted(true) {
		log.Print("Accessibility permission is missing. Enable it for this terminal in System Settings > Privacy & Security > Accessibility, then run again.")
		os.Exit(1)
	}
	printAddresses(*listen)
	go func() { log.Fatal(srv.Serve()) }()
	log.Fatal(capture.Run(srv.Handle))
}

func runClient(args []string) {
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	addr := fs.String("server", "", "server address, e.g. 192.168.1.10 or 192.168.1.10:24800")
	password := fs.String("password", "", "shared password (or env SHARE_DEVICE_PASSWORD)")
	host, _ := os.Hostname()
	name := fs.String("name", host, "name shown on the server")
	swap := fs.Bool("swap-cmd-ctrl", runtime.GOOS == "windows", "map Mac Command to Ctrl (Cmd+C = copy) and Ctrl to the Windows key")
	verbose := fs.Bool("v", false, "verbose logging")
	fs.Parse(args)
	if *addr == "" {
		log.Fatal("-server is required")
	}
	if _, _, err := net.SplitHostPort(*addr); err != nil {
		*addr = net.JoinHostPort(*addr, defaultPort)
	}
	inj, err := inject.New()
	if err != nil {
		log.Fatal(err)
	}
	cfg := client.Config{Server: *addr, Password: passwordOrEnv(*password), Name: *name, SwapCmdCtrl: *swap, Verbose: *verbose}
	log.Fatal(client.Run(cfg, inj))
}

func printAddresses(listen string) {
	_, port, _ := net.SplitHostPort(listen)
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ip, ok := a.(*net.IPNet); ok && ip.IP.To4() != nil && !ip.IP.IsLoopback() {
			log.Printf("on the other computer run:  share-device client -server %s:%s -password ...", ip.IP, port)
		}
	}
}
