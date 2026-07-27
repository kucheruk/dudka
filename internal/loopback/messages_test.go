package loopback_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/loopback"
)

func TestSendAndMessagesEndpoints(t *testing.T) {
	t.Parallel()

	store := chat.NewStore()
	peers := discovery.NewPeerStore()
	hub := chat.NewHub(chat.Config{
		PeerID: "local-peer",
		Name:   "Local",
		Store:  store,
		Peers:  peers,
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

	base := "http://" + ln.Addr().String()

	resp, err := http.Post(base+"/send", "application/json", bytes.NewReader([]byte(`{"text":"yo"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	var sent map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&sent); err != nil {
		t.Fatal(err)
	}
	if sent["status"] != "accepted" {
		t.Fatalf("status=%v", sent["status"])
	}

	get, err := http.Get(base + "/messages")
	if err != nil {
		t.Fatal(err)
	}
	defer get.Body.Close()
	var envelope struct {
		Messages []chat.Message `json:"messages"`
	}
	if err := json.NewDecoder(get.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Messages) != 1 || envelope.Messages[0].Text != "yo" {
		t.Fatalf("messages=%+v", envelope.Messages)
	}
	if envelope.Messages[0].DisplayNameAtSend != "Local" {
		t.Fatalf("name_at_send=%q", envelope.Messages[0].DisplayNameAtSend)
	}
	if envelope.Messages[0].TS.IsZero() {
		t.Fatal("missing ts")
	}
	_ = time.Now()
}
