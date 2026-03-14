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
	SeqsFrom(context.Context, string, uint64) []uint64
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

func (s *Service) Handoff(ctx context.Context, sessionID string, in HandoffInput) (*Anchor, error) {
	entry, err := s.Append(ctx, sessionID, AppendInput{
		Kind:    EntryHandoff,
		Content: in.Summary,
		Actor:   in.Owner,
		Metadata: map[string]any{
			"next_steps":  in.NextSteps,
			"source_seqs": in.SourceSeqs,
			"phase_tag":   in.PhaseTag,
		},
	})
	if err != nil {
		return nil, err
	}

	return s.CreateAnchor(ctx, sessionID, CreateAnchorInput{
		PhaseTag:   in.PhaseTag,
		Summary:    in.Summary,
		SourceSeqs: in.SourceSeqs,
		State:      in.StateDelta,
		Owner:      in.Owner,
		AtSeq:      entry.Seq,
	})
}

func (s *Service) BuildView(ctx context.Context, req ViewRequest) (*View, error) {
	if anchor, err := s.store.GetLatestAnchor(ctx, req.SessionID); err == nil && anchor != nil {
		return &View{
			SessionID:    req.SessionID,
			AnchorID:     anchor.ID,
			IncludedSeqs: s.store.SeqsFrom(ctx, req.SessionID, anchor.AtSeq),
			Provenance:   anchor.SourceSeqs,
		}, nil
	}

	seqs := s.store.SeqsFrom(ctx, req.SessionID, 1)
	return &View{
		SessionID:    req.SessionID,
		IncludedSeqs: seqs,
		Provenance:   seqs,
	}, nil
}
