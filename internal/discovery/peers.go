package discovery

import (
	"sort"
	"sync"
	"time"
)

// Peer is a known LAN neighbor after successful register.
type Peer struct {
	PeerID      string    `json:"peer_id"`
	DisplayName string    `json:"display_name"`
	InstanceID  string    `json:"instance_id"`
	ProtoMajor  int       `json:"proto_major"`
	ProtoMinor  int       `json:"proto_minor"`
	Host        string    `json:"host"`
	TCPPort     int       `json:"tcp_port"`
	LastSeen    time.Time `json:"last_seen"`
}

// PeerStore is the in-memory neighbor table (P021).
type PeerStore struct {
	mu    sync.RWMutex
	peers map[string]Peer
}

// NewPeerStore returns an empty peer table.
func NewPeerStore() *PeerStore {
	return &PeerStore{peers: make(map[string]Peer)}
}

// Upsert inserts or replaces a peer by peer_id.
func (s *PeerStore) Upsert(p Peer) {
	if p.PeerID == "" {
		return
	}
	if p.LastSeen.IsZero() {
		p.LastSeen = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.peers[p.PeerID] = p
}

// List returns peers sorted by peer_id.
func (s *PeerStore) List() []Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Peer, 0, len(s.peers))
	for _, p := range s.peers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PeerID < out[j].PeerID })
	return out
}
