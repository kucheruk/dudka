package loopback_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/loopback"
)

func TestSendResponseBestEffortStatusesOnly(t *testing.T) {
	t.Parallel()

	peers := discovery.NewPeerStore()
	_ = peers.Upsert(discovery.Peer{
		PeerID: "other", Host: "127.0.0.1", TCPPort: 9, LastSeen: time.Now().UTC(),
	})
	hub := chat.NewHub(chat.Config{
		PeerID: "local-peer",
		Name:   "Local",
		Store:  chat.NewStore(),
		Peers:  peers,
		Dialer: func(network, address string, timeout time.Duration) (net.Conn, error) {
			return nil, net.ErrClosed
		},
	})
	api := loopback.New("local-peer", "Local")
	api.SetChat(hub)

	ln, err := api.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() { _ = api.Serve(ln) }()

	resp, err := http.Post("http://"+ln.Addr().String()+"/send", "application/json", bytes.NewReader([]byte(`{"text":"yo"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	st, _ := got["status"].(string)
	if !chat.IsBestEffortStatus(st) {
		t.Fatalf("status=%q body=%s", st, body)
	}
	if _, ok := got["queued"]; !ok {
		t.Fatalf("missing queued field: %s", body)
	}
	lower := strings.ToLower(string(body))
	for _, bad := range []string{"delivered", "доставлено"} {
		if strings.Contains(lower, bad) {
			t.Fatalf("API claimed delivery: %s", body)
		}
	}
}
