// Package signaling routes short-lived WebRTC negotiation messages.
// It never accepts chat messages or file contents.
package signaling

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	MaxPeersPerRoom   = 8
	MaxSignalBytes    = 64 << 10
	MaxSignalsPerSec  = 20
	clientSendQueue   = 64
	writeTimeout      = 5 * time.Second
	pingInterval      = 25 * time.Second
	closePolicyReason = "недопустимый signaling"
)

var (
	errRoomFull      = errors.New("room full")
	errRateLimit     = errors.New("signal rate limit")
	errInvalidSignal = errors.New("invalid signal")
)

// Server is the memory-only signaling rendezvous.
type Server struct {
	allowedOrigin string

	mu    sync.Mutex
	rooms map[string]map[string]*client
}

type client struct {
	id   string
	room string
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
}

type wireSignal struct {
	Type        string          `json:"type"`
	To          string          `json:"to,omitempty"`
	From        string          `json:"from,omitempty"`
	Peers       []string        `json:"peers,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
	Candidate   json.RawMessage `json:"candidate,omitempty"`
}

// NewServer creates a signaling server restricted to one exact browser origin.
func NewServer(allowedOrigin string) *Server {
	return &Server{
		allowedOrigin: strings.TrimRight(strings.TrimSpace(allowedOrigin), "/"),
		rooms:         make(map[string]map[string]*client),
	}
}

// Handler exposes GET /health and the WebSocket endpoint at /.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /debug", func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		roomSizes := make([]int, 0, len(s.rooms))
		peerCount := 0
		for _, room := range s.rooms {
			roomSizes = append(roomSizes, len(room))
			peerCount += len(room)
		}
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rooms":      len(roomSizes),
			"peers":      peerCount,
			"room_sizes": roomSizes,
		})
	})
	mux.HandleFunc("GET /", s.serveWebSocket)
	return securityHeaders(mux)
}

func (s *Server) serveWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.originAllowed(r.Header.Get("Origin")) {
		http.Error(w, "origin запрещён", http.StatusForbidden)
		return
	}
	room, err := roomKey(clientIP(r))
	if err != nil {
		http.Error(w, "адрес клиента не определён", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{originHost(s.allowedOrigin)},
	})
	if err != nil {
		return
	}
	conn.SetReadLimit(MaxSignalBytes)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &client{
		id:   randomID(),
		room: room,
		conn: conn,
		send: make(chan []byte, clientSendQueue),
		done: make(chan struct{}),
	}
	peers, err := s.join(c)
	if err != nil {
		_ = conn.Close(websocket.StatusPolicyViolation, "слишком много вкладок")
		return
	}
	defer func() {
		s.leave(c)
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	welcome, _ := json.Marshal(wireSignal{Type: "welcome", From: c.id, Peers: peers})
	if err := writeText(ctx, conn, welcome); err != nil {
		return
	}
	s.notifyPeers(c.room, peers, wireSignal{Type: "peer-joined", From: c.id})

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		ping := time.NewTicker(pingInterval)
		defer ping.Stop()
		for {
			select {
			case <-c.done:
				return
			case <-ping.C:
				pingCtx, pingCancel := context.WithTimeout(ctx, writeTimeout)
				err := conn.Ping(pingCtx)
				pingCancel()
				if err != nil {
					return
				}
			case payload := <-c.send:
				if err := writeText(ctx, conn, payload); err != nil {
					return
				}
			}
		}
	}()

	err = s.readLoop(ctx, c)
	if err != nil && !errors.Is(err, context.Canceled) {
		_ = conn.Close(websocket.StatusPolicyViolation, closePolicyReason)
	}
	s.leave(c)
	close(c.done)
	<-writeDone
}

func (s *Server) readLoop(ctx context.Context, c *client) error {
	windowStart := time.Now()
	count := 0
	for {
		typ, raw, err := c.conn.Read(ctx)
		if err != nil {
			return err
		}
		if typ != websocket.MessageText || len(raw) > MaxSignalBytes {
			return errInvalidSignal
		}
		now := time.Now()
		if now.Sub(windowStart) >= time.Second {
			windowStart = now
			count = 0
		}
		count++
		if count > MaxSignalsPerSec {
			return errRateLimit
		}

		var msg wireSignal
		if err := json.Unmarshal(raw, &msg); err != nil || !validSignal(msg) {
			return errInvalidSignal
		}
		msg.From = c.id
		payload, _ := json.Marshal(msg)
		if err := s.route(c, msg.To, payload); err != nil {
			return err
		}
	}
}

func validSignal(msg wireSignal) bool {
	if strings.TrimSpace(msg.To) == "" || msg.From != "" || len(msg.Peers) != 0 {
		return false
	}
	switch msg.Type {
	case "offer", "answer":
		return isJSONObject(msg.Description) && len(msg.Candidate) == 0
	case "ice":
		return (isJSONObject(msg.Candidate) || string(msg.Candidate) == "null") &&
			len(msg.Description) == 0
	default:
		return false
	}
}

func isJSONObject(raw json.RawMessage) bool {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	return len(raw) >= 2 && raw[0] == '{' && raw[len(raw)-1] == '}'
}

func (s *Server) join(c *client) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	room := s.rooms[c.room]
	if room == nil {
		room = make(map[string]*client)
		s.rooms[c.room] = room
	}
	if len(room) >= MaxPeersPerRoom {
		return nil, errRoomFull
	}
	peers := make([]string, 0, len(room))
	for id := range room {
		peers = append(peers, id)
	}
	room[c.id] = c
	return peers, nil
}

func (s *Server) leave(c *client) {
	s.mu.Lock()
	room := s.rooms[c.room]
	if room[c.id] != c {
		s.mu.Unlock()
		return
	}
	delete(room, c.id)
	peerIDs := make([]string, 0, len(room))
	for id := range room {
		peerIDs = append(peerIDs, id)
	}
	if len(room) == 0 {
		delete(s.rooms, c.room)
	}
	s.mu.Unlock()
	s.notifyPeers(c.room, peerIDs, wireSignal{Type: "peer-left", From: c.id})
}

func (s *Server) notifyPeers(room string, peerIDs []string, message wireSignal) {
	payload, _ := json.Marshal(message)
	s.mu.Lock()
	targets := make([]*client, 0, len(peerIDs))
	for _, id := range peerIDs {
		if target := s.rooms[room][id]; target != nil {
			targets = append(targets, target)
		}
	}
	s.mu.Unlock()
	for _, target := range targets {
		select {
		case <-target.done:
		case target.send <- payload:
		default:
		}
	}
}

func (s *Server) route(from *client, targetID string, payload []byte) error {
	s.mu.Lock()
	target := s.rooms[from.room][targetID]
	s.mu.Unlock()
	if target == nil || target == from {
		return errInvalidSignal
	}
	select {
	case <-target.done:
		return errInvalidSignal
	case target.send <- payload:
		return nil
	default:
		return errors.New("target queue full")
	}
}

func (s *Server) originAllowed(origin string) bool {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	return origin != "" && origin == s.allowedOrigin
}

func clientIP(r *http.Request) string {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	remote := net.ParseIP(host)
	if remote != nil && remote.IsLoopback() {
		if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
			return forwarded
		}
	}
	return host
}

func roomKey(rawIP string) (string, error) {
	ip := net.ParseIP(strings.TrimSpace(rawIP))
	if ip == nil {
		return "", errors.New("invalid client IP")
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	sum := sha256.Sum256([]byte(ip.String()))
	return base64.RawURLEncoding.EncodeToString(sum[:16]), nil
}

func randomID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(fmt.Sprintf("signaling random id: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(raw[:])
}

func originHost(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return u.Host
}

func writeText(parent context.Context, conn *websocket.Conn, payload []byte) error {
	ctx, cancel := context.WithTimeout(parent, writeTimeout)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, payload)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
