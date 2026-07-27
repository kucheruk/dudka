package chat

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"dudka/internal/discovery"
	"dudka/internal/files"
	"dudka/internal/identity"
)

// Config wires a Hub to identity, peer table and dialer.
type Config struct {
	PeerID    string
	Name      string
	Store     *Store
	Peers     *discovery.PeerStore
	Dialer    discovery.DialFunc
	Timeout   time.Duration
	Logf      func(format string, args ...any)
	Blobs     *files.Store // local source blobs (P051)
	InboxDir  string       // where fetched files land (P051)
	ChunkSize int64        // LAN chunk limit; 0 → files.DefaultChunkSize
}

// Hub fans out local sends to online peers and ingests inbound chat lines.
type Hub struct {
	mu        sync.RWMutex
	peerID    string
	name      string
	store     *Store
	peers     *discovery.PeerStore
	dialer    discovery.DialFunc
	timeout   time.Duration
	logf      func(format string, args ...any)
	blobs     *files.Store
	inboxDir  string
	chunkSize int64
	syncing   bool
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
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = files.DefaultChunkSize
	}
	return &Hub{
		peerID:    cfg.PeerID,
		name:      cfg.Name,
		store:     cfg.Store,
		peers:     cfg.Peers,
		dialer:    cfg.Dialer,
		timeout:   cfg.Timeout,
		logf:      cfg.Logf,
		blobs:     cfg.Blobs,
		inboxDir:  cfg.InboxDir,
		chunkSize: cfg.ChunkSize,
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

// Send builds a message, stores it locally, and schedules best-effort fan-out.
// Status is only accepted/queued — never a delivery claim (DUD-CHAT-130 / P035).
func (h *Hub) Send(text string) (SendResult, error) {
	text = strings.TrimSpace(text)
	if err := ValidateText(text); err != nil {
		return SendResult{}, err
	}
	id, err := identity.NewUUIDv4()
	if err != nil {
		return SendResult{}, err
	}
	h.mu.RLock()
	msg := Message{
		Type:              TypeChat,
		MsgID:             id,
		PeerID:            h.peerID,
		DisplayNameAtSend: h.name,
		TS:                time.Now().UTC(),
		Text:              text,
	}
	h.mu.RUnlock()
	return h.publish(msg)
}

// AnnounceFile publishes file metadata into the feed without transferring bytes (DUD-FILE-101 / P050).
// When Content is set, bytes are stored locally for later chunk serving (P051) and never placed on the announce wire.
func (h *Hub) AnnounceFile(a FileAnnounce) (SendResult, error) {
	a.Name = strings.TrimSpace(a.Name)
	a.Mime = strings.TrimSpace(a.Mime)
	a.Hash = strings.TrimSpace(a.Hash)
	if len(a.Content) > 0 {
		if a.Size == 0 {
			a.Size = int64(len(a.Content))
		}
		if a.Size != int64(len(a.Content)) {
			return SendResult{}, fmt.Errorf("chat: content size mismatch")
		}
	}
	if err := ValidateFileAnnounce(a); err != nil {
		return SendResult{}, err
	}
	msgID, err := identity.NewUUIDv4()
	if err != nil {
		return SendResult{}, err
	}
	fileID, err := identity.NewUUIDv4()
	if err != nil {
		return SendResult{}, err
	}
	if len(a.Content) > 0 {
		if h.blobs == nil {
			return SendResult{}, fmt.Errorf("chat: blob store unavailable")
		}
		if err := h.blobs.Put(fileID, a.Content); err != nil {
			return SendResult{}, err
		}
	}
	h.mu.RLock()
	msg := Message{
		Type:              TypeFileAnnounce,
		MsgID:             msgID,
		PeerID:            h.peerID,
		DisplayNameAtSend: h.name,
		TS:                time.Now().UTC(),
		FileID:            fileID,
		FileName:          a.Name,
		Size:              a.Size,
		Mime:              a.Mime,
		Hash:              a.Hash,
	}
	h.mu.RUnlock()
	return h.publish(msg)
}

func (h *Hub) publish(msg Message) (SendResult, error) {
	_ = h.store.Append(msg)
	peers := h.peers.List()
	for _, p := range peers {
		p := p
		go h.fanout(p, msg)
	}
	queued := len(peers)
	status := SendStatusForQueued(queued)
	switch msg.Type {
	case TypeFileAnnounce:
		if status == StatusQueued {
			h.logf("file_announce_queued msg_id=%s file_id=%s queued=%d name=%q size=%d",
				msg.MsgID, msg.FileID, queued, msg.FileName, msg.Size)
		} else {
			h.logf("file_announce_accepted msg_id=%s file_id=%s queued=0 name=%q size=%d",
				msg.MsgID, msg.FileID, msg.FileName, msg.Size)
		}
	default:
		if status == StatusQueued {
			h.logf("chat_queued msg_id=%s queued=%d text_len=%d", msg.MsgID, queued, len(msg.Text))
		} else {
			h.logf("chat_accepted msg_id=%s queued=0 text_len=%d", msg.MsgID, len(msg.Text))
		}
	}
	return SendResult{Status: status, Queued: queued, Message: msg}, nil
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

// fanout is best-effort write to one peer; success is not an end-to-end ack.
func (h *Hub) fanout(p discovery.Peer, msg Message) {
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
	h.logf("chat_fanout_ok peer_id=%s msg_id=%s", p.PeerID, msg.MsgID)
}
