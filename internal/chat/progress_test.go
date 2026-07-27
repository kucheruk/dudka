package chat_test

import (
	"net"
	"path/filepath"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/files"
)

func TestFetchExposesTransferProgressToHundred(t *testing.T) {
	t.Parallel()

	dirA := t.TempDir()
	dirB := t.TempDir()
	blobsA, err := files.NewStore(filepath.Join(dirA, "blobs"))
	if err != nil {
		t.Fatal(err)
	}

	// Large enough + tiny chunks so polling can observe mid progress.
	payload := make([]byte, 64)
	for i := range payload {
		payload[i] = byte('a' + i%26)
	}

	storeA := discovery.NewPeerStore()
	hubA := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: chat.NewStore(), Peers: storeA,
		Blobs: blobsA, InboxDir: filepath.Join(dirA, "inbox"), ChunkSize: 8,
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
	msgsB := chat.NewStore()
	hubB := chat.NewHub(chat.Config{
		PeerID: "peer-b", Name: "Bob", Store: msgsB, Peers: storeB,
		InboxDir: filepath.Join(dirB, "inbox"), ChunkSize: 8,
		Dialer: net.DialTimeout, Timeout: 3 * time.Second,
		ProgressYield: 2 * time.Millisecond,
	})

	res, err := hubA.AnnounceFile(chat.FileAnnounce{
		Name: "big.bin", Mime: "application/octet-stream", Hash: files.SHA256Sum(payload),
		Content: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = msgsB.Append(res.Message)

	seen := map[int]bool{}
	done := make(chan error, 1)
	go func() {
		_, err := hubB.Fetch(res.Message.FileID)
		done <- err
	}()

	deadline := time.Now().Add(5 * time.Second)
	var fetchErr error
loop:
	for time.Now().Before(deadline) {
		for _, tr := range hubB.Transfers() {
			if tr.FileID == res.Message.FileID {
				seen[tr.Percent] = true
			}
		}
		select {
		case fetchErr = <-done:
			break loop
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if fetchErr != nil {
		t.Fatal(fetchErr)
	}
	// Drain final transfers.
	for _, tr := range hubB.Transfers() {
		if tr.FileID == res.Message.FileID {
			seen[tr.Percent] = true
			if tr.Status != chat.TransferDone || tr.Percent != 100 {
				t.Fatalf("final transfer=%+v", tr)
			}
		}
	}
	if !seen[100] {
		t.Fatalf("never saw 100%%; seen=%v", seen)
	}
	sawMid := false
	for p := range seen {
		if p > 0 && p < 100 {
			sawMid = true
			break
		}
	}
	if !sawMid {
		t.Fatalf("want mid progress in (0,100); seen=%v", seen)
	}
}
