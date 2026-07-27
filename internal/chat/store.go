package chat

import (
	"sync"
)

// MaxTailMessages is the keeper ring size (DUD-CHAT-120 / P033).
const MaxTailMessages = 200

// Store is an in-memory message log for the local engine (P030).
type Store struct {
	mu   sync.RWMutex
	byID map[string]struct{}
	msgs []Message
}

// NewStore returns an empty message store.
func NewStore() *Store {
	return &Store{byID: make(map[string]struct{})}
}

// Append inserts msg if msg_id is new. Returns true when inserted.
// Keeps at most MaxTailMessages (drops oldest).
func (s *Store) Append(msg Message) bool {
	if msg.MsgID == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[msg.MsgID]; ok {
		return false
	}
	s.byID[msg.MsgID] = struct{}{}
	s.msgs = append(s.msgs, msg)
	for len(s.msgs) > MaxTailMessages {
		old := s.msgs[0]
		s.msgs = s.msgs[1:]
		delete(s.byID, old.MsgID)
	}
	return true
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
