package chat_test

import (
	"net"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
)

// P034: keeper leaves → re-election → third peer still gets tail from new keeper.
func TestKeeperLeaveNewPeerGetsTail(t *testing.T) {
	t.Parallel()

	storeA := discovery.NewPeerStore()
	storeB := discovery.NewPeerStore()
	storeC := discovery.NewPeerStore()
	msgsA := chat.NewStore()
	msgsB := chat.NewStore()
	msgsC := chat.NewStore()

	hubA := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: msgsA, Peers: storeA,
		Dialer: net.DialTimeout, Timeout: time.Second,
	})
	hubB := chat.NewHub(chat.Config{
		PeerID: "peer-b", Name: "Bob", Store: msgsB, Peers: storeB,
		Dialer: net.DialTimeout, Timeout: time.Second,
	})
	hubC := chat.NewHub(chat.Config{
		PeerID: "peer-c", Name: "Carol", Store: msgsC, Peers: storeC,
		Dialer: net.DialTimeout, Timeout: time.Second,
	})

	ttl := 350 * time.Millisecond
	interval := 60 * time.Millisecond

	nodeA := discovery.NewNode(discovery.Config{
		PeerID: "peer-a", DisplayName: "Alice", InstanceID: "inst-a",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: interval, PeerTTL: ttl,
		Peers: storeA, OnChatLine: hubA.HandleChatLine, OnTailRequest: hubA.HandleTailRequest,
		OnPeerUpserted: hubA.OnPeerUpserted, OnPeerRemoved: hubA.OnPeerRemoved,
	})
	nodeB := discovery.NewNode(discovery.Config{
		PeerID: "peer-b", DisplayName: "Bob", InstanceID: "inst-b",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: interval, PeerTTL: ttl,
		Peers: storeB, OnChatLine: hubB.HandleChatLine, OnTailRequest: hubB.HandleTailRequest,
		OnPeerUpserted: hubB.OnPeerUpserted, OnPeerRemoved: hubB.OnPeerRemoved,
	})
	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeB.Stop() })
	nodeA.SetTarget(nodeB.LocalAddr().String())
	nodeB.SetTarget(nodeA.LocalAddr().String())

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if peerNamed(storeA, "peer-b") && peerNamed(storeB, "peer-a") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !peerNamed(storeB, "peer-a") {
		t.Fatal("B missing A before leave")
	}

	for _, text := range []string{"keep-1", "keep-2"} {
		if _, err := hubA.Send(text); err != nil {
			t.Fatal(err)
		}
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(msgsB.List()) >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(msgsB.List()) < 2 {
		t.Fatalf("B missing fanout: %v", msgsB.List())
	}
	if hubA.Tail().KeeperID != "peer-a" || !hubA.Tail().IsKeeper {
		t.Fatalf("expected A keeper: %+v", hubA.Tail())
	}

	// Keeper leaves.
	if err := nodeA.Stop(); err != nil {
		t.Fatal(err)
	}

	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		tailB := hubB.Tail()
		if !peerNamed(storeB, "peer-a") && tailB.KeeperID == "peer-b" && tailB.IsKeeper {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if peerNamed(storeB, "peer-a") {
		t.Fatal("stale keeper peer-a still in B table")
	}
	if hubB.Tail().KeeperID != "peer-b" || !hubB.Tail().IsKeeper {
		t.Fatalf("B should be new keeper: %+v", hubB.Tail())
	}

	// Third peer joins after re-election.
	nodeC := discovery.NewNode(discovery.Config{
		PeerID: "peer-c", DisplayName: "Carol", InstanceID: "inst-c",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: interval, PeerTTL: ttl,
		Target: nodeB.LocalAddr().String(),
		Peers:  storeC, OnChatLine: hubC.HandleChatLine, OnTailRequest: hubC.HandleTailRequest,
		OnPeerUpserted: hubC.OnPeerUpserted, OnPeerRemoved: hubC.OnPeerRemoved,
	})
	if err := nodeC.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeC.Stop() })

	deadline = time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		tailC := hubC.Tail()
		tailB := hubB.Tail()
		if tailC.KeeperID == "peer-b" && !tailC.IsKeeper &&
			len(tailC.Messages) == len(tailB.Messages) && len(tailC.Messages) >= 2 &&
			sameMsgIDs(tailC.Messages, tailB.Messages) {
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
	t.Fatalf("C tail != new keeper B: C=%+v B=%+v", hubC.Tail(), hubB.Tail())
}

func TestPeerStorePruneRemovesStale(t *testing.T) {
	t.Parallel()
	s := discovery.NewPeerStore()
	_ = s.Upsert(discovery.Peer{PeerID: "old", LastSeen: time.Now().UTC().Add(-time.Second)})
	_ = s.Upsert(discovery.Peer{PeerID: "fresh", LastSeen: time.Now().UTC()})
	removed := s.PruneOlderThan(time.Now().UTC().Add(-500 * time.Millisecond))
	if len(removed) != 1 || removed[0].PeerID != "old" {
		t.Fatalf("removed=%+v", removed)
	}
	if peerNamed(s, "old") || !peerNamed(s, "fresh") {
		t.Fatalf("list=%+v", s.List())
	}
}
