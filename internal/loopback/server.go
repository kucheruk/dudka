// Package loopback serves the engine↔local-UI HTTP API on loopback only.
package loopback

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

// Server is the loopback HTTP surface (P012 /health, P015 /me).
type Server struct {
	mu     sync.RWMutex
	peerID string
	name   string
	mux    *http.ServeMux
}

// New returns a Server bound to the given local identity.
func New(peerID, name string) *Server {
	s := &Server{
		peerID: peerID,
		name:   name,
		mux:    http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	s.mux.HandleFunc("GET /me", s.handleMe)
	return s
}

func (s *Server) handleMe(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"peer_id": s.peerID,
		"name":    s.name,
	})
}

// Handler exposes routes wrapped with a loopback-only remote-addr guard (DUD-NET-130).
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !remoteIsLoopback(r.RemoteAddr) {
			http.Error(w, "forbidden: loopback only\n", http.StatusForbidden)
			return
		}
		s.mux.ServeHTTP(w, r)
	})
}

// Listen opens a TCP listener; addr must be loopback (127.0.0.1 / ::1).
func (s *Server) Listen(addr string) (net.Listener, error) {
	if !isLoopbackAddr(addr) {
		return nil, fmt.Errorf("loopback: refuse non-loopback listen addr %q", addr)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("loopback: listen: %w", err)
	}
	return ln, nil
}

// Serve serves HTTP until the listener closes.
func (s *Server) Serve(ln net.Listener) error {
	err := http.Serve(ln, s.Handler())
	if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}

// FormatReady builds the P012 stdout readiness line.
func FormatReady(peerID, name string) string {
	return fmt.Sprintf("ready peer_id=%s name=%s", peerID, name)
}

func remoteIsLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && ip.IsLoopback()
}

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	switch strings.ToLower(host) {
	case "localhost":
		return true
	default:
		return false
	}
}
