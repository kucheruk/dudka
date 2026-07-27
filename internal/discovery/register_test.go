package discovery_test

import (
	"net"
	"testing"
	"time"

	"dudka/internal/discovery"
)

func TestRegisterRoundTrip(t *testing.T) {
	t.Parallel()
	in := discovery.Register{
		PeerID:      "p1",
		DisplayName: "Аня",
		ProtoMajor:  1,
		ProtoMinor:  0,
		TCPPort:     41777,
		InstanceID:  "i1",
	}
	raw, err := discovery.EncodeRegister(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := discovery.DecodeRegister(raw)
	if err != nil {
		t.Fatal(err)
	}
	in.Type = "register"
	if out != in {
		t.Fatalf("got %+v want %+v", out, in)
	}
}

func TestPeerStoreUpsertAndList(t *testing.T) {
	t.Parallel()
	s := discovery.NewPeerStore()
	_ = s.Upsert(discovery.Peer{
		PeerID: "p1", DisplayName: "A", InstanceID: "i1",
		Host: "127.0.0.1", TCPPort: 1, LastSeen: time.Now(),
	})
	_ = s.Upsert(discovery.Peer{
		PeerID: "p1", DisplayName: "A2", InstanceID: "i1",
		Host: "127.0.0.1", TCPPort: 1, LastSeen: time.Now(),
	})
	list := s.List()
	if len(list) != 1 || list[0].DisplayName != "A2" {
		t.Fatalf("list=%+v", list)
	}
}

func TestAnnounceTriggersMutualRegister(t *testing.T) {
	t.Parallel()

	storeA := discovery.NewPeerStore()
	storeB := discovery.NewPeerStore()

	nodeA := discovery.NewNode(discovery.Config{
		PeerID:      "peer-a",
		DisplayName: "Alice",
		InstanceID:  "inst-a",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    80 * time.Millisecond,
		Peers:       storeA,
	})
	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeA.Stop() })

	addrA := nodeA.LocalAddr().(*net.UDPAddr)
	nodeB := discovery.NewNode(discovery.Config{
		PeerID:      "peer-b",
		DisplayName: "Bob",
		InstanceID:  "inst-b",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    80 * time.Millisecond,
		Target:      addrA.String(),
		Peers:       storeB,
	})
	// Point A at B after B starts.
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeB.Stop() })
	nodeA.SetTarget(nodeB.LocalAddr().String())

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		aHasB := peerNamed(storeA, "peer-b")
		bHasA := peerNamed(storeB, "peer-a")
		if aHasB && bHasA {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("mutual register failed: A=%v B=%v", storeA.List(), storeB.List())
}

func peerNamed(s *discovery.PeerStore, id string) bool {
	for _, p := range s.List() {
		if p.PeerID == id {
			return true
		}
	}
	return false
}
