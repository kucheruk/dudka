package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// MaxTailMessages is the keeper ring size (DUD-CHAT-120 / P033).
const MaxTailMessages = 200

// Store is the bounded local message log. A persistent store keeps the same
// tail when the engine or desktop application is replaced.
type Store struct {
	mu          sync.RWMutex
	byID        map[string]struct{}
	msgs        []Message
	historyPath string
}

// NewStore returns an empty message store.
func NewStore() *Store {
	return &Store{byID: make(map[string]struct{})}
}

// NewPersistentStore loads a bounded message tail from historyPath and writes
// each later append back atomically.
func NewPersistentStore(historyPath string) (*Store, error) {
	if historyPath == "" {
		return nil, errors.New("chat: history path is empty")
	}
	s := &Store{
		byID:        make(map[string]struct{}),
		historyPath: historyPath,
	}
	raw, err := os.ReadFile(historyPath)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("chat: read history: %w", err)
	}
	var messages []Message
	if err := json.Unmarshal(raw, &messages); err != nil {
		return nil, fmt.Errorf("chat: decode history: %w", err)
	}
	if len(messages) > MaxTailMessages {
		messages = messages[len(messages)-MaxTailMessages:]
	}
	for _, msg := range messages {
		if msg.MsgID == "" {
			continue
		}
		if _, exists := s.byID[msg.MsgID]; exists {
			continue
		}
		s.byID[msg.MsgID] = struct{}{}
		s.msgs = append(s.msgs, msg)
	}
	return s, nil
}

// Append inserts msg if msg_id is new. Returns true when inserted.
// Keeps at most MaxTailMessages (drops oldest).
func (s *Store) Append(msg Message) bool {
	inserted, _ := s.AppendPersistent(msg)
	return inserted
}

// AppendPersistent reports both insertion and a disk-write error.
func (s *Store) AppendPersistent(msg Message) (bool, error) {
	if msg.MsgID == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[msg.MsgID]; ok {
		return false, nil
	}
	next := append(append([]Message(nil), s.msgs...), msg)
	if len(next) > MaxTailMessages {
		next = next[len(next)-MaxTailMessages:]
	}
	if err := s.persistLocked(next); err != nil {
		return false, err
	}
	s.msgs = next
	s.byID = make(map[string]struct{}, len(next))
	for _, item := range next {
		s.byID[item.MsgID] = struct{}{}
	}
	return true, nil
}

func (s *Store) persistLocked(messages []Message) error {
	if s.historyPath == "" {
		return nil
	}
	raw, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("chat: encode history: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.historyPath), 0o700); err != nil {
		return fmt.Errorf("chat: mkdir history: %w", err)
	}
	tmp := s.historyPath + ".new"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("chat: write history: %w", err)
	}
	if err := os.Rename(tmp, s.historyPath); err != nil {
		// Windows does not replace an existing destination with Rename.
		if removeErr := os.Remove(s.historyPath); removeErr != nil &&
			!errors.Is(removeErr, os.ErrNotExist) {
			_ = os.Remove(tmp)
			return fmt.Errorf("chat: replace history: %w", err)
		}
		if retryErr := os.Rename(tmp, s.historyPath); retryErr != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("chat: replace history: %w", retryErr)
		}
	}
	return nil
}

// List returns messages in append order (≤ MaxTailMessages).
func (s *Store) List() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Message, len(s.msgs))
	copy(out, s.msgs)
	return out
}

// FindFile returns the file_announce message for fileID, if present.
func (s *Store) FindFile(fileID string) (Message, bool) {
	if fileID == "" {
		return Message{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := len(s.msgs) - 1; i >= 0; i-- {
		m := s.msgs[i]
		if m.Type == TypeFileAnnounce && m.FileID == fileID {
			return m, true
		}
	}
	return Message{}, false
}
