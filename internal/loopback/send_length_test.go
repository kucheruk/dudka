package loopback_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/loopback"
)

func TestSendOversizedReturns4xx(t *testing.T) {
	t.Parallel()

	hub := chat.NewHub(chat.Config{
		PeerID: "local-peer",
		Name:   "Local",
		Store:  chat.NewStore(),
		Peers:  discovery.NewPeerStore(),
		Dialer: net.DialTimeout,
	})
	api := loopback.New("local-peer", "Local")
	api.SetChat(hub)

	ln, err := api.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() { _ = api.Serve(ln) }()

	body, err := json.Marshal(map[string]string{
		"text": strings.Repeat("a", chat.MaxTextCodePoints+1),
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post("http://"+ln.Addr().String()+"/send", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 400 || resp.StatusCode > 499 {
		t.Fatalf("status=%d body=%s", resp.StatusCode, raw)
	}
	if !strings.Contains(string(raw), "4000") {
		t.Fatalf("body should explain limit: %s", raw)
	}
	if len(hub.Messages()) != 0 {
		t.Fatal("oversized must not land in messages")
	}
}
