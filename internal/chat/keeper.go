package chat

import (
	"dudka/internal/discovery"
)

// SelectTailKeeper returns the lexicographically minimal non-empty peer_id
// among online candidates (DUD-CHAT-121 / P032). Empty input → ("", false).
func SelectTailKeeper(peerIDs []string) (string, bool) {
	keeper := ""
	found := false
	for _, id := range peerIDs {
		if id == "" {
			continue
		}
		if !found || id < keeper {
			keeper = id
			found = true
		}
	}
	return keeper, found
}

// SelectTailKeeperAmong chooses the keeper among local peer_id and known peers.
func SelectTailKeeperAmong(selfPeerID string, peers []discovery.Peer) (string, bool) {
	ids := make([]string, 0, len(peers)+1)
	if selfPeerID != "" {
		ids = append(ids, selfPeerID)
	}
	for _, p := range peers {
		if p.PeerID != "" {
			ids = append(ids, p.PeerID)
		}
	}
	return SelectTailKeeper(ids)
}
