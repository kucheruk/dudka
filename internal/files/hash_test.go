package files_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"dudka/internal/files"
)

func TestSHA256FormatAndVerify(t *testing.T) {
	t.Parallel()
	payload := []byte("hello-hash")
	sum := files.SHA256Sum(payload)
	if !bytes.HasPrefix([]byte(sum), []byte("sha256:")) {
		t.Fatalf("sum=%q", sum)
	}
	path := filepath.Join(t.TempDir(), "f.bin")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := files.VerifyFile(path, sum); err != nil {
		t.Fatal(err)
	}
	// bare hex also accepted
	hexOnly := sum[len("sha256:"):]
	if err := files.VerifyFile(path, hexOnly); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyFileMismatchCorrupt(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bad.bin")
	if err := os.WriteFile(path, []byte("actual-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := files.SHA256Sum([]byte("other-bytes"))
	err := files.VerifyFile(path, want)
	if err == nil {
		t.Fatal("want corrupt error")
	}
	if !files.IsCorrupt(err) {
		t.Fatalf("err=%v want IsCorrupt", err)
	}
	if err.Error() == "" || !containsRuCorrupt(err.Error()) {
		t.Fatalf("want Russian corrupt message, got %q", err.Error())
	}
}

func containsRuCorrupt(s string) bool {
	return bytes.Contains([]byte(s), []byte("повреждён")) || bytes.Contains([]byte(s), []byte("поврежден"))
}
