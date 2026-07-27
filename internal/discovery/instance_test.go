package discovery_test

import (
	"strings"
	"testing"
	"time"

	"dudka/internal/discovery"
)

func TestUpsertInstanceChangeMarksUpdatedNoDuplicate(t *testing.T) {
	t.Parallel()
	s := discovery.NewPeerStore()
	s.Upsert(discovery.Peer{
		PeerID: "p1", DisplayName: "A", InstanceID: "inst-1",
		Host: "127.0.0.1", TCPPort: 1, LastSeen: time.Now(),
	})

	res := s.Upsert(discovery.Peer{
		PeerID: "p1", DisplayName: "A", InstanceID: "inst-2",
		Host: "127.0.0.1", TCPPort: 2, LastSeen: time.Now(),
	})
	if !res.InstanceChanged {
		t.Fatal("expected InstanceChanged")
	}
	list := s.List()
	if len(list) != 1 {
		t.Fatalf("want 1 peer (no zombies), got %d: %+v", len(list), list)
	}
	if list[0].InstanceID != "inst-2" || !list[0].Updated {
		t.Fatalf("peer=%+v", list[0])
	}
}

func TestUpsertSameInstanceClearsUpdated(t *testing.T) {
	t.Parallel()
	s := discovery.NewPeerStore()
	s.Upsert(discovery.Peer{PeerID: "p1", InstanceID: "i1", Host: "h", TCPPort: 1})
	s.Upsert(discovery.Peer{PeerID: "p1", InstanceID: "i2", Host: "h", TCPPort: 1})
	res := s.Upsert(discovery.Peer{PeerID: "p1", InstanceID: "i2", Host: "h", TCPPort: 1})
	if res.InstanceChanged {
		t.Fatal("same instance should not change")
	}
	if s.List()[0].Updated {
		t.Fatal("Updated should clear on same-instance upsert")
	}
}

func TestFormatPeerUpdated(t *testing.T) {
	t.Parallel()
	line := discovery.FormatPeerUpdated("p1", "old", "new")
	if !strings.Contains(line, "peer_updated") || !strings.Contains(line, "peer_id=p1") {
		t.Fatalf("line=%q", line)
	}
	if !strings.Contains(line, "old_instance=old") || !strings.Contains(line, "new_instance=new") {
		t.Fatalf("line=%q", line)
	}
}
