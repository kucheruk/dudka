package loopback_test

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/files"
	"dudka/internal/loopback"
)

func TestTransfersEndpointReportsPercent(t *testing.T) {
	t.Parallel()

	dirA := t.TempDir()
	dirB := t.TempDir()
	blobsA, err := files.NewStore(filepath.Join(dirA, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, 48)
	for i := range payload {
		payload[i] = 'x'
	}

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
		ProgressYield: 5 * time.Millisecond,
	})
	res, err := hubA.AnnounceFile(chat.FileAnnounce{
		Name: "p.bin", Mime: "application/octet-stream", Hash: "sha256:p",
		Content: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = storeB.Append(res.Message)

	api := loopback.New("peer-b", "Bob")
	api.SetChat(hubB)
	ln, err := api.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() { _ = api.Serve(ln) }()
	base := "http://" + ln.Addr().String()

	body, _ := json.Marshal(map[string]any{"file_id": res.Message.FileID, "wait": false})
	resp, err := http.Post(base+"/files/fetch", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	seenMid := false
	seenHundred := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		tr, err := http.Get(base + "/files/transfers")
		if err != nil {
			t.Fatal(err)
		}
		var env struct {
			Transfers []chat.Transfer `json:"transfers"`
		}
		_ = json.NewDecoder(tr.Body).Decode(&env)
		tr.Body.Close()
		for _, x := range env.Transfers {
			if x.FileID != res.Message.FileID {
				continue
			}
			if x.Percent > 0 && x.Percent < 100 {
				seenMid = true
			}
			if x.Percent == 100 && x.Status == chat.TransferDone {
				seenHundred = true
			}
		}
		if seenHundred {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if !seenHundred {
		t.Fatal("never reached 100%")
	}
	if !seenMid {
		t.Fatal("never saw mid progress via GET /files/transfers")
	}
}
