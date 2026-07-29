// Package rtcmesh connects native Dudka clients to the same WebRTC mesh as
// browser clients. Signaling carries negotiation only; chat stays in DTLS
// DataChannels.
package rtcmesh

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/files"

	"github.com/coder/websocket"
	"github.com/pion/webrtc/v4"
)

const (
	defaultSignalURL = "wss://zamoo.team/dudka/signal"
	defaultOrigin    = "https://zamoo.team"
	defaultSTUNURL   = "stun:zamoo.team:3478"
)

type Config struct {
	PeerID    string
	Name      string
	Peers     *discovery.PeerStore
	History   func() []chat.Message
	OnMessage func([]byte)
	Blobs     *files.Store
	Logf      func(string, ...any)
	SignalURL string
	Origin    string
	STUNURL   string
}

type Client struct {
	mu       sync.RWMutex
	writeMu  sync.Mutex
	cfg      Config
	name     string
	signalID string
	socket   *websocket.Conn
	peers    map[string]*peer
	cancel   context.CancelFunc
	running  bool
}

type peer struct {
	mu                  sync.Mutex
	signalID            string
	remotePeerID        string
	pc                  *webrtc.PeerConnection
	channel             *webrtc.DataChannel
	open                bool
	descriptionSent     bool
	pendingLocalICE     []*webrtc.ICECandidateInit
	pendingRemoteICE    []*webrtc.ICECandidateInit
	remoteDescriptionOK bool
}

type signal struct {
	Type        string                     `json:"type"`
	To          string                     `json:"to,omitempty"`
	From        string                     `json:"from,omitempty"`
	Peers       []string                   `json:"peers,omitempty"`
	Description *webrtc.SessionDescription `json:"description,omitempty"`
	Candidate   *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
}

type packet struct {
	Type     string           `json:"type"`
	PeerID   string           `json:"peerID,omitempty"`
	Name     string           `json:"name,omitempty"`
	Message  *browserMessage  `json:"message,omitempty"`
	Messages []browserMessage `json:"messages,omitempty"`
}

type browserMessage struct {
	ID       string `json:"id"`
	SenderID string `json:"senderID"`
	Sender   string `json:"sender"`
	Text     string `json:"text"`
	SentAt   string `json:"sentAt"`
}

type fileMeta struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Sender string `json:"sender"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	MIME   string `json:"mime"`
	SentAt string `json:"sentAt"`
}

func New(cfg Config) *Client {
	if cfg.Peers == nil {
		cfg.Peers = discovery.NewPeerStore()
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	if strings.TrimSpace(cfg.SignalURL) == "" {
		cfg.SignalURL = defaultSignalURL
	}
	if strings.TrimSpace(cfg.Origin) == "" {
		cfg.Origin = defaultOrigin
	}
	if strings.TrimSpace(cfg.STUNURL) == "" {
		cfg.STUNURL = defaultSTUNURL
	}
	return &Client{cfg: cfg, name: cfg.Name, peers: make(map[string]*peer)}
}

func (c *Client) Start(parent context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	c.cancel = cancel
	c.running = true
	c.mu.Unlock()
	go c.reconnectLoop(ctx)
}

func (c *Client) Stop() {
	c.mu.Lock()
	cancel := c.cancel
	c.cancel = nil
	c.running = false
	socket := c.socket
	c.socket = nil
	peers := c.peers
	c.peers = make(map[string]*peer)
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if socket != nil {
		_ = socket.Close(websocket.StatusNormalClosure, "клиент остановлен")
	}
	for _, p := range peers {
		_ = p.pc.Close()
	}
	for _, known := range c.cfg.Peers.List() {
		c.cfg.Peers.Remove(known.PeerID)
	}
}

func (c *Client) SetName(name string) {
	c.mu.Lock()
	c.name = strings.TrimSpace(name)
	c.mu.Unlock()
	c.broadcastPacket(packet{Type: "hello", PeerID: c.cfg.PeerID, Name: name})
}

func (c *Client) Broadcast(message chat.Message) int {
	if message.Type == chat.TypeFileAnnounce {
		return c.broadcastFile(message)
	}
	if message.Type == chat.TypeChat {
		item := fromChat(message)
		return c.broadcastPacket(packet{Type: "chat", Message: &item})
	}
	return 0
}

func (c *Client) Connected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.socket != nil
}

func (c *Client) Restart() {
	c.mu.RLock()
	socket := c.socket
	c.mu.RUnlock()
	if socket != nil {
		_ = socket.Close(websocket.StatusNormalClosure, "повторный поиск")
	}
}

func (c *Client) reconnectLoop(ctx context.Context) {
	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
	}()
	delay := time.Second
	for {
		if err := c.runSession(ctx); err != nil && !errors.Is(err, context.Canceled) {
			c.cfg.Logf("webrtc_signal_err err=%v retry=%s", err, delay)
		}
		c.closePeers()
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		if delay < 10*time.Second {
			delay *= 2
		}
	}
}

func (c *Client) runSession(ctx context.Context) error {
	headers := http.Header{}
	headers.Set("Origin", c.cfg.Origin)
	conn, _, err := websocket.Dial(ctx, c.cfg.SignalURL, &websocket.DialOptions{
		HTTPHeader: headers,
	})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	conn.SetReadLimit(64 << 10)
	c.mu.Lock()
	c.socket = conn
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		if c.socket == conn {
			c.socket = nil
		}
		c.signalID = ""
		c.mu.Unlock()
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()
	c.cfg.Logf("webrtc_signal_open")

	for {
		kind, raw, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		if kind != websocket.MessageText {
			return errors.New("signaling sent non-text frame")
		}
		var incoming signal
		if err := json.Unmarshal(raw, &incoming); err != nil {
			return err
		}
		if err := c.handleSignal(incoming); err != nil {
			c.cfg.Logf("webrtc_signal_handle_err type=%s peer=%s err=%v", incoming.Type, incoming.From, err)
		}
	}
}

func (c *Client) handleSignal(in signal) error {
	switch in.Type {
	case "welcome":
		c.mu.Lock()
		c.signalID = in.From
		c.mu.Unlock()
		for _, id := range in.Peers {
			if err := c.connectPeer(id); err != nil {
				return err
			}
		}
	case "peer-joined":
		return c.connectPeer(in.From)
	case "peer-left":
		c.removePeer(in.From)
	case "offer":
		if in.Description == nil {
			return errors.New("offer without description")
		}
		p, err := c.ensurePeer(in.From, false)
		if err != nil {
			return err
		}
		if err := p.pc.SetRemoteDescription(*in.Description); err != nil {
			return err
		}
		if err := c.remoteDescriptionReady(p); err != nil {
			return err
		}
		answer, err := p.pc.CreateAnswer(nil)
		if err != nil {
			return err
		}
		if err := p.pc.SetLocalDescription(answer); err != nil {
			return err
		}
		if err := c.sendSignal(signal{Type: "answer", To: in.From, Description: &answer}); err != nil {
			return err
		}
		c.descriptionSent(p)
	case "answer":
		if in.Description == nil {
			return errors.New("answer without description")
		}
		p := c.getPeer(in.From)
		if p == nil {
			return errors.New("answer from unknown peer")
		}
		if err := p.pc.SetRemoteDescription(*in.Description); err != nil {
			return err
		}
		return c.remoteDescriptionReady(p)
	case "ice":
		p := c.getPeer(in.From)
		if p == nil {
			return nil
		}
		p.mu.Lock()
		ready := p.remoteDescriptionOK
		if !ready {
			p.pendingRemoteICE = append(p.pendingRemoteICE, in.Candidate)
		}
		p.mu.Unlock()
		if ready && in.Candidate != nil {
			return p.pc.AddICECandidate(*in.Candidate)
		}
	default:
		return fmt.Errorf("unknown signaling type %q", in.Type)
	}
	return nil
}

func (c *Client) connectPeer(signalID string) error {
	c.mu.RLock()
	ours := c.signalID
	c.mu.RUnlock()
	if signalID == "" || ours == "" || signalID == ours {
		return nil
	}
	initiator := strings.Compare(ours, signalID) > 0
	p, err := c.ensurePeer(signalID, initiator)
	if err != nil || !initiator {
		return err
	}
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		return err
	}
	if err := p.pc.SetLocalDescription(offer); err != nil {
		return err
	}
	if err := c.sendSignal(signal{Type: "offer", To: signalID, Description: &offer}); err != nil {
		return err
	}
	c.descriptionSent(p)
	return nil
}

func (c *Client) ensurePeer(signalID string, initiator bool) (*peer, error) {
	if existing := c.getPeer(signalID); existing != nil {
		return existing, nil
	}
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{c.cfg.STUNURL}}},
	})
	if err != nil {
		return nil, err
	}
	p := &peer{signalID: signalID, pc: pc}
	c.mu.Lock()
	if existing := c.peers[signalID]; existing != nil {
		c.mu.Unlock()
		_ = pc.Close()
		return existing, nil
	}
	c.peers[signalID] = p
	c.mu.Unlock()

	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		init := candidate.ToJSON()
		p.mu.Lock()
		if !p.descriptionSent {
			p.pendingLocalICE = append(p.pendingLocalICE, &init)
			p.mu.Unlock()
			return
		}
		p.mu.Unlock()
		if err := c.sendSignal(signal{Type: "ice", To: signalID, Candidate: &init}); err != nil {
			c.cfg.Logf("webrtc_ice_send_err peer=%s err=%v", signalID, err)
		}
	})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
			c.removePeer(signalID)
		}
	})
	pc.OnDataChannel(func(channel *webrtc.DataChannel) {
		if channel.Label() == "dudka-chat" {
			c.wireChannel(p, channel)
		} else if strings.HasPrefix(channel.Label(), "dudka-file:") {
			c.receiveFile(p, channel)
		}
	})
	if initiator {
		channel, err := pc.CreateDataChannel("dudka-chat", nil)
		if err != nil {
			c.removePeer(signalID)
			return nil, err
		}
		c.wireChannel(p, channel)
	}
	return p, nil
}

func (c *Client) wireChannel(p *peer, channel *webrtc.DataChannel) {
	p.mu.Lock()
	p.channel = channel
	p.mu.Unlock()
	channel.OnOpen(func() {
		p.mu.Lock()
		p.open = true
		p.mu.Unlock()
		c.mu.RLock()
		name := c.name
		c.mu.RUnlock()
		c.sendPacket(p, packet{Type: "hello", PeerID: c.cfg.PeerID, Name: name})
		if c.cfg.History != nil {
			history := c.cfg.History()
			for start := 0; start < len(history); start += 10 {
				end := start + 10
				if end > len(history) {
					end = len(history)
				}
				items := make([]browserMessage, 0, end-start)
				for _, message := range history[start:end] {
					if message.Type == chat.TypeChat {
						items = append(items, fromChat(message))
					}
				}
				if len(items) > 0 {
					c.sendPacket(p, packet{Type: "tail", Messages: items})
				}
			}
		}
	})
	channel.OnMessage(func(message webrtc.DataChannelMessage) {
		if !message.IsString {
			return
		}
		var incoming packet
		if err := json.Unmarshal(message.Data, &incoming); err != nil {
			return
		}
		c.handlePacket(p, incoming)
	})
	channel.OnClose(func() { c.removePeer(p.signalID) })
}

func (c *Client) handlePacket(p *peer, incoming packet) {
	switch incoming.Type {
	case "hello":
		id := strings.TrimSpace(incoming.PeerID)
		name := strings.TrimSpace(incoming.Name)
		if id == "" || name == "" {
			return
		}
		p.mu.Lock()
		p.remotePeerID = id
		p.mu.Unlock()
		c.cfg.Peers.Upsert(discovery.Peer{
			PeerID: id, DisplayName: name, InstanceID: p.signalID,
			ProtoMajor: discovery.DefaultProtoMajor, LastSeen: time.Now().UTC(),
		})
	case "chat":
		if incoming.Message != nil {
			c.ingest(*incoming.Message)
		}
	case "tail":
		for _, item := range incoming.Messages {
			c.ingest(item)
		}
	}
}

func (c *Client) ingest(item browserMessage) {
	if c.cfg.OnMessage == nil {
		return
	}
	parsed, err := time.Parse(time.RFC3339Nano, item.SentAt)
	if err != nil || item.ID == "" || item.SenderID == "" ||
		strings.TrimSpace(item.Sender) == "" || strings.TrimSpace(item.Text) == "" {
		return
	}
	raw, err := chat.EncodeMessage(chat.Message{
		Type: chat.TypeChat, MsgID: item.ID, PeerID: item.SenderID,
		DisplayNameAtSend: item.Sender, Text: item.Text, TS: parsed,
		Channel: chat.DefaultChannel,
	})
	if err == nil {
		c.cfg.OnMessage(raw)
	}
}

func fromChat(message chat.Message) browserMessage {
	return browserMessage{
		ID: message.MsgID, SenderID: message.PeerID,
		Sender: message.DisplayNameAtSend, Text: message.Text,
		SentAt: message.TS.UTC().Format(time.RFC3339Nano),
	}
}

func (c *Client) broadcastFile(message chat.Message) int {
	if c.cfg.Blobs == nil {
		return 0
	}
	source, err := c.cfg.Blobs.Open(message.FileID)
	if err != nil {
		return 0
	}
	content, err := io.ReadAll(source)
	_ = source.Close()
	if err != nil {
		return 0
	}
	c.mu.RLock()
	peers := make([]*peer, 0, len(c.peers))
	for _, p := range c.peers {
		p.mu.Lock()
		open := p.open
		p.mu.Unlock()
		if open {
			peers = append(peers, p)
		}
	}
	c.mu.RUnlock()
	for _, p := range peers {
		p := p
		go c.sendFile(p, message, content)
	}
	return len(peers)
}

func (c *Client) sendFile(p *peer, message chat.Message, content []byte) {
	channel, err := p.pc.CreateDataChannel("dudka-file:"+message.FileID, nil)
	if err != nil {
		return
	}
	channel.OnOpen(func() {
		meta, _ := json.Marshal(fileMeta{
			Type: "meta", ID: message.FileID, Sender: message.DisplayNameAtSend,
			Name: message.FileName, Size: message.Size, MIME: message.Mime,
			SentAt: message.TS.UTC().Format(time.RFC3339Nano),
		})
		if err := channel.SendText(string(meta)); err != nil {
			return
		}
		const chunk = 16 << 10
		for start := 0; start < len(content); start += chunk {
			end := start + chunk
			if end > len(content) {
				end = len(content)
			}
			if err := channel.Send(content[start:end]); err != nil {
				return
			}
		}
		_ = channel.SendText(`{"type":"done"}`)
	})
}

func (c *Client) receiveFile(p *peer, channel *webrtc.DataChannel) {
	var meta *fileMeta
	var content []byte
	channel.OnMessage(func(message webrtc.DataChannelMessage) {
		if message.IsString {
			var envelope struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(message.Data, &envelope) != nil {
				return
			}
			switch envelope.Type {
			case "meta":
				var next fileMeta
				if json.Unmarshal(message.Data, &next) != nil ||
					next.ID == "" || strings.TrimSpace(next.Name) == "" ||
					next.Size < 0 || strings.TrimSpace(next.MIME) == "" {
					_ = channel.Close()
					return
				}
				meta = &next
				content = make([]byte, 0, next.Size)
			case "done":
				if meta == nil || int64(len(content)) != meta.Size || c.cfg.Blobs == nil {
					_ = channel.Close()
					return
				}
				if err := c.cfg.Blobs.Put(meta.ID, content); err != nil {
					_ = channel.Close()
					return
				}
				p.mu.Lock()
				remotePeerID := p.remotePeerID
				p.mu.Unlock()
				if remotePeerID == "" {
					remotePeerID = p.signalID
				}
				ts, err := time.Parse(time.RFC3339Nano, meta.SentAt)
				if err != nil {
					ts = time.Now().UTC()
				}
				sum := sha256.Sum256(content)
				raw, err := chat.EncodeMessage(chat.Message{
					Type: chat.TypeFileAnnounce, MsgID: meta.ID,
					PeerID: remotePeerID, DisplayNameAtSend: meta.Sender, TS: ts,
					FileID: meta.ID, FileName: meta.Name, Size: meta.Size,
					Mime: meta.MIME, Hash: fmt.Sprintf("%x", sum[:]),
				})
				if err == nil && c.cfg.OnMessage != nil {
					c.cfg.OnMessage(raw)
				}
				_ = channel.Close()
			}
			return
		}
		if meta == nil || int64(len(content)+len(message.Data)) > meta.Size {
			_ = channel.Close()
			return
		}
		content = append(content, message.Data...)
	})
}

func (c *Client) descriptionSent(p *peer) {
	p.mu.Lock()
	p.descriptionSent = true
	pending := p.pendingLocalICE
	p.pendingLocalICE = nil
	p.mu.Unlock()
	for _, candidate := range pending {
		if err := c.sendSignal(signal{Type: "ice", To: p.signalID, Candidate: candidate}); err != nil {
			c.cfg.Logf("webrtc_ice_send_err peer=%s err=%v", p.signalID, err)
		}
	}
}

func (c *Client) remoteDescriptionReady(p *peer) error {
	p.mu.Lock()
	p.remoteDescriptionOK = true
	pending := p.pendingRemoteICE
	p.pendingRemoteICE = nil
	p.mu.Unlock()
	for _, candidate := range pending {
		if candidate != nil {
			if err := p.pc.AddICECandidate(*candidate); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) sendSignal(message signal) error {
	c.mu.RLock()
	socket := c.socket
	c.mu.RUnlock()
	if socket == nil {
		return errors.New("signaling is closed")
	}
	raw, err := json.Marshal(message)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return socket.Write(ctx, websocket.MessageText, raw)
}

func (c *Client) broadcastPacket(message packet) int {
	c.mu.RLock()
	peers := make([]*peer, 0, len(c.peers))
	for _, p := range c.peers {
		peers = append(peers, p)
	}
	c.mu.RUnlock()
	sent := 0
	for _, p := range peers {
		if c.sendPacket(p, message) {
			sent++
		}
	}
	return sent
}

func (c *Client) sendPacket(p *peer, message packet) bool {
	raw, err := json.Marshal(message)
	if err != nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.open || p.channel == nil {
		return false
	}
	if err := p.channel.SendText(string(raw)); err != nil {
		return false
	}
	return true
}

func (c *Client) getPeer(signalID string) *peer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.peers[signalID]
}

func (c *Client) removePeer(signalID string) {
	c.mu.Lock()
	p := c.peers[signalID]
	if p != nil {
		delete(c.peers, signalID)
	}
	c.mu.Unlock()
	if p == nil {
		return
	}
	p.mu.Lock()
	remotePeerID := p.remotePeerID
	p.open = false
	p.mu.Unlock()
	if remotePeerID != "" {
		c.cfg.Peers.Remove(remotePeerID)
	}
	_ = p.pc.Close()
}

func (c *Client) closePeers() {
	c.mu.Lock()
	peers := c.peers
	c.peers = make(map[string]*peer)
	c.mu.Unlock()
	for _, p := range peers {
		p.mu.Lock()
		remotePeerID := p.remotePeerID
		p.mu.Unlock()
		if remotePeerID != "" {
			c.cfg.Peers.Remove(remotePeerID)
		}
		_ = p.pc.Close()
	}
}
