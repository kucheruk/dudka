package files_test

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"testing"

	"dudka/internal/files"
)

func TestEncodeDecodeChunkReq(t *testing.T) {
	t.Parallel()
	raw, err := files.EncodeChunkReq(files.ChunkReq{
		FileID: "fid",
		Offset: 10,
		Limit:  4,
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := files.DecodeChunkReq(raw)
	if err != nil {
		t.Fatal(err)
	}
	if req.FileID != "fid" || req.Offset != 10 || req.Limit != 4 {
		t.Fatalf("%+v", req)
	}
}

func TestServeAndFetchChunksReassemble(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store, err := files.NewStore(filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("abcdefghij") // 10 bytes; chunk limit 4 → multiple chunks
	if err := store.Put("fid-1", payload); err != nil {
		t.Fatal(err)
	}

	var wire bytes.Buffer
	if err := files.ServeChunks(&wire, store, files.ChunkReq{
		FileID: "fid-1",
		Offset: 0,
		Limit:  4,
	}); err != nil {
		t.Fatal(err)
	}
	// Must be more than one chunk line for a 10-byte file with limit 4.
	lines := bytes.Count(wire.Bytes(), []byte("\n"))
	if lines < 3 {
		t.Fatalf("want ≥3 chunk lines for chunked transfer, got %d wire=%q", lines, wire.Bytes())
	}

	outPath := filepath.Join(dir, "out.bin")
	got, err := files.ReadChunks(bytes.NewReader(wire.Bytes()), "fid-1", outPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != int64(len(payload)) {
		t.Fatalf("size=%d", got)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, payload) {
		t.Fatalf("got %q want %q", raw, payload)
	}
	// Chunks carry base64, not raw body as a single HTTP blob.
	if !bytes.Contains(wire.Bytes(), []byte(base64.StdEncoding.EncodeToString([]byte("abcd")))) {
		t.Fatal("expected base64 chunk payload on wire")
	}
	_ = io.EOF
}
