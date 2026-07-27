package identity_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"dudka/internal/identity"
)

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestLoadOrCreatePersistsAcrossRestarts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	first, err := identity.LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("first LoadOrCreate: %v", err)
	}
	if !uuidRe.MatchString(first) {
		t.Fatalf("peer_id %q is not UUID v4", first)
	}

	path := filepath.Join(dir, identity.PeerIDFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("peer_id file missing: %v", err)
	}
	if got := string(raw); got != first+"\n" && got != first {
		t.Fatalf("on-disk peer_id = %q, want %q", got, first)
	}

	second, err := identity.LoadOrCreate(dir)
	if err != nil {
		t.Fatalf("second LoadOrCreate: %v", err)
	}
	if second != first {
		t.Fatalf("restart peer_id = %q, want same as first %q", second, first)
	}
}

func TestLoadOrCreateDifferentDirsGetDifferentIDs(t *testing.T) {
	t.Parallel()
	a, err := identity.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b, err := identity.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("independent dirs unexpectedly share peer_id %q", a)
	}
}
