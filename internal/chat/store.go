package chat

import (
	"sync"
)

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
	return true
}

// List returns messages in append order.
func (s *Store) List() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Message, len(s.msgs))
	copy(out, s.msgs)
	return out
}
