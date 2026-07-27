package loopback_test

import (
	"encoding/json"
	"net"
	"net/http"
	"testing"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/loopback"
)

func TestGetTailEndpoint(t *testing.T) {
	t.Parallel()

	store := chat.NewStore()
	_ = store.Append(chat.Message{MsgID: "m1", PeerID: "local-peer", Text: "hi"})
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

	resp, err := http.Get("http://" + ln.Addr().String() + "/tail")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var view chat.TailView
	if err := json.NewDecoder(resp.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.KeeperID != "local-peer" || !view.IsKeeper {
		t.Fatalf("view=%+v", view)
	}
	if len(view.Messages) != 1 || view.Messages[0].Text != "hi" {
		t.Fatalf("messages=%+v", view.Messages)
	}
}
