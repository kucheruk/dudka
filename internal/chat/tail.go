package chat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"dudka/internal/discovery"
)

// TailView is the loopback GET /tail payload (P033).
type TailView struct {
	KeeperID string    `json:"keeper_id"`
	IsKeeper bool      `json:"is_keeper"`
	Messages []Message `json:"messages"`
}

// TailEnvelope is the TCP response for type "tail".
type TailEnvelope struct {
	Type     string    `json:"type"`
	Messages []Message `json:"messages"`
}

// EncodeTailReq builds a newline-delimited tail request.
func EncodeTailReq() []byte {
	return []byte(`{"type":"tail_req"}` + "\n")
}

// EncodeTail serializes a tail response line.
func EncodeTail(msgs []Message) ([]byte, error) {
	if msgs == nil {
		msgs = []Message{}
	}
	raw, err := json.Marshal(TailEnvelope{Type: "tail", Messages: msgs})
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// DecodeTail parses a tail response line.
func DecodeTail(raw []byte) (TailEnvelope, error) {
	var env TailEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return TailEnvelope{}, err
	}
	if env.Type != "" && env.Type != "tail" {
		return TailEnvelope{}, fmt.Errorf("chat: unexpected type %q", env.Type)
	}
	env.Type = "tail"
	if env.Messages == nil {
		env.Messages = []Message{}
	}
	return env, nil
}

// Tail returns the local ring plus current keeper identity.
func (h *Hub) Tail() TailView {
	h.mu.RLock()
	self := h.peerID
	h.mu.RUnlock()
	peers := h.peers.List()
	keeperID, ok := SelectTailKeeperAmong(self, peers)
	if !ok {
		keeperID = self
	}
	msgs := h.store.List()
	if msgs == nil {
		msgs = []Message{}
	}
	return TailView{
		KeeperID: keeperID,
		IsKeeper: keeperID == self,
		Messages: msgs,
	}
}

// HandleTailRequest serves TCP type=tail_req with the local ring.
func (h *Hub) HandleTailRequest(_ string, conn net.Conn) {
	_ = conn.SetDeadline(time.Now().Add(h.timeout))
	raw, err := EncodeTail(h.store.List())
	if err != nil {
		return
	}
	if _, err := conn.Write(raw); err != nil {
		h.logf("tail_write_err err=%v", err)
		return
	}
	h.logf("tail_serve n=%d", len(h.store.List()))
}

// OnPeerUpserted triggers a best-effort sync from the current keeper (P033).
func (h *Hub) OnPeerUpserted(_ discovery.Peer, _ discovery.UpsertResult) {
	go h.SyncTail()
}

// OnPeerRemoved re-evaluates keeper after a peer TTL expiry (P034).
func (h *Hub) OnPeerRemoved(_ discovery.Peer) {
	go h.SyncTail()
}

// SyncTail pulls the keeper ring when this node is not the keeper.
func (h *Hub) SyncTail() {
	h.mu.Lock()
	if h.syncing {
		h.mu.Unlock()
		return
	}
	h.syncing = true
	self := h.peerID
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		h.syncing = false
		h.mu.Unlock()
	}()

	peers := h.peers.List()
	keeperID, ok := SelectTailKeeperAmong(self, peers)
	if !ok || keeperID == self {
		return
	}
	var keeper discovery.Peer
	found := false
	for _, p := range peers {
		if p.PeerID == keeperID {
			keeper = p
			found = true
			break
		}
	}
	if !found || keeper.Host == "" || keeper.TCPPort <= 0 {
		return
	}
	if err := h.fetchTail(keeper); err != nil {
		h.logf("tail_sync_err keeper=%s err=%v", keeperID, err)
		return
	}
	h.logf("tail_sync_ok keeper=%s n=%d", keeperID, len(h.store.List()))
}

func (h *Hub) fetchTail(p discovery.Peer) error {
	if err := discovery.CheckDialHost(p.Host); err != nil {
		return err
	}
	addr := net.JoinHostPort(p.Host, fmt.Sprintf("%d", p.TCPPort))
	conn, err := h.dialer("tcp", addr, h.timeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(h.timeout))
	if _, err := conn.Write(EncodeTailReq()); err != nil {
		return err
	}
	br := bufio.NewReader(conn)
	line, err := br.ReadBytes('\n')
	if err != nil && !(err == io.EOF && len(line) > 0) {
		return err
	}
	env, err := DecodeTail(line)
	if err != nil {
		return err
	}
	for _, msg := range env.Messages {
		m := msg
		h.materializeThumb(&m)
		if _, err := h.store.AppendPersistent(m); err != nil {
			h.logf("tail_persist_err msg_id=%s err=%v", m.MsgID, err)
		}
	}
	return nil
}
