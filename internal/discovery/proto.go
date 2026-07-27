package discovery

import (
	"fmt"
	"sync"
	"time"
)

// CompatibleProto reports whether two peers share a wire major version.
func CompatibleProto(localMajor, remoteMajor int) bool {
	if localMajor == 0 {
		localMajor = DefaultProtoMajor
	}
	if remoteMajor == 0 {
		remoteMajor = DefaultProtoMajor
	}
	return localMajor == remoteMajor
}

// FormatProtoMismatch is the log/status line for P023.
func FormatProtoMismatch(peerID string, theirs, ours int) string {
	return fmt.Sprintf(
		"proto_mismatch peer_id=%s theirs=%d ours=%d",
		peerID, theirs, ours,
	)
}

// IncompatiblePeer is a neighbor rejected for proto_major mismatch.
type IncompatiblePeer struct {
	PeerID     string    `json:"peer_id"`
	ProtoMajor int       `json:"proto_major"`
	SeenAt     time.Time `json:"seen_at"`
}

// Status is a snapshot of local discovery proto health (P023) and LAN (P044).
type Status struct {
	ProtoMajor   int                `json:"proto_major"`
	ProtoMinor   int                `json:"proto_minor"`
	Network      string             `json:"network"` // ok | no_network (DUD-NET-140)
	Incompatible []IncompatiblePeer `json:"incompatible"`
}

type protoBook struct {
	mu    sync.Mutex
	byID  map[string]IncompatiblePeer
	order []string
}

func newProtoBook() *protoBook {
	return &protoBook{byID: make(map[string]IncompatiblePeer)}
}

func (b *protoBook) note(peerID string, major int) {
	if peerID == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.byID[peerID]; !ok {
		b.order = append(b.order, peerID)
	}
	b.byID[peerID] = IncompatiblePeer{
		PeerID:     peerID,
		ProtoMajor: major,
		SeenAt:     time.Now().UTC(),
	}
	// Cap memory.
	const max = 32
	for len(b.order) > max {
		old := b.order[0]
		b.order = b.order[1:]
		delete(b.byID, old)
	}
}

func (b *protoBook) list() []IncompatiblePeer {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]IncompatiblePeer, 0, len(b.order))
	for _, id := range b.order {
		if p, ok := b.byID[id]; ok {
			out = append(out, p)
		}
	}
	return out
}
