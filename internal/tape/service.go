package tape

import (
	"context"
	"time"
)

type Store interface {
	NextSeq(context.Context, string) (uint64, error)
	PutEntry(context.Context, string, *Entry) error
	NextAnchorID(context.Context, string) (string, error)
	PutAnchor(context.Context, string, *Anchor) error
	GetLatestAnchor(context.Context, string) (*Anchor, error)
	GetTape(context.Context, string) (*Tape, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Append(ctx context.Context, sessionID string, in AppendInput) (*Entry, error) {
	return s.append(ctx, sessionID, in, nil)
}

func (s *Service) append(ctx context.Context, sessionID string, in AppendInput, correctsSeq *uint64) (*Entry, error) {
	seq, err := s.store.NextSeq(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		Seq:       seq,
		Kind:      in.Kind,
		Content:   in.Content,
		Metadata:  in.Metadata,
		CorrectsSeq: correctsSeq,
		CreatedAt: time.Now().UTC(),
		Actor:     in.Actor,
	}
	if err := s.store.PutEntry(ctx, sessionID, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *Service) AppendCorrection(ctx context.Context, sessionID string, correctsSeq uint64, in AppendInput) (*Entry, error) {
	return s.append(ctx, sessionID, in, &correctsSeq)
}

func (s *Service) CreateAnchor(ctx context.Context, sessionID string, in CreateAnchorInput) (*Anchor, error) {
	id, err := s.store.NextAnchorID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	var prevID string
	if prev, err := s.store.GetLatestAnchor(ctx, sessionID); err == nil && prev != nil {
		prevID = prev.ID
	}

	atSeq := in.AtSeq
	if atSeq == 0 {
		tp, err := s.store.GetTape(ctx, sessionID)
		if err == nil && tp != nil {
			atSeq = tp.HeadSeq
		}
	}

	anchor := &Anchor{
		ID:           id,
		SessionID:    sessionID,
		AtSeq:        atSeq,
		PrevAnchorID: prevID,
		PhaseTag:     in.PhaseTag,
		Summary:      in.Summary,
		State:        in.State,
		SourceSeqs:   in.SourceSeqs,
		CreatedAt:    time.Now().UTC(),
		Owner:        in.Owner,
	}
	if err := s.store.PutAnchor(ctx, sessionID, anchor); err != nil {
		return nil, err
	}
	return anchor, nil
}
