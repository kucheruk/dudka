package discovery

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegisterCarriesReverseBacklog(t *testing.T) {
	t.Parallel()
	const frame = "{\"type\":\"chat\",\"text\":\"reverse\"}\n"
	nodeA := NewNode(Config{
		PeerID:      "a",
		DisplayName: "A",
		InstanceID:  "ia",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    time.Hour,
		OnRegisterBacklog: func(Peer) [][]byte {
			return [][]byte{[]byte(frame)}
		},
	})
	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeA.Stop()

	got := make(chan string, 1)
	nodeB := NewNode(Config{
		PeerID:      "b",
		DisplayName: "B",
		InstanceID:  "ib",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    time.Hour,
		OnChatLine: func(_ string, line []byte) {
			got <- string(line)
		},
	})
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeB.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := nodeB.Scan(ctx, ScanRequest{Hosts: []string{"127.0.0.1"}, Port: nodeA.TCPPort()}); err != nil {
		t.Fatal(err)
	}
	select {
	case line := <-got:
		if line != frame {
			t.Fatalf("line=%q want %q", line, frame)
		}
	case <-ctx.Done():
		t.Fatal("reverse backlog was not delivered")
	}
}

func TestKnownPeerReregistersForNewReverseBacklog(t *testing.T) {
	t.Parallel()
	const frame = "{\"type\":\"chat\",\"text\":\"after discovery\"}\n"
	var ready atomic.Bool
	nodeA := NewNode(Config{
		PeerID:      "a",
		DisplayName: "A",
		InstanceID:  "ia",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    time.Hour,
		OnRegisterBacklog: func(Peer) [][]byte {
			if ready.Load() {
				return [][]byte{[]byte(frame)}
			}
			return nil
		},
	})
	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeA.Stop()

	got := make(chan string, 1)
	nodeB := NewNode(Config{
		PeerID:      "b",
		DisplayName: "B",
		InstanceID:  "ib",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    time.Hour,
		OnChatLine: func(_ string, line []byte) {
			got <- string(line)
		},
	})
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	defer nodeB.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := nodeB.Scan(ctx, ScanRequest{Hosts: []string{"127.0.0.1"}, Port: nodeA.TCPPort()}); err != nil {
		t.Fatal(err)
	}

	ready.Store(true)
	nodeB.maybeRegister(Announce{
		PeerID:     "a",
		InstanceID: "ia",
		ProtoMajor: DefaultProtoMajor,
		TCPPort:    nodeA.TCPPort(),
	}, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: DefaultUDPPort})

	select {
	case line := <-got:
		if line != frame {
			t.Fatalf("line=%q want %q", line, frame)
		}
	case <-ctx.Done():
		t.Fatal("known peer did not fetch new reverse backlog")
	}
}
