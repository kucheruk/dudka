package chat_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/files"
)

func TestAnnounceHEICThumbOrHonestFallback(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "sample.heic"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	blobs, err := files.NewStore(filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	hub := chat.NewHub(chat.Config{
		PeerID: "peer-a", Name: "Alice", Store: chat.NewStore(), Peers: discovery.NewPeerStore(),
		Blobs: blobs, ThumbsDir: filepath.Join(dir, "thumbs"),
	})
	res, err := hub.AnnounceFile(chat.FileAnnounce{
		Name: "img.heic", Mime: "image/heic", Hash: files.SHA256Sum(data), Content: data,
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := res.Message
	if files.HEICAvailable() {
		if msg.ThumbB64 == "" || msg.ThumbPath == "" {
			t.Fatalf("darwin HEIC should get thumb: %+v avail=%v goos=%s", msg, files.HEICAvailable(), runtime.GOOS)
		}
		if _, err := os.Stat(msg.ThumbPath); err != nil {
			t.Fatal(err)
		}
		return
	}
	if msg.ThumbB64 != "" || msg.ThumbPath != "" {
		t.Fatalf("without HEIC decode must not fake thumb: %+v", msg)
	}
}
