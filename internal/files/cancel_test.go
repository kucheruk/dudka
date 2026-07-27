package files_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dudka/internal/files"
)

func TestReadChunksCancelDiscardsPartial(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store, err := files.NewStore(filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte("abcdefgh"), 32) // 256 bytes
	if err := store.Put("fid", payload); err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	if err := files.ServeChunks(&wire, store, files.ChunkReq{FileID: "fid", Limit: 16}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	out := filepath.Join(dir, "out.bin")
	var sawProgress bool
	done := make(chan error, 1)
	go func() {
		_, err := files.ReadChunks(ctx, bytes.NewReader(wire.Bytes()), "fid", out, int64(len(payload)), func(recv, total int64) {
			if recv > 0 && recv < total {
				sawProgress = true
				cancel()
			}
		})
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, files.ErrCancelled) {
			t.Fatalf("err=%v want ErrCancelled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
	if !sawProgress {
		t.Fatal("expected mid progress before cancel")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("final file must not exist after cancel; err=%v", err)
	}
	if _, err := os.Stat(out + ".partial"); !os.IsNotExist(err) {
		t.Fatalf("partial must be discarded; err=%v", err)
	}
}
