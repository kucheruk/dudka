package discovery_test

import (
	"net"
	"sync"
	"testing"
	"time"

	"dudka/internal/discovery"
)

func TestNodeAnnounceHeardByPeer(t *testing.T) {
	t.Parallel()

	var got discovery.Announce
	var from string
	var mu sync.Mutex
	done := make(chan struct{})

	recv := discovery.NewNode(discovery.Config{
		PeerID:      "receiver-id",
		DisplayName: "Recv",
		InstanceID:  "recv-instance",
		Bind:        "127.0.0.1:0",
		Interval:    time.Hour,
		Target:      "127.0.0.1:1",
		OnAnnounce: func(a discovery.Announce, addr net.Addr) {
			mu.Lock()
			defer mu.Unlock()
			if a.PeerID == "sender-id" {
				got = a
				from = addr.String()
				select {
				case <-done:
				default:
					close(done)
				}
			}
		},
	})
	if err := recv.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = recv.Stop() })

	recvAddr := recv.LocalAddr()
	if recvAddr == nil {
		t.Fatal("receiver has no local addr")
	}

	send := discovery.NewNode(discovery.Config{
		PeerID:      "sender-id",
		DisplayName: "Send",
		InstanceID:  "send-instance",
		ProtoMajor:  1,
		TCPPort:     41777,
		Bind:        "127.0.0.1:0",
		Interval:    50 * time.Millisecond,
		Target:      recvAddr.String(),
	})
	if err := send.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = send.Stop() })

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for announce_rx")
	}

	mu.Lock()
	defer mu.Unlock()
	if got.PeerID != "sender-id" || got.DisplayName != "Send" {
		t.Fatalf("got %+v", got)
	}
	if from == "" {
		t.Fatal("missing from addr")
	}
}
