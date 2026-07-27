package chat

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"dudka/internal/discovery"
	"dudka/internal/identity"
)

// Config wires a Hub to identity, peer table and dialer.
type Config struct {
	PeerID  string
	Name    string
	Store   *Store
	Peers   *discovery.PeerStore
	Dialer  discovery.DialFunc
	Timeout time.Duration
	Logf    func(format string, args ...any)
}

// Hub fans out local sends to online peers and ingests inbound chat lines.
type Hub struct {
	mu      sync.RWMutex
	peerID  string
	name    string
	store   *Store
	peers   *discovery.PeerStore
	dialer  discovery.DialFunc
	timeout time.Duration
	logf    func(format string, args ...any)
}

// NewHub builds a chat hub. Store/Peers must be non-nil.
func NewHub(cfg Config) *Hub {
	if cfg.Store == nil {
		cfg.Store = NewStore()
	}
	if cfg.Peers == nil {
		cfg.Peers = discovery.NewPeerStore()
	}
	if cfg.Dialer == nil {
		cfg.Dialer = net.DialTimeout
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	return &Hub{
		peerID:  cfg.PeerID,
		name:    cfg.Name,
		store:   cfg.Store,
		peers:   cfg.Peers,
		dialer:  cfg.Dialer,
		timeout: cfg.Timeout,
		logf:    cfg.Logf,
	}
}

// SetName updates display_name used for subsequent sends (DUD-CHAT-111 foreshadow).
func (h *Hub) SetName(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.name = name
}

// Messages returns the local log.
func (h *Hub) Messages() []Message { return h.store.List() }

// Send builds a message, stores it locally, and fans out to all known peers.
func (h *Hub) Send(text string) (Message, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Message{}, fmt.Errorf("chat: text is required")
	}
	id, err := identity.NewUUIDv4()
	if err != nil {
		return Message{}, err
	}
	h.mu.RLock()
	msg := Message{
		Type:              "chat",
		MsgID:             id,
		PeerID:            h.peerID,
		DisplayNameAtSend: h.name,
		TS:                time.Now().UTC(),
		Text:              text,
	}
	h.mu.RUnlock()

	_ = h.store.Append(msg)
	peers := h.peers.List()
	for _, p := range peers {
		p := p
		go h.deliver(p, msg)
	}
	h.logf("chat_send msg_id=%s peers=%d text_len=%d", msg.MsgID, len(peers), len(msg.Text))
	return msg, nil
}

// HandleChatLine is a discovery.OnChatLine callback for inbound TCP chat frames.
func (h *Hub) HandleChatLine(_ string, line []byte) {
	msg, err := DecodeMessage(line)
	if err != nil {
		h.logf("chat_rx_err err=%v", err)
		return
	}
	if h.store.Append(msg) {
		h.logf("chat_rx msg_id=%s peer_id=%s", msg.MsgID, msg.PeerID)
	}
}

func (h *Hub) deliver(p discovery.Peer, msg Message) {
	if p.Host == "" || p.TCPPort <= 0 {
		return
	}
	if err := discovery.CheckDialHost(p.Host); err != nil {
		h.logf("%s", discovery.FormatWanRefuse(p.Host))
		return
	}
	addr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.TCPPort))
	conn, err := h.dialer("tcp", addr, h.timeout)
	if err != nil {
		h.logf("chat_dial_err peer_id=%s addr=%s err=%v", p.PeerID, addr, err)
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(h.timeout))
	raw, err := EncodeMessage(msg)
	if err != nil {
		return
	}
	if _, err := conn.Write(raw); err != nil {
		h.logf("chat_write_err peer_id=%s err=%v", p.PeerID, err)
		return
	}
	h.logf("chat_deliver_ok peer_id=%s msg_id=%s", p.PeerID, msg.MsgID)
}
