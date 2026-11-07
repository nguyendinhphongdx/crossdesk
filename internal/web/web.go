// Package web serves the local control UI on 127.0.0.1.
package web

import (
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"

	"crossdesk/internal/agent"
	"crossdesk/internal/autostart"
	"crossdesk/internal/server"
	"crossdesk/internal/updater"
)

//go:embed index.html
var indexHTML []byte

// LogRing keeps the last lines written to the log for the UI.
type LogRing struct {
	mu    sync.Mutex
	lines []string
	part  string
}

func (r *LogRing) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.part + string(p)
	parts := strings.Split(s, "\n")
	r.part = parts[len(parts)-1]
	r.lines = append(r.lines, parts[:len(parts)-1]...)
	if len(r.lines) > 200 {
		r.lines = r.lines[len(r.lines)-200:]
	}
	return len(p), nil
}

func (r *LogRing) Lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.lines...)
}

type Server struct {
	Agent   *agent.Agent
	Token   string
	Logs    *LogRing
	Quit    func()
	Show    func()
	Updater *updater.Updater
	port    string
}

// Handler returns the HTTP handler. Every request must target localhost (no
// DNS rebinding) and API calls must carry the per-run token (no CSRF from
// other web pages).
func (s *Server) Handler(port string) http.Handler {
	s.port = port
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(indexHTML)
	})
	mux.HandleFunc("GET /api/status", s.api(func(r *http.Request) (any, error) {
		var up *updater.Status
		if s.Updater != nil {
			st := s.Updater.Status()
			up = &st
		}
		return struct {
			agent.Status
			Logs      []string        `json:"logs"`
			Autostart bool            `json:"autostart"`
			Update    *updater.Status `json:"update,omitempty"`
		}{s.Agent.Status(), s.Logs.Lines(), autostart.Enabled(), up}, nil
	}))
	mux.HandleFunc("POST /api/password", s.api(func(r *http.Request) (any, error) {
		var b struct{ Password string }
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			return nil, err
		}
		return nil, s.Agent.SetPassword(b.Password)
	}))
	mux.HandleFunc("POST /api/name", s.api(func(r *http.Request) (any, error) {
		var b struct{ Name string }
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			return nil, err
		}
		return nil, s.Agent.SetName(strings.TrimSpace(b.Name))
	}))
	mux.HandleFunc("POST /api/host", s.api(func(r *http.Request) (any, error) {
		var b struct{ ID string }
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			return nil, err
		}
		return nil, s.Agent.SetHost(b.ID)
	}))
	mux.HandleFunc("POST /api/layout", s.api(func(r *http.Request) (any, error) {
		var b struct{ Layout map[string]server.Cell }
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			return nil, err
		}
		return nil, s.Agent.SetLayout(b.Layout)
	}))
	mux.HandleFunc("POST /api/options", s.api(func(r *http.Request) (any, error) {
		var b struct {
			SwapCmdCtrl bool
			Speed       float64
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			return nil, err
		}
		s.Agent.SetOptions(b.SwapCmdCtrl, b.Speed)
		return nil, nil
	}))
	mux.HandleFunc("POST /api/permission", s.api(func(r *http.Request) (any, error) {
		return map[string]bool{"trusted": s.Agent.RequestPermission()}, nil
	}))
	mux.HandleFunc("POST /api/autostart", s.api(func(r *http.Request) (any, error) {
		var b struct{ Enabled bool }
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			return nil, err
		}
		return nil, autostart.Set(b.Enabled)
	}))
	mux.HandleFunc("POST /api/update/check", s.api(func(r *http.Request) (any, error) {
		if s.Updater == nil {
			return nil, nil
		}
		return nil, s.Updater.Check()
	}))
	mux.HandleFunc("POST /api/update/install", s.api(func(r *http.Request) (any, error) {
		if s.Updater == nil {
			return nil, nil
		}
		return nil, s.Updater.Install()
	}))
	mux.HandleFunc("POST /api/update/auto", s.api(func(r *http.Request) (any, error) {
		var b struct{ Enabled bool }
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			return nil, err
		}
		if s.Updater == nil {
			return nil, nil
		}
		return nil, s.Updater.SetAuto(b.Enabled)
	}))
	mux.HandleFunc("POST /api/show", s.api(func(r *http.Request) (any, error) {
		if s.Show != nil {
			s.Show()
		}
		return nil, nil
	}))
	mux.HandleFunc("POST /api/quit", s.api(func(r *http.Request) (any, error) {
		go s.Quit()
		return nil, nil
	}))
	return s.guard(mux)
}

func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, port, err := net.SplitHostPort(r.Host)
		if err != nil || port != s.port || (host != "127.0.0.1" && host != "localhost") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) api(fn func(*http.Request) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Token")), []byte(s.Token)) != 1 {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
		res, err := fn(r)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		if res == nil {
			res = map[string]bool{"ok": true}
		}
		json.NewEncoder(w).Encode(res)
	}
}
