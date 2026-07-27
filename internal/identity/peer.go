// Package identity owns local peer identity that survives process restarts.
package identity

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PeerIDFile is the basename of the on-disk peer_id record under the data dir.
const PeerIDFile = "peer_id"

var uuidV4Re = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// LoadOrCreate returns the stable peer_id for dataDir, creating and persisting
// a new UUID v4 on first start (DUD-CHAT-110 / ROADMAP P010).
func LoadOrCreate(dataDir string) (string, error) {
	if strings.TrimSpace(dataDir) == "" {
		return "", errors.New("identity: data dir is empty")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", fmt.Errorf("identity: mkdir data dir: %w", err)
	}

	path := filepath.Join(dataDir, PeerIDFile)
	raw, err := os.ReadFile(path)
	if err == nil {
		id := strings.TrimSpace(string(raw))
		if uuidV4Re.MatchString(id) {
			return strings.ToLower(id), nil
		}
		// Corrupt/legacy file: replace with a fresh id.
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("identity: read peer_id: %w", err)
	}

	id, err := NewUUIDv4()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("identity: write peer_id: %w", err)
	}
	return id, nil
}

// NewUUIDv4 returns a random RFC 4122 version-4 UUID string.
func NewUUIDv4() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("identity: generate uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122
	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:],
	), nil
}
