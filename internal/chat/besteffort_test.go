package chat_test

import (
	"encoding/json"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
)

func TestSendStatusIsAcceptedOrQueuedOnly(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(chat.Config{
		PeerID: "p",
		Name:   "N",
		Store:  chat.NewStore(),
		Peers:  discovery.NewPeerStore(),
	})
	res, err := hub.Send("alone")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != chat.StatusAccepted {
		t.Fatalf("status=%q want %q when no peers", res.Status, chat.StatusAccepted)
	}
	if res.Queued != 0 {
		t.Fatalf("queued=%d", res.Queued)
	}
	if !chat.IsBestEffortStatus(res.Status) {
		t.Fatalf("status not allowed: %q", res.Status)
	}
}

func TestSendQueuedWhenPeersPresent(t *testing.T) {
	t.Parallel()

	peers := discovery.NewPeerStore()
	_ = peers.Upsert(discovery.Peer{
		PeerID: "other", Host: "127.0.0.1", TCPPort: 9, LastSeen: time.Now().UTC(),
	})
	hub := chat.NewHub(chat.Config{
		PeerID: "p", Name: "N", Store: chat.NewStore(), Peers: peers,
		Dialer: func(network, address string, timeout time.Duration) (net.Conn, error) {
			return nil, net.ErrClosed // fanout may fail; still queued, not delivered
		},
	})
	res, err := hub.Send("hi")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != chat.StatusQueued {
		t.Fatalf("status=%q want %q", res.Status, chat.StatusQueued)
	}
	if res.Queued != 1 {
		t.Fatalf("queued=%d", res.Queued)
	}
	raw, _ := json.Marshal(res)
	lower := strings.ToLower(string(raw))
	for _, bad := range []string{"delivered", "deliver_ok", "доставлено"} {
		if strings.Contains(lower, bad) {
			t.Fatalf("send result must not claim delivery: %s in %s", bad, raw)
		}
	}
}

func TestSendLogsAvoidDeliveryClaims(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var logs []string
	peers := discovery.NewPeerStore()
	_ = peers.Upsert(discovery.Peer{
		PeerID: "other", Host: "127.0.0.1", TCPPort: 9, LastSeen: time.Now().UTC(),
	})
	hub := chat.NewHub(chat.Config{
		PeerID: "p", Name: "N", Store: chat.NewStore(), Peers: peers,
		Timeout: 50 * time.Millisecond,
		Dialer: func(network, address string, timeout time.Duration) (net.Conn, error) {
			return nil, net.ErrClosed
		},
		Logf: func(format string, args ...any) {
			mu.Lock()
			logs = append(logs, strings.ToLower(strings.TrimSpace(format)))
			mu.Unlock()
		},
	})
	if _, err := hub.Send("log-check"); err != nil {
		t.Fatal(err)
	}
	// Allow fanout goroutine to log dial err.
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	joined := strings.Join(logs, "\n")
	for _, bad := range []string{"deliver", "delivered", "доставлено", "chat_deliver"} {
		if strings.Contains(joined, bad) {
			t.Fatalf("forbidden delivery claim %q in logs: %v", bad, logs)
		}
	}
	ok := false
	for _, line := range logs {
		if strings.Contains(line, "chat_accepted") || strings.Contains(line, "chat_queued") {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("expected chat_accepted/chat_queued log, got %v", logs)
	}
}
