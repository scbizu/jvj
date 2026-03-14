package tape

import (
	"context"
	"time"
)

type Store interface {
	NextSeq(context.Context, string) (uint64, error)
	PutEntry(context.Context, string, *Entry) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Append(ctx context.Context, sessionID string, in AppendInput) (*Entry, error) {
	seq, err := s.store.NextSeq(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	entry := &Entry{
		Seq:       seq,
		Kind:      in.Kind,
		Content:   in.Content,
		Metadata:  in.Metadata,
		CreatedAt: time.Now().UTC(),
		Actor:     in.Actor,
	}
	if err := s.store.PutEntry(ctx, sessionID, entry); err != nil {
		return nil, err
	}
	return entry, nil
}
