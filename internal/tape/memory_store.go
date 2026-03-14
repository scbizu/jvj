package tape

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type InMemoryStore struct {
	mu           sync.RWMutex
	tapes        map[string]*Tape
	entries      map[string][]Entry
	anchors      map[string][]Anchor
	anchorCounts map[string]uint64
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		tapes:        map[string]*Tape{},
		entries:      map[string][]Entry{},
		anchors:      map[string][]Anchor{},
		anchorCounts: map[string]uint64{},
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

func (s *InMemoryStore) NextAnchorID(_ context.Context, sessionID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.anchorCounts[sessionID]++
	return fmt.Sprintf("%s-anchor-%d", sessionID, s.anchorCounts[sessionID]), nil
}

func (s *InMemoryStore) PutAnchor(_ context.Context, sessionID string, anchor *Anchor) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.anchors[sessionID] = append(s.anchors[sessionID], *anchor)
	return nil
}

func (s *InMemoryStore) GetLatestAnchor(_ context.Context, sessionID string) (*Anchor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	anchors := s.anchors[sessionID]
	if len(anchors) == 0 {
		return nil, errors.New("anchor not found")
	}
	anchor := anchors[len(anchors)-1]
	return &anchor, nil
}

func (s *InMemoryStore) GetTape(_ context.Context, sessionID string) (*Tape, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tp, ok := s.tapes[sessionID]
	if !ok {
		return nil, errors.New("tape not found")
	}
	copyTape := *tp
	return &copyTape, nil
}

func (s *InMemoryStore) SeqsFrom(_ context.Context, sessionID string, fromSeq uint64) []uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := s.entries[sessionID]
	seqs := make([]uint64, 0, len(entries))
	for _, entry := range entries {
		if entry.Seq >= fromSeq {
			seqs = append(seqs, entry.Seq)
		}
	}
	return seqs
}
