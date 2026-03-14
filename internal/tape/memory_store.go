package tape

import (
	"context"
	"sync"
	"time"
)

type InMemoryStore struct {
	mu      sync.RWMutex
	tapes   map[string]*Tape
	entries map[string][]Entry
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		tapes:   map[string]*Tape{},
		entries: map[string][]Entry{},
	}
}

func (s *InMemoryStore) NextSeq(_ context.Context, sessionID string) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	tp, ok := s.tapes[sessionID]
	if !ok {
		tp = &Tape{
			SessionID: sessionID,
			CreatedAt: now,
		}
		s.tapes[sessionID] = tp
	}
	tp.HeadSeq++
	tp.UpdatedAt = now
	return tp.HeadSeq, nil
}

func (s *InMemoryStore) PutEntry(_ context.Context, sessionID string, entry *Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[sessionID] = append(s.entries[sessionID], *entry)
	return nil
}
