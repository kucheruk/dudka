package chat_test

import (
	"net"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
)

// P030 / DUD-CHAT-101: POST-equivalent Send fans out; second peer sees text ≤ 2s.
func TestFanoutSecondPeerSeesMessageWithin2s(t *testing.T) {
	t.Parallel()

	storeB := discovery.NewPeerStore()
	msgsB := chat.NewStore()
	hubB := chat.NewHub(chat.Config{
		PeerID:  "peer-b",
		Name:    "Bob",
		Store:   msgsB,
		Peers:   storeB,
		Dialer:  net.DialTimeout,
		Timeout: time.Second,
	})

	nodeB := discovery.NewNode(discovery.Config{
		PeerID:      "peer-b",
		DisplayName: "Bob",
		InstanceID:  "inst-b",
		Bind:        "127.0.0.1:0",
		TCPBind:     "127.0.0.1:0",
		Interval:    time.Hour,
		Peers:       storeB,
		OnChatLine:  hubB.HandleChatLine,
	})
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeB.Stop() })

	storeA := discovery.NewPeerStore()
	_ = storeA.Upsert(discovery.Peer{
		PeerID:      "peer-b",
		DisplayName: "Bob",
		InstanceID:  "inst-b",
		Host:        "127.0.0.1",
		TCPPort:     nodeB.TCPPort(),
		LastSeen:    time.Now().UTC(),
	})
	msgsA := chat.NewStore()
	hubA := chat.NewHub(chat.Config{
		PeerID:  "peer-a",
		Name:    "Alice",
		Store:   msgsA,
		Peers:   storeA,
		Dialer:  net.DialTimeout,
		Timeout: time.Second,
	})

	msg, err := hubA.Send("hello lan")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Text != "hello lan" || msg.MsgID == "" {
		t.Fatalf("bad local msg %+v", msg)
	}
	local := msgsA.List()
	if len(local) != 1 || local[0].Text != "hello lan" {
		t.Fatalf("sender store=%+v", local)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got := msgsB.List()
		if len(got) == 1 && got[0].Text == "hello lan" && got[0].MsgID == msg.MsgID && got[0].PeerID == "peer-a" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("peer B missing message within 2s: %v", msgsB.List())
}
