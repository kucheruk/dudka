package loopback_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/loopback"
)

func TestPostFileAnnounceAppearsInMessages(t *testing.T) {
	t.Parallel()
	peers := discovery.NewPeerStore()
	hub := chat.NewHub(chat.Config{
		PeerID: "local-peer",
		Name:   "Local",
		Store:  chat.NewStore(),
		Peers:  peers,
	})
	api := loopback.New("local-peer", "Local")
	api.SetChat(hub)
	api.SetPeers(peers)
	ln, err := api.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() { _ = api.Serve(ln) }()
	base := "http://" + ln.Addr().String()

	body := `{"name":"doc.pdf","size":99,"mime":"application/pdf","hash":"sha256:ff"}`
	resp, err := http.Post(base+"/files/announce", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, raw)
	}
	var res chat.SendResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if res.Message.Type != chat.TypeFileAnnounce || res.Message.FileID == "" {
		t.Fatalf("res=%+v", res)
	}
	if res.Message.FileName != "doc.pdf" || res.Message.Size != 99 {
		t.Fatalf("msg=%+v", res.Message)
	}

	get, err := http.Get(base + "/messages")
	if err != nil {
		t.Fatal(err)
	}
	defer get.Body.Close()
	var env struct {
		Messages []chat.Message `json:"messages"`
	}
	if err := json.NewDecoder(get.Body).Decode(&env); err != nil {
		t.Fatal(err)
	}
	if len(env.Messages) != 1 || env.Messages[0].FileID != res.Message.FileID {
		t.Fatalf("messages=%+v", env.Messages)
	}
	// No auto-download surface in P050.
	dl, err := http.Get(base + "/files/" + res.Message.FileID)
	if err != nil {
		t.Fatal(err)
	}
	defer dl.Body.Close()
	if dl.StatusCode == http.StatusOK {
		t.Fatal("P050 must not auto-serve full file bytes")
	}
	_ = time.Now()
}
