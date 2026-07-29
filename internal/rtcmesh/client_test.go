package rtcmesh

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/signaling"
)

func TestNativePeersExchangeBrowserCompatibleChat(t *testing.T) {
	signalServer := httptest.NewServer(signaling.NewServer("https://zamoo.team").Handler())
	defer signalServer.Close()
	signalURL := "ws" + strings.TrimPrefix(signalServer.URL, "http")

	received := make(chan chat.Message, 1)
	peersA := discovery.NewPeerStore()
	peersB := discovery.NewPeerStore()
	clientA := New(Config{
		PeerID: "native-a", Name: "Мак", Peers: peersA,
		SignalURL: signalURL, Origin: "https://zamoo.team",
		STUNURL: "stun:127.0.0.1:9",
	})
	clientB := New(Config{
		PeerID: "native-b", Name: "Линукс", Peers: peersB,
		SignalURL: signalURL, Origin: "https://zamoo.team",
		STUNURL: "stun:127.0.0.1:9",
		OnMessage: func(raw []byte) {
			if message, err := chat.DecodeMessage(raw); err == nil {
				received <- message
			}
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clientA.Start(ctx)
	clientB.Start(ctx)
	defer clientA.Stop()
	defer clientB.Stop()

	waitFor(t, 10*time.Second, func() bool {
		return len(peersA.List()) == 1 && len(peersB.List()) == 1
	})
	message := chat.Message{
		Type: chat.TypeChat, MsgID: "message-1", PeerID: "native-a",
		DisplayNameAtSend: "Мак", Text: "проверка", TS: time.Now().UTC(),
	}
	if queued := clientA.Broadcast(message); queued != 1 {
		t.Fatalf("queued=%d, want 1", queued)
	}
	select {
	case got := <-received:
		if got.MsgID != message.MsgID || got.Text != message.Text ||
			got.DisplayNameAtSend != message.DisplayNameAtSend {
			t.Fatalf("received=%+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("chat was not received")
	}
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("condition timeout")
}
