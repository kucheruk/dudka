package files

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrCorrupt is returned when downloaded bytes do not match the announce hash (DUD-FILE-130).
var ErrCorrupt = errors.New("файл повреждён")

// SHA256Sum returns content hash as "sha256:<hex>".
func SHA256Sum(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// NormalizeHash accepts "sha256:<hex>" or bare hex; returns lowercase hex without prefix.
func NormalizeHash(want string) (string, error) {
	s := strings.TrimSpace(want)
	if s == "" {
		return "", fmt.Errorf("files: empty hash")
	}
	if strings.HasPrefix(strings.ToLower(s), "sha256:") {
		s = s[len("sha256:"):]
	}
	s = strings.TrimSpace(s)
	if len(s) != hex.EncodedLen(sha256.Size) {
		return "", fmt.Errorf("files: bad hash length")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("files: bad hash hex: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// VerifyFile checks that path bytes match want hash (sha256:… or bare hex).
func VerifyFile(path, want string) error {
	wantHex, err := NormalizeHash(want)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != wantHex {
		return ErrCorrupt
	}
	return nil
}

// IsCorrupt reports whether err is (or wraps) a content-hash mismatch.
func IsCorrupt(err error) bool {
	return errors.Is(err, ErrCorrupt)
}
