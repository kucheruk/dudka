package discovery

import (
	"fmt"
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
	Updated     bool      `json:"updated"`
	LastSeen    time.Time `json:"last_seen"`
}

// UpsertResult describes how the peer table changed.
type UpsertResult struct {
	Created         bool
	InstanceChanged bool
	OldInstanceID   string
}

// PeerStore is the in-memory neighbor table (P021/P022).
type PeerStore struct {
	mu    sync.RWMutex
	peers map[string]Peer
}

// NewPeerStore returns an empty peer table.
func NewPeerStore() *PeerStore {
	return &PeerStore{peers: make(map[string]Peer)}
}

// Upsert inserts or replaces a peer by peer_id.
// Same peer_id + new instance_id marks Updated and never creates a second row (P022).
func (s *PeerStore) Upsert(p Peer) UpsertResult {
	var res UpsertResult
	if p.PeerID == "" {
		return res
	}
	if p.LastSeen.IsZero() {
		p.LastSeen = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.peers[p.PeerID]
	if !ok {
		res.Created = true
		p.Updated = false
		s.peers[p.PeerID] = p
		return res
	}
	if old.InstanceID != "" && p.InstanceID != "" && old.InstanceID != p.InstanceID {
		res.InstanceChanged = true
		res.OldInstanceID = old.InstanceID
		p.Updated = true
	} else {
		p.Updated = false
	}
	s.peers[p.PeerID] = p
	return res
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

// Remove deletes a peer by id. Returns false if unknown.
func (s *PeerStore) Remove(peerID string) bool {
	if peerID == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.peers[peerID]; !ok {
		return false
	}
	delete(s.peers, peerID)
	return true
}

// Touch updates LastSeen for a known peer (announce heartbeat).
func (s *PeerStore) Touch(peerID string) bool {
	if peerID == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.peers[peerID]
	if !ok {
		return false
	}
	p.LastSeen = time.Now().UTC()
	s.peers[peerID] = p
	return true
}

// PruneOlderThan removes peers with LastSeen before cutoff (P034).
func (s *PeerStore) PruneOlderThan(cutoff time.Time) []Peer {
	s.mu.Lock()
	defer s.mu.Unlock()
	var removed []Peer
	for id, p := range s.peers {
		if p.LastSeen.Before(cutoff) {
			removed = append(removed, p)
			delete(s.peers, id)
		}
	}
	sort.Slice(removed, func(i, j int) bool { return removed[i].PeerID < removed[j].PeerID })
	return removed
}

// FormatPeerUpdated is the log line when a neighbor restarts (new instance_id).
func FormatPeerUpdated(peerID, oldInstance, newInstance string) string {
	return fmt.Sprintf(
		"peer_updated peer_id=%s old_instance=%s new_instance=%s",
		peerID, oldInstance, newInstance,
	)
}

// FormatPeerGone is the log line when a neighbor is pruned after TTL (P034).
func FormatPeerGone(peerID string) string {
	return fmt.Sprintf("peer_gone peer_id=%s", peerID)
}
