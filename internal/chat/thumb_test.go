package chat_test

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/files"
)

func sampleJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, color.RGBA{R: 20, G: 180, B: 60, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestAnnounceImageWritesThumbAndWiresB64(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	blobs, err := files.NewStore(filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	payload := sampleJPEG(t)
	hub := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: chat.NewStore(), Peers: discovery.NewPeerStore(),
		Blobs: blobs, ThumbsDir: filepath.Join(dir, "thumbs"),
	})
	res, err := hub.AnnounceFile(chat.FileAnnounce{
		Name: "pic.jpg", Mime: "image/jpeg", Hash: files.SHA256Sum(payload), Content: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := res.Message
	if msg.ThumbB64 == "" {
		t.Fatal("want thumb_b64 on announce")
	}
	if msg.ThumbPath == "" {
		t.Fatal("want local thumb_path")
	}
	if _, err := os.Stat(msg.ThumbPath); err != nil {
		t.Fatalf("thumb file missing: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(msg.ThumbB64)
	if err != nil || len(raw) == 0 {
		t.Fatalf("thumb_b64 decode: %v len=%d", err, len(raw))
	}
	wire, err := chat.EncodeMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wire), `"thumb_b64"`) {
		t.Fatalf("wire missing thumb_b64: %s", wire)
	}
	if strings.Contains(string(wire), `"thumb_path"`) {
		t.Fatalf("wire must not leak local thumb_path: %s", wire)
	}
}

func TestPeerReceivesThumbWithoutFullDownload(t *testing.T) {
	t.Parallel()
	dirA := t.TempDir()
	dirB := t.TempDir()
	blobsA, err := files.NewStore(filepath.Join(dirA, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	payload := sampleJPEG(t)

	storeA := discovery.NewPeerStore()
	hubA := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: chat.NewStore(), Peers: storeA,
		Blobs: blobsA, ThumbsDir: filepath.Join(dirA, "thumbs"),
		Dialer: net.DialTimeout, Timeout: 3 * time.Second,
	})
	nodeA := discovery.NewNode(discovery.Config{
		PeerID: "peer-a", DisplayName: "Alice", InstanceID: "a1",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: time.Hour,
		Peers: storeA, DialTimeout: 3 * time.Second,
	})
	if err := nodeA.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeA.Stop() })

	storeB := discovery.NewPeerStore()
	msgsB := chat.NewStore()
	hubB := chat.NewHub(chat.Config{
		PeerID: "peer-b", Name: "Bob", Store: msgsB, Peers: storeB,
		ThumbsDir: filepath.Join(dirB, "thumbs"),
		Dialer:    net.DialTimeout, Timeout: 3 * time.Second,
	})
	nodeB := discovery.NewNode(discovery.Config{
		PeerID: "peer-b", DisplayName: "Bob", InstanceID: "b1",
		Bind: "127.0.0.1:0", TCPBind: "127.0.0.1:0", Interval: time.Hour,
		Peers: storeB, OnChatLine: hubB.HandleChatLine, DialTimeout: 3 * time.Second,
	})
	if err := nodeB.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = nodeB.Stop() })

	_ = storeA.Upsert(discovery.Peer{
		PeerID: "peer-b", Host: "127.0.0.1", TCPPort: nodeB.TCPPort(),
		LastSeen: time.Now().UTC(),
	})

	res, err := hubA.AnnounceFile(chat.FileAnnounce{
		Name: "pic.jpg", Mime: "image/jpeg", Hash: files.SHA256Sum(payload), Content: payload,
	})
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	var got chat.Message
	for time.Now().Before(deadline) {
		for _, m := range msgsB.List() {
			if m.FileID == res.Message.FileID {
				got = m
				break
			}
		}
		if got.FileID != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got.FileID == "" {
		t.Fatal("bob missing announce")
	}
	if got.ThumbB64 == "" {
		t.Fatal("bob missing thumb_b64")
	}
	if got.ThumbPath == "" {
		t.Fatal("bob missing materialized thumb_path")
	}
	if _, err := os.Stat(got.ThumbPath); err != nil {
		t.Fatalf("bob thumb file: %v", err)
	}
	// Full blob must not appear just from announce.
	if _, err := os.Stat(filepath.Join(dirB, "blobs", got.FileID)); err == nil {
		t.Fatal("bob must not auto-download full blob")
	}
}

func TestAnnounceNonImageHasNoThumb(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	blobs, err := files.NewStore(filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("plain-text-bytes")
	hub := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: chat.NewStore(), Peers: discovery.NewPeerStore(),
		Blobs: blobs, ThumbsDir: filepath.Join(dir, "thumbs"),
	})
	res, err := hub.AnnounceFile(chat.FileAnnounce{
		Name: "notes.txt", Mime: "text/plain", Hash: files.SHA256Sum(payload), Content: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Message.ThumbB64 != "" || res.Message.ThumbPath != "" {
		t.Fatalf("non-image must not get thumb: %+v", res.Message)
	}
}
