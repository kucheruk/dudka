// Package files owns local blob storage and chunked LAN transfer (DUD-FILE-110 / P051).
package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store keeps complete source blobs keyed by file_id under a directory.
type Store struct {
	dir string
}

// NewStore creates (or opens) a blob directory.
func NewStore(dir string) (*Store, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("files: empty store dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// Dir returns the blob root.
func (s *Store) Dir() string { return s.dir }

func (s *Store) pathFor(fileID string) (string, error) {
	id := strings.TrimSpace(fileID)
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "\\") || id == "." || id == ".." {
		return "", fmt.Errorf("files: bad file_id")
	}
	return filepath.Join(s.dir, id), nil
}

// Put writes complete blob bytes for file_id.
func (s *Store) Put(fileID string, data []byte) error {
	if s == nil {
		return fmt.Errorf("files: nil store")
	}
	p, err := s.pathFor(fileID)
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Has reports whether a complete blob exists.
func (s *Store) Has(fileID string) bool {
	if s == nil {
		return false
	}
	p, err := s.pathFor(fileID)
	if err != nil {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

// Open opens a blob for reading.
func (s *Store) Open(fileID string) (*os.File, error) {
	if s == nil {
		return nil, fmt.Errorf("files: nil store")
	}
	p, err := s.pathFor(fileID)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// Path returns the absolute blob path if present.
func (s *Store) Path(fileID string) (string, error) {
	p, err := s.pathFor(fileID)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}
