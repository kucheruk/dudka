package chat_test

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
)

func TestEncodeDecodeFileAnnounce(t *testing.T) {
	t.Parallel()
	in := chat.Message{
		Type:              chat.TypeFileAnnounce,
		MsgID:             "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		PeerID:            "peer-src",
		DisplayNameAtSend: "Аня",
		TS:                time.Date(2026, 7, 27, 15, 0, 0, 0, time.UTC),
		FileID:            "ffffffff-ffff-4fff-8fff-ffffffffffff",
		FileName:          "photo.jpg",
		Size:              42,
		Mime:              "image/jpeg",
		Hash:              "sha256:deadbeef",
	}
	raw, err := chat.EncodeMessage(in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"text"`) && strings.Contains(string(raw), `"text":"`) {
		// empty text may be omitted; binary payload must not appear
	}
	if strings.Contains(string(raw), "\x00") || len(raw) > 2048 {
		t.Fatalf("announce wire must stay metadata-only, got len=%d", len(raw))
	}
	out, err := chat.DecodeMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Type != chat.TypeFileAnnounce {
		t.Fatalf("type=%q", out.Type)
	}
	if out.FileID != in.FileID || out.FileName != in.FileName || out.Size != in.Size ||
		out.Mime != in.Mime || out.Hash != in.Hash || out.PeerID != in.PeerID {
		t.Fatalf("got %+v want %+v", out, in)
	}
	if out.Text != "" {
		t.Fatalf("file announce must not carry chat text, got %q", out.Text)
	}
}

func TestValidateFileAnnounceRequiresFields(t *testing.T) {
	t.Parallel()
	err := chat.ValidateFileAnnounce(chat.FileAnnounce{
		Name: "x", Size: 1, Mime: "application/octet-stream", Hash: "sha256:aa",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := chat.ValidateFileAnnounce(chat.FileAnnounce{}); err == nil {
		t.Fatal("empty announce must fail")
	}
	if err := chat.ValidateFileAnnounce(chat.FileAnnounce{
		Name: "x", Size: -1, Mime: "a/b", Hash: "h",
	}); err == nil {
		t.Fatal("negative size must fail")
	}
}

func TestAnnounceFileFanoutSecondPeerSeesMetadataOnly(t *testing.T) {
	t.Parallel()

	storeB := discovery.NewPeerStore()
	msgsB := chat.NewStore()
	hubB := chat.NewHub(chat.Config{
		PeerID: "peer-b", Name: "Bob", Store: msgsB, Peers: storeB,
		Dialer: net.DialTimeout, Timeout: time.Second,
	})
	nodeB := discovery.NewNode(discovery.Config{
		PeerID: "peer-b", DisplayName: "Bob", InstanceID: "inst-b",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: time.Hour,
		Peers: storeB, OnChatLine: hubB.HandleChatLine,
	})
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeB.Stop() })

	storeA := discovery.NewPeerStore()
	_ = storeA.Upsert(discovery.Peer{
		PeerID: "peer-b", DisplayName: "Bob", InstanceID: "inst-b",
		Host: "127.0.0.1", TCPPort: nodeB.TCPPort(), LastSeen: time.Now().UTC(),
	})
	hubA := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: chat.NewStore(), Peers: storeA,
		Dialer: net.DialTimeout, Timeout: time.Second,
	})

	res, err := hubA.AnnounceFile(chat.FileAnnounce{
		Name: "notes.txt",
		Size: 11,
		Mime: "text/plain",
		Hash: "sha256:abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := res.Message
	if msg.Type != chat.TypeFileAnnounce || msg.FileID == "" || msg.FileName != "notes.txt" {
		t.Fatalf("bad announce %+v", msg)
	}
	if msg.PeerID != "peer-a" || msg.Size != 11 || msg.Hash != "sha256:abc" {
		t.Fatalf("bad fields %+v", msg)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got := msgsB.List()
		if len(got) == 1 && got[0].Type == chat.TypeFileAnnounce &&
			got[0].FileID == msg.FileID && got[0].FileName == "notes.txt" &&
			got[0].Size == 11 && got[0].Mime == "text/plain" && got[0].Hash == "sha256:abc" &&
			got[0].PeerID == "peer-a" {
			raw, _ := json.Marshal(got[0])
			if strings.Contains(string(raw), "hello file body") {
				t.Fatal("must not auto-transfer file body")
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("peer-b missing file announce; got %+v", msgsB.List())
}
