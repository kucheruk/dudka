// Package loopback serves the engine↔local-UI HTTP API on loopback only.
package loopback

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// Server is the minimal loopback HTTP surface (P012: /health).
type Server struct {
	mux *http.ServeMux
}

// New returns a Server with /health registered.
func New() *Server {
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	return s
}

// Handler exposes the HTTP routes (for tests).
func (s *Server) Handler() http.Handler { return s.mux }

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
	err := http.Serve(ln, s.mux)
	if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}

// FormatReady builds the P012 stdout readiness line.
func FormatReady(peerID, name string) string {
	return fmt.Sprintf("ready peer_id=%s name=%s", peerID, name)
}

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// bare host without port
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
	// Allow explicit loopback hostnames only.
	switch strings.ToLower(host) {
	case "localhost":
		return true
	default:
		return false
	}
}
