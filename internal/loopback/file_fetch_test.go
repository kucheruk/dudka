package loopback_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/files"
	"dudka/internal/loopback"
)

func TestPostFilesFetchDownloadsFromSource(t *testing.T) {
	t.Parallel()

	dirA := t.TempDir()
	dirB := t.TempDir()
	blobsA, err := files.NewStore(filepath.Join(dirA, "blobs"))
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("loopback-file-bytes-xxxx")
	peersA := discovery.NewPeerStore()
	hubA := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: chat.NewStore(), Peers: peersA,
		Blobs: blobsA, InboxDir: filepath.Join(dirA, "inbox"), ChunkSize: 8,
		Dialer: net.DialTimeout, Timeout: 3 * time.Second,
	})
	nodeA := discovery.NewNode(discovery.Config{
		PeerID: "peer-a", DisplayName: "Alice", InstanceID: "a1",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: time.Hour,
		Peers: peersA, OnFileChunkRequest: hubA.HandleFileChunkRequest,
		DialTimeout: 3 * time.Second,
	})
	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeA.Stop() })

	peersB := discovery.NewPeerStore()
	_ = peersB.Upsert(discovery.Peer{
		PeerID: "peer-a", Host: "127.0.0.1", TCPPort: nodeA.TCPPort(),
		LastSeen: time.Now().UTC(),
	})
	storeB := chat.NewStore()
	hubB := chat.NewHub(chat.Config{
		PeerID: "peer-b", Name: "Bob", Store: storeB, Peers: peersB,
		InboxDir: filepath.Join(dirB, "inbox"), ChunkSize: 8,
		Dialer: net.DialTimeout, Timeout: 3 * time.Second,
	})

	res, err := hubA.AnnounceFile(chat.FileAnnounce{
		Name: "x.bin", Mime: "application/octet-stream", Hash: "sha256:x",
		Content: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = storeB.Append(res.Message)

	api := loopback.New("peer-b", "Bob")
	api.SetChat(hubB)
	api.SetPeers(peersB)
	ln, err := api.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() { _ = api.Serve(ln) }()
	base := "http://" + ln.Addr().String()

	body, _ := json.Marshal(map[string]string{"file_id": res.Message.FileID})
	resp, err := http.Post(base+"/files/fetch", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, raw)
	}
	var out struct {
		Path string `json:"path"`
		Size int64  `json:"size"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("got %q want %q", got, payload)
	}
	if out.Size != int64(len(payload)) {
		t.Fatalf("size=%d", out.Size)
	}
}
