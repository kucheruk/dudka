package chat_test

import (
	"net"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
)

// P033 / DUD-CHAT-120: third peer after register gets GET /tail matching keeper.
func TestThirdPeerTailMatchesKeeper(t *testing.T) {
	t.Parallel()

	storeA := discovery.NewPeerStore()
	storeB := discovery.NewPeerStore()
	storeC := discovery.NewPeerStore()
	msgsA := chat.NewStore()
	msgsB := chat.NewStore()
	msgsC := chat.NewStore()

	hubA := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: msgsA, Peers: storeA,
		Dialer: net.DialTimeout, Timeout: time.Second,
	})
	hubB := chat.NewHub(chat.Config{
		PeerID: "peer-b", Name: "Bob", Store: msgsB, Peers: storeB,
		Dialer: net.DialTimeout, Timeout: time.Second,
	})
	hubC := chat.NewHub(chat.Config{
		PeerID: "peer-c", Name: "Carol", Store: msgsC, Peers: storeC,
		Dialer: net.DialTimeout, Timeout: time.Second,
	})

	nodeA := discovery.NewNode(discovery.Config{
		PeerID: "peer-a", DisplayName: "Alice", InstanceID: "inst-a",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: 80 * time.Millisecond,
		Peers: storeA, OnChatLine: hubA.HandleChatLine, OnTailRequest: hubA.HandleTailRequest,
		OnPeerUpserted: hubA.OnPeerUpserted,
	})
	nodeB := discovery.NewNode(discovery.Config{
		PeerID: "peer-b", DisplayName: "Bob", InstanceID: "inst-b",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: 80 * time.Millisecond,
		Peers: storeB, OnChatLine: hubB.HandleChatLine, OnTailRequest: hubB.HandleTailRequest,
		OnPeerUpserted: hubB.OnPeerUpserted,
	})
	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeA.Stop() })
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeB.Stop() })
	nodeA.SetTarget(nodeB.LocalAddr().String())
	nodeB.SetTarget(nodeA.LocalAddr().String())

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if peerNamed(storeA, "peer-b") && peerNamed(storeB, "peer-a") {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if !peerNamed(storeA, "peer-b") || !peerNamed(storeB, "peer-a") {
		t.Fatal("A/B register failed")
	}

	for _, text := range []string{"one", "two", "three"} {
		if _, err := hubA.Send(text); err != nil {
			t.Fatal(err)
		}
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(msgsB.List()) >= 3 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(msgsB.List()) < 3 {
		t.Fatalf("B missing fanout: %v", msgsB.List())
	}

	nodeC := discovery.NewNode(discovery.Config{
		PeerID: "peer-c", DisplayName: "Carol", InstanceID: "inst-c",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: 80 * time.Millisecond,
		Target: nodeA.LocalAddr().String(),
		Peers:  storeC, OnChatLine: hubC.HandleChatLine, OnTailRequest: hubC.HandleTailRequest,
		OnPeerUpserted: hubC.OnPeerUpserted,
	})
	if err := nodeC.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeC.Stop() })

	deadline = time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		tailC := hubC.Tail()
		tailA := hubA.Tail()
		if tailC.KeeperID == "peer-a" && !tailC.IsKeeper &&
			len(tailC.Messages) == len(tailA.Messages) && len(tailC.Messages) >= 3 &&
			sameMsgIDs(tailC.Messages, tailA.Messages) {
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
	t.Fatalf("C tail mismatch: C=%+v A=%+v", hubC.Tail(), hubA.Tail())
}

func peerNamed(store *discovery.PeerStore, id string) bool {
	for _, p := range store.List() {
		if p.PeerID == id {
			return true
		}
	}
	return false
}

func sameMsgIDs(a, b []chat.Message) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].MsgID != b[i].MsgID || a[i].Text != b[i].Text {
			return false
		}
	}
	return true
}
