package files_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"dudka/internal/files"
)

func TestMaterializeLocalPreservesDestinationNameAndBytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, "blobs", "uuid-without-extension")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte("dudka"), 20_000)
	if err := os.WriteFile(source, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	dest, err := files.InboxPath(
		filepath.Join(dir, "inbox"),
		"file-id",
		"family-photo.gif",
	)
	if err != nil {
		t.Fatal(err)
	}

	written, err := files.MaterializeLocal(context.Background(), source, dest, nil)
	if err != nil {
		t.Fatal(err)
	}
	if written != int64(len(payload)) {
		t.Fatalf("written=%d want=%d", written, len(payload))
	}
	if filepath.Base(dest) != "family-photo.gif" {
		t.Fatalf("dest=%q lost original basename", dest)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("materialized bytes differ")
	}
}

func TestMaterializeLocalCancellationRemovesPartial(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.WriteFile(source, bytes.Repeat([]byte("x"), 128*1024), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "inbox", "payload.bin")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := files.MaterializeLocal(ctx, source, dest, nil); err != files.ErrCancelled {
		t.Fatalf("err=%v want ErrCancelled", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("destination must not exist after cancellation: %v", err)
	}
	if _, err := os.Stat(dest + ".partial"); !os.IsNotExist(err) {
		t.Fatalf("partial must not exist after cancellation: %v", err)
	}
}
