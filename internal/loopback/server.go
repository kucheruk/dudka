// Package loopback serves the engine↔local-UI HTTP API on loopback only.
package loopback

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"dudka/internal/agent"
	"dudka/internal/chat"
	"dudka/internal/discovery"
)

// Server is the loopback HTTP surface (health/me/nick/peers/status/scan/send/messages/files).
type Server struct {
	mu          sync.RWMutex
	peerID      string
	name        string
	persistName func(string) error
	peers       *discovery.PeerStore
	status      func() discovery.Status
	scan        func(context.Context, discovery.ScanRequest) (discovery.ScanResult, error)
	chat        *chat.Hub
	updatesDir  string
	isAgent     bool
	mux         *http.ServeMux
}

// SetIsAgent toggles agent nick rules on POST /nick (P112).
func (s *Server) SetIsAgent(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isAgent = v
}

// SetUpdatesDir enables LAN offline update sharing under data-dir (P099).
func (s *Server) SetUpdatesDir(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updatesDir = dir
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
	s.mux.HandleFunc("POST /send", s.handleSend)
	s.mux.HandleFunc("POST /files/announce", s.handleFileAnnounce)
	s.mux.HandleFunc("POST /files/fetch", s.handleFileFetch)
	s.mux.HandleFunc("POST /files/cancel", s.handleFileCancel)
	s.mux.HandleFunc("GET /files/transfers", s.handleFileTransfers)
	s.mux.HandleFunc("GET /messages", s.handleMessages)
	s.mux.HandleFunc("GET /channels", s.handleChannels)
	s.mux.HandleFunc("POST /channels", s.handleCreateChannel)
	s.mux.HandleFunc("GET /updates", s.handleUpdatesList)
	s.mux.HandleFunc("POST /updates", s.handleUpdatesPut)
	s.mux.HandleFunc("GET /tail", s.handleTail)
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

// SetChat wires the chat hub into POST /send and GET /messages (P030).
func (s *Server) SetChat(hub *chat.Hub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chat = hub
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
	hub := s.chat
	s.name = name
	peerID := s.peerID
	s.mu.Unlock()

	s.mu.RLock()
	asAgent := s.isAgent
	s.mu.RUnlock()
	if asAgent {
		if err := agent.ValidateAgentNick(name); err != nil {
			http.Error(w, "agent nick invalid: "+err.Error()+"\n", http.StatusBadRequest)
			return
		}
	} else if agent.LooksLikeAgentNick(name) {
		http.Error(w, "human nick must not use agent triple-prefix\n", http.StatusBadRequest)
		return
	}
	if persist != nil {
		if err := persist(name); err != nil {
			http.Error(w, "persist failed\n", http.StatusInternalServerError)
			return
		}
	}
	if hub != nil {
		hub.SetName(name)
	}
	writeMeJSON(w, peerID, name)
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	hub := s.chat
	s.mu.RUnlock()
	if hub == nil {
		http.Error(w, "chat unavailable\n", http.StatusServiceUnavailable)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	var req struct {
		Text    string `json:"text"`
		Channel string `json:"channel"`
		WantAck bool   `json:"want_ack"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json\n", http.StatusBadRequest)
		return
	}
	res, err := hub.SendOpts(req.Text, chat.SendOptions{Channel: req.Channel, WantAck: req.WantAck})
	if err != nil {
		http.Error(w, err.Error()+"\n", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleFileAnnounce(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	hub := s.chat
	s.mu.RUnlock()
	if hub == nil {
		http.Error(w, "chat unavailable\n", http.StatusServiceUnavailable)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	var req struct {
		Name       string `json:"name"`
		Size       int64  `json:"size"`
		Mime       string `json:"mime"`
		Hash       string `json:"hash"`
		ContentB64 string `json:"content_b64"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json\n", http.StatusBadRequest)
		return
	}
	var content []byte
	if strings.TrimSpace(req.ContentB64) != "" {
		raw, err := decodeB64(req.ContentB64)
		if err != nil {
			http.Error(w, "invalid content_b64\n", http.StatusBadRequest)
			return
		}
		content = raw
	}
	res, err := hub.AnnounceFile(chat.FileAnnounce{
		Name:    req.Name,
		Size:    req.Size,
		Mime:    req.Mime,
		Hash:    req.Hash,
		Content: content,
	})
	if err != nil {
		http.Error(w, err.Error()+"\n", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleFileFetch(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	hub := s.chat
	s.mu.RUnlock()
	if hub == nil {
		http.Error(w, "chat unavailable\n", http.StatusServiceUnavailable)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	var req struct {
		FileID string `json:"file_id"`
		Wait   *bool  `json:"wait"` // default true (P051 sync); false → async progress (P052)
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json\n", http.StatusBadRequest)
		return
	}
	fileID := strings.TrimSpace(req.FileID)
	wait := true
	if req.Wait != nil {
		wait = *req.Wait
	}
	if !wait {
		tr, err := hub.StartFetch(fileID)
		if err != nil {
			http.Error(w, err.Error()+"\n", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(tr)
		return
	}
	res, err := hub.Fetch(fileID)
	if err != nil {
		http.Error(w, err.Error()+"\n", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(res)
}

func (s *Server) handleFileTransfers(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	hub := s.chat
	s.mu.RUnlock()
	list := []chat.Transfer{}
	if hub != nil {
		list = hub.Transfers()
		if list == nil {
			list = []chat.Transfer{}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"transfers": list})
}

func (s *Server) handleFileCancel(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	hub := s.chat
	s.mu.RUnlock()
	if hub == nil {
		http.Error(w, "chat unavailable\n", http.StatusServiceUnavailable)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	var req struct {
		FileID string `json:"file_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json\n", http.StatusBadRequest)
		return
	}
	tr, err := hub.CancelFetch(strings.TrimSpace(req.FileID))
	if err != nil {
		http.Error(w, err.Error()+"\n", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(tr)
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	hub := s.chat
	s.mu.RUnlock()
	msgs := []chat.Message{}
	if hub != nil {
		msgs = hub.MessagesInChannel(r.URL.Query().Get("channel"))
	}
	if msgs == nil {
		msgs = []chat.Message{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"messages": msgs})
}

func (s *Server) handleChannels(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	hub := s.chat
	s.mu.RUnlock()
	chs := []string{chat.DefaultChannel}
	if hub != nil {
		chs = hub.Channels()
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"channels": chs})
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	hub := s.chat
	s.mu.RUnlock()
	if hub == nil {
		http.Error(w, "chat unavailable\n", http.StatusServiceUnavailable)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
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
	if err := hub.EnsureChannel(req.Name); err != nil {
		http.Error(w, err.Error()+"\n", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"channels": hub.Channels()})
}

func (s *Server) handleTail(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	hub := s.chat
	s.mu.RUnlock()
	view := chat.TailView{
		Messages: []chat.Message{},
	}
	if hub != nil {
		view = hub.Tail()
		if view.Messages == nil {
			view.Messages = []chat.Message{}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(view)
}

func writeMeJSON(w http.ResponseWriter, peerID, name string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"peer_id": peerID,
		"name":    name,
	})
}

func decodeB64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(s))
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

func (s *Server) handleUpdatesList(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	dir := s.updatesDir
	s.mu.RUnlock()
	type item struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	}
	out := []item{}
	if dir != "" {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				out = append(out, item{Name: e.Name(), Size: info.Size()})
			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"updates": out})
}

func (s *Server) handleUpdatesPut(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	dir := s.updatesDir
	s.mu.RUnlock()
	if dir == "" {
		http.Error(w, "updates unavailable\n", http.StatusServiceUnavailable)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		http.Error(w, "bad request\n", http.StatusBadRequest)
		return
	}
	var req struct {
		Name      string `json:"name"`
		ContentB64 string `json:"content_b64"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json\n", http.StatusBadRequest)
		return
	}
	name := filepath.Base(strings.TrimSpace(req.Name))
	if name == "" || name == "." || name == ".." {
		http.Error(w, "bad name\n", http.StatusBadRequest)
		return
	}
	raw, err := base64.StdEncoding.DecodeString(req.ContentB64)
	if err != nil {
		http.Error(w, "bad content_b64\n", http.StatusBadRequest)
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, "mkdir failed\n", http.StatusInternalServerError)
		return
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		http.Error(w, "write failed\n", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"name": name, "size": len(raw)})
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
