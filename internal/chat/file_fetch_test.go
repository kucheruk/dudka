package chat_test

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/files"
)

func TestFetchFileFromSourceWritesFullDiskCopy(t *testing.T) {
	t.Parallel()

	dirA := t.TempDir()
	dirB := t.TempDir()
	blobsA, err := files.NewStore(filepath.Join(dirA, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	inboxB := filepath.Join(dirB, "inbox")

	payload := []byte("hello-from-alice-chunked-file!!") // > 8 bytes
	storeB := discovery.NewPeerStore()
	msgsB := chat.NewStore()
	hubB := chat.NewHub(chat.Config{
		PeerID: "peer-b", Name: "Bob", Store: msgsB, Peers: storeB,
		Dialer: net.DialTimeout, Timeout: 3 * time.Second,
		Blobs: nil, InboxDir: inboxB, ChunkSize: 8,
	})

	nodeB := discovery.NewNode(discovery.Config{
		PeerID: "peer-b", DisplayName: "Bob", InstanceID: "inst-b",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: time.Hour,
		Peers: storeB, OnChatLine: hubB.HandleChatLine,
		DialTimeout: 3 * time.Second,
	})
	// Source node with blobs + chunk handler.
	storeA := discovery.NewPeerStore()
	msgsA := chat.NewStore()
	hubA := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: msgsA, Peers: storeA,
		Dialer: net.DialTimeout, Timeout: 3 * time.Second,
		Blobs: blobsA, InboxDir: filepath.Join(dirA, "inbox"), ChunkSize: 8,
	})
	nodeA := discovery.NewNode(discovery.Config{
		PeerID: "peer-a", DisplayName: "Alice", InstanceID: "inst-a",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: time.Hour,
		Peers: storeA, OnChatLine: hubA.HandleChatLine,
		OnFileChunkRequest: hubA.HandleFileChunkRequest,
		DialTimeout:        3 * time.Second,
	})
	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeA.Stop() })
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeB.Stop() })

	_ = storeB.Upsert(discovery.Peer{
		PeerID: "peer-a", DisplayName: "Alice", InstanceID: "inst-a",
		Host: "127.0.0.1", TCPPort: nodeA.TCPPort(), LastSeen: time.Now().UTC(),
	})
	_ = storeA.Upsert(discovery.Peer{
		PeerID: "peer-b", DisplayName: "Bob", InstanceID: "inst-b",
		Host: "127.0.0.1", TCPPort: nodeB.TCPPort(), LastSeen: time.Now().UTC(),
	})

	res, err := hubA.AnnounceFile(chat.FileAnnounce{
		Name:    "hello.txt",
		Mime:    "text/plain",
		Hash:    "sha256:test",
		Content: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	fileID := res.Message.FileID

	// Wait for announce on Bob.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := msgsB.FindFile(fileID); ok {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, ok := msgsB.FindFile(fileID); !ok {
		t.Fatal("bob missing announce")
	}

	path, err := hubB.FetchFile(fileID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("disk=%q want=%q path=%s", got, payload, path)
	}
}
