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

func TestFetchHashMismatchNotSuccessful(t *testing.T) {
	t.Parallel()

	dirA := t.TempDir()
	dirB := t.TempDir()
	blobsA, err := files.NewStore(filepath.Join(dirA, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("real-payload-bytes")
	wrongHash := files.SHA256Sum([]byte("tampered-expected"))

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
	inboxB := filepath.Join(dirB, "inbox")
	msgsB := chat.NewStore()
	hubB := chat.NewHub(chat.Config{
		PeerID: "peer-b", Name: "Bob", Store: msgsB, Peers: storeB,
		InboxDir: inboxB, ChunkSize: 8,
		Dialer: net.DialTimeout, Timeout: 3 * time.Second,
	})

	// Put blob with real bytes but announce a wrong hash (simulates bit-rot / lie).
	res, err := hubA.AnnounceFile(chat.FileAnnounce{
		Name: "x.bin", Mime: "application/octet-stream", Hash: wrongHash,
		Content: payload,
	})
	if err != nil {
		// If announce validates content vs hash, seed manually.
		t.Logf("announce rejected (ok if hash enforced): %v", err)
		fileID, err2 := seedAnnounceWithBlob(t, hubA, blobsA, msgsB, payload, wrongHash)
		if err2 != nil {
			t.Fatal(err2)
		}
		assertFetchCorrupt(t, hubB, fileID, inboxB, "x.bin")
		return
	}
	_ = msgsB.Append(res.Message)
	assertFetchCorrupt(t, hubB, res.Message.FileID, inboxB, "x.bin")
}

func TestFetchHashMatchSucceeds(t *testing.T) {
	t.Parallel()

	dirA := t.TempDir()
	dirB := t.TempDir()
	blobsA, err := files.NewStore(filepath.Join(dirA, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("good-payload-ok")
	sum := files.SHA256Sum(payload)

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
	})

	res, err := hubA.AnnounceFile(chat.FileAnnounce{
		Name: "ok.bin", Mime: "application/octet-stream", Hash: sum, Content: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = msgsB.Append(res.Message)

	out, err := hubB.Fetch(res.Message.FileID)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != chat.TransferDone || out.Path == "" {
		t.Fatalf("want done success: %+v", out)
	}
	got, err := os.ReadFile(out.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("bytes=%q", got)
	}
}

func seedAnnounceWithBlob(t *testing.T, hubA *chat.Hub, blobs *files.Store, msgsB *chat.Store, payload []byte, hash string) (string, error) {
	t.Helper()
	res, err := hubA.AnnounceFile(chat.FileAnnounce{
		Name: "x.bin", Mime: "application/octet-stream", Hash: files.SHA256Sum(payload), Content: payload,
	})
	if err != nil {
		return "", err
	}
	// Overwrite message hash in Bob's view to wrong value while Alice keeps real blob.
	msg := res.Message
	msg.Hash = hash
	_ = msgsB.Append(msg)
	return msg.FileID, nil
}

func assertFetchCorrupt(t *testing.T, hubB *chat.Hub, fileID, inbox, name string) {
	t.Helper()
	_, err := hubB.Fetch(fileID)
	if err == nil {
		t.Fatal("want hash mismatch error")
	}
	if !files.IsCorrupt(err) {
		t.Fatalf("err=%v want corrupt", err)
	}
	tr, ok := hubB.Transfer(fileID)
	if !ok || tr.Status == chat.TransferDone || tr.Path != "" {
		t.Fatalf("must not be successful transfer: ok=%v %+v", ok, tr)
	}
	if tr.Status != chat.TransferError {
		t.Fatalf("status=%q want error", tr.Status)
	}
	dest, err := files.InboxPath(inbox, fileID, name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("corrupt file must not remain as success path: %s", dest)
	}
}
