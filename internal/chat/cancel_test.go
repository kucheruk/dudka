package chat_test

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/files"
)

func TestCancelFetchDiscardsPartialNotDone(t *testing.T) {
	t.Parallel()

	dirA := t.TempDir()
	dirB := t.TempDir()
	blobsA, err := files.NewStore(filepath.Join(dirA, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, 256)
	for i := range payload {
		payload[i] = byte(i)
	}

	storeA := discovery.NewPeerStore()
	hubA := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: chat.NewStore(), Peers: storeA,
		Blobs: blobsA, InboxDir: filepath.Join(dirA, "inbox"), ChunkSize: 16,
		Dialer: net.DialTimeout, Timeout: 3 * time.Second,
	})
	nodeA := discovery.NewNode(discovery.Config{
		PeerID: "peer-a", DisplayName: "Alice", InstanceID: "a1",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: time.Hour,
		Peers: storeA, OnFileChunkRequest: hubA.HandleFileChunkRequest,
		DialTimeout: 3 * time.Second,
	})
	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeA.Stop() })

	storeB := discovery.NewPeerStore()
	_ = storeB.Upsert(discovery.Peer{
		PeerID: "peer-a", Host: "127.0.0.1", TCPPort: nodeA.TCPPort(),
		LastSeen: time.Now().UTC(),
	})
	inboxB := filepath.Join(dirB, "inbox")
	msgsB := chat.NewStore()
	hubB := chat.NewHub(chat.Config{
		PeerID: "peer-b", Name: "Bob", Store: msgsB, Peers: storeB,
		InboxDir: inboxB, ChunkSize: 16,
		Dialer: net.DialTimeout, Timeout: 3 * time.Second,
		ProgressYield: 10 * time.Millisecond,
	})

	res, err := hubA.AnnounceFile(chat.FileAnnounce{
		Name: "big.bin", Mime: "application/octet-stream", Hash: files.SHA256Sum(payload),
		Content: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = msgsB.Append(res.Message)
	fileID := res.Message.FileID

	if _, err := hubB.StartFetch(fileID); err != nil {
		t.Fatal(err)
	}

	// Wait until mid progress, then cancel.
	var cancelled chat.Transfer
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, tr := range hubB.Transfers() {
			if tr.FileID == fileID && tr.Percent >= 20 && tr.Percent < 100 && tr.Status == chat.TransferDownloading {
				cancelled, err = hubB.CancelFetch(fileID)
				if err != nil {
					t.Fatal(err)
				}
				goto cancelled
			}
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("never reached mid progress to cancel")
cancelled:
	if cancelled.Status != chat.TransferCancelled {
		t.Fatalf("cancel status=%q want cancelled: %+v", cancelled.Status, cancelled)
	}
	if cancelled.Status == chat.TransferDone || cancelled.Path != "" {
		t.Fatalf("must not look successful: %+v", cancelled)
	}

	// Allow fetch goroutine to settle.
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tr, ok := hubB.Transfer(fileID)
		if ok && tr.Status == chat.TransferCancelled {
			cancelled = tr
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if cancelled.Status != chat.TransferCancelled {
		t.Fatalf("final status=%+v", cancelled)
	}
	if cancelled.Path != "" {
		t.Fatalf("cancelled transfer must not expose success path: %+v", cancelled)
	}

	dest, err := files.InboxPath(inboxB, fileID, "big.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("inbox file must be discarded, still exists: %s", dest)
	}
	if _, err := os.Stat(dest + ".partial"); !os.IsNotExist(err) {
		t.Fatalf("partial must be discarded")
	}
}
