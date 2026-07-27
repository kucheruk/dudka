package files_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"dudka/internal/files"
)

func TestPercentClampsToHundred(t *testing.T) {
	t.Parallel()
	cases := []struct {
		recv, total int64
		want        int
	}{
		{0, 100, 0},
		{25, 100, 25},
		{100, 100, 100},
		{150, 100, 100},
		{0, 0, 0},
		{1, 0, 100},
	}
	for _, tc := range cases {
		if got := files.Percent(tc.recv, tc.total); got != tc.want {
			t.Fatalf("Percent(%d,%d)=%d want %d", tc.recv, tc.total, got, tc.want)
		}
	}
}

func TestReadChunksReportsProgressToHundred(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store, err := files.NewStore(filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("0123456789abcdef") // 16 bytes, limit 4 → 4 chunks
	if err := store.Put("fid", payload); err != nil {
		t.Fatal(err)
	}
	var wire bytes.Buffer
	if err := files.ServeChunks(&wire, store, files.ChunkReq{FileID: "fid", Limit: 4}); err != nil {
		t.Fatal(err)
	}

	var percents []int
	out := filepath.Join(dir, "out.bin")
	n, err := files.ReadChunks(bytes.NewReader(wire.Bytes()), "fid", out, int64(len(payload)), func(recv, total int64) {
		percents = append(percents, files.Percent(recv, total))
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(payload)) {
		t.Fatalf("n=%d", n)
	}
	if len(percents) < 2 {
		t.Fatalf("want multiple progress ticks, got %v", percents)
	}
	if percents[len(percents)-1] != 100 {
		t.Fatalf("last percent=%d want 100; all=%v", percents[len(percents)-1], percents)
	}
	sawMid := false
	for _, p := range percents {
		if p > 0 && p < 100 {
			sawMid = true
			break
		}
	}
	if !sawMid {
		t.Fatalf("want a mid-progress tick in (0,100), got %v", percents)
	}
}
