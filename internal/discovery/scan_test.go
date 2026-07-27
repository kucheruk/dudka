package discovery_test

import (
	"context"
	"testing"
	"time"

	"dudka/internal/discovery"
)

func TestScanFindsPeerWithoutBroadcast(t *testing.T) {
	t.Parallel()

	// Two nodes: no UDP cross-talk (different announce ports, no Target).
	// Discovery must happen only via Scan → TCP register.
	storeA := discovery.NewPeerStore()
	storeB := discovery.NewPeerStore()

	nodeB := discovery.NewNode(discovery.Config{
		PeerID:      "peer-b",
		DisplayName: "Bob",
		InstanceID:  "inst-b",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    time.Hour, // effectively no announce help
		Peers:       storeB,
	})
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeB.Stop() })

	nodeA := discovery.NewNode(discovery.Config{
		PeerID:      "peer-a",
		DisplayName: "Alice",
		InstanceID:  "inst-a",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    time.Hour,
		Peers:       storeA,
	})
	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeA.Stop() })

	if len(storeA.List()) != 0 {
		t.Fatalf("pre-scan peers should be empty: %+v", storeA.List())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := nodeA.Scan(ctx, discovery.ScanRequest{
		Hosts: []string{"127.0.0.1"},
		Port:  nodeB.TCPPort(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Found != 1 {
		t.Fatalf("Found=%d want 1; peers=%+v", res.Found, storeA.List())
	}
	if !peerNamed(storeA, "peer-b") {
		t.Fatalf("Alice missing Bob after scan: %+v", storeA.List())
	}
}
