// Package loopback serves the engine↔local-UI HTTP API on loopback only.
package loopback

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"dudka/internal/discovery"
)

// Server is the loopback HTTP surface (health/me/nick/peers/status/scan).
type Server struct {
	mu          sync.RWMutex
	peerID      string
	name        string
	persistName func(string) error
	peers       *discovery.PeerStore
	status      func() discovery.Status
	scan        func(context.Context, discovery.ScanRequest) (discovery.ScanResult, error)
	mux         *http.ServeMux
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
	s.mux.HandleFunc("POST /nick", s.handleNick)
	s.mux.HandleFunc("GET /peers", s.handlePeers)
	s.mux.HandleFunc("GET /status", s.handleStatus)
	s.mux.HandleFunc("POST /scan", s.handleScan)
	return s
}

// SetPersistName registers an optional hook to save the nick (e.g. to data-dir).
func (s *Server) SetPersistName(fn func(string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.persistName = fn
}

// SetPeers wires the discovery peer table into GET /peers.
func (s *Server) SetPeers(store *discovery.PeerStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.peers = store
}

// SetStatusProvider wires discovery proto status into GET /status (P023).
func (s *Server) SetStatusProvider(fn func() discovery.Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = fn
}

// SetScanProvider wires discovery Scan into POST /scan (P024).
func (s *Server) SetScanProvider(fn func(context.Context, discovery.ScanRequest) (discovery.ScanResult, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scan = fn
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	fn := s.scan
	s.mu.RUnlock()
	if fn == nil {
		http.Error(w, "scan unavailable\n", http.StatusServiceUnavailable)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	var req discovery.ScanRequest
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid json\n", http.StatusBadRequest)
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), discovery.DefaultScanTimeout)
	defer cancel()
	res, err := fn(ctx, req)
	if err != nil {
		http.Error(w, err.Error()+"\n", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handlePeers(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	store := s.peers
	s.mu.RUnlock()
	peers := []discovery.Peer{}
	if store != nil {
		peers = store.List()
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"peers": peers})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	fn := s.status
	s.mu.RUnlock()
	st := discovery.Status{
		ProtoMajor:   discovery.DefaultProtoMajor,
		Incompatible: []discovery.IncompatiblePeer{},
	}
	if fn != nil {
		st = fn()
		if st.Incompatible == nil {
			st.Incompatible = []discovery.IncompatiblePeer{}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(st)
}

func (s *Server) handleMe(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeMeJSON(w, s.peerID, s.name)
}

func (s *Server) handleNick(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json\n", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "name is required\n", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	persist := s.persistName
	s.name = name
	peerID := s.peerID
	s.mu.Unlock()

	if persist != nil {
		if err := persist(name); err != nil {
			http.Error(w, "persist failed\n", http.StatusInternalServerError)
			return
		}
	}
	writeMeJSON(w, peerID, name)
}

func writeMeJSON(w http.ResponseWriter, peerID, name string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"peer_id": peerID,
		"name":    name,
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
