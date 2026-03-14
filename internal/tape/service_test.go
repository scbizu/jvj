package tape

import (
	"context"
	"errors"
	"testing"
)

func TestServiceAppendAssignsMonotonicSeq(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	first, err := svc.Append(context.Background(), "session-1", AppendInput{
		Kind:    EntryUser,
		Content: "hello",
		Actor:   "user",
	})
	if err != nil {
		t.Fatalf("append first entry: %v", err)
	}

	second, err := svc.Append(context.Background(), "session-1", AppendInput{
		Kind:    EntryAssistant,
		Content: "hi",
		Actor:   "agent",
	})
	if err != nil {
		t.Fatalf("append second entry: %v", err)
	}

	if first.Seq != 1 || second.Seq != 2 {
		t.Fatalf("expected seqs 1,2 got %d,%d", first.Seq, second.Seq)
	}
}

func TestServiceAppendCorrectionKeepsOriginalEntry(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	original, err := svc.Append(context.Background(), "session-1", AppendInput{
		Kind:    EntryAssistant,
		Content: "wrong",
		Actor:   "agent",
	})
	if err != nil {
		t.Fatalf("append original entry: %v", err)
	}

	correction, err := svc.AppendCorrection(context.Background(), "session-1", original.Seq, AppendInput{
		Kind:    EntryCorrection,
		Content: "fixed",
		Actor:   "agent",
	})
	if err != nil {
		t.Fatalf("append correction: %v", err)
	}

	if correction.CorrectsSeq == nil || *correction.CorrectsSeq != original.Seq {
		t.Fatalf("expected correction to point to %d", original.Seq)
	}
}

func TestServiceCreateAnchorBuildsLinearChain(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	first, err := svc.CreateAnchor(context.Background(), "session-1", CreateAnchorInput{
		PhaseTag:   "discover",
		Summary:    "summary",
		SourceSeqs: []uint64{1, 2},
		State:      map[string]any{"phase_tag": "discover"},
		Owner:      "agent",
	})
	if err != nil {
		t.Fatalf("create first anchor: %v", err)
	}

	second, err := svc.CreateAnchor(context.Background(), "session-1", CreateAnchorInput{
		PhaseTag:   "implement",
		Summary:    "next",
		SourceSeqs: []uint64{3},
		State:      map[string]any{"phase_tag": "implement"},
		Owner:      "agent",
	})
	if err != nil {
		t.Fatalf("create second anchor: %v", err)
	}

	if second.PrevAnchorID != first.ID {
		t.Fatalf("expected linear chain to previous anchor")
	}
}

func TestServiceHandoffWritesEntryAndAnchor(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	anchor, err := svc.Handoff(context.Background(), "session-1", HandoffInput{
		Summary:    "Discovery complete.",
		NextSteps:  []string{"Run migration"},
		SourceSeqs: []uint64{1, 2},
		Owner:      "agent",
		PhaseTag:   "implement",
	})
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}

	if anchor.PhaseTag != "implement" {
		t.Fatalf("expected implement anchor, got %q", anchor.PhaseTag)
	}

	entries := store.entries["session-1"]
	if len(entries) == 0 || entries[len(entries)-1].Kind != EntryHandoff {
		t.Fatal("expected handoff entry to be appended")
	}
}

func TestServiceHandoffKeepsEntryWhenAnchorWriteFails(t *testing.T) {
	store := &failingAnchorStore{InMemoryStore: NewInMemoryStore()}
	svc := NewService(store)

	_, err := svc.Handoff(context.Background(), "session-1", HandoffInput{
		Summary:    "Discovery complete.",
		NextSteps:  []string{"Run migration"},
		SourceSeqs: []uint64{1},
		Owner:      "agent",
		PhaseTag:   "implement",
	})
	if err == nil {
		t.Fatal("expected anchor write failure")
	}

	if got := len(store.entries["session-1"]); got != 1 {
		t.Fatalf("expected handoff entry to remain, got %d persisted entries", got)
	}
}

type failingAnchorStore struct {
	*InMemoryStore
}

func (s *failingAnchorStore) PutAnchor(context.Context, string, *Anchor) error {
	return errors.New("anchor write failed")
}

func TestBuildViewUsesLatestAnchor(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	if _, err := svc.Append(context.Background(), "session-1", AppendInput{
		Kind:    EntryUser,
		Content: "first",
		Actor:   "user",
	}); err != nil {
		t.Fatalf("append first entry: %v", err)
	}
	if _, err := svc.Append(context.Background(), "session-1", AppendInput{
		Kind:    EntryAssistant,
		Content: "second",
		Actor:   "agent",
	}); err != nil {
		t.Fatalf("append second entry: %v", err)
	}

	first, err := svc.CreateAnchor(context.Background(), "session-1", CreateAnchorInput{
		PhaseTag:   "discover",
		Summary:    "done",
		SourceSeqs: []uint64{1},
		State:      map[string]any{"phase_tag": "discover"},
		Owner:      "agent",
		AtSeq:      1,
	})
	if err != nil {
		t.Fatalf("create first anchor: %v", err)
	}
	latest, err := svc.CreateAnchor(context.Background(), "session-1", CreateAnchorInput{
		PhaseTag:   "implement",
		Summary:    "latest",
		SourceSeqs: []uint64{2},
		State:      map[string]any{"phase_tag": "implement"},
		Owner:      "agent",
		AtSeq:      2,
	})
	if err != nil {
		t.Fatalf("create latest anchor: %v", err)
	}

	view, err := svc.BuildView(context.Background(), ViewRequest{
		SessionID:    "session-1",
		Task:         "implement migration",
		BudgetTokens: 512,
	})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if view.AnchorID != latest.ID || view.AnchorID == first.ID {
		t.Fatal("expected latest anchor to be used")
	}
}

func TestBuildViewFallsBackToTapeHeadWithoutAnchor(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	if _, err := svc.Append(context.Background(), "session-1", AppendInput{
		Kind:    EntryUser,
		Content: "hello",
		Actor:   "user",
	}); err != nil {
		t.Fatalf("append entry: %v", err)
	}

	view, err := svc.BuildView(context.Background(), ViewRequest{
		SessionID:    "session-1",
		Task:         "implement migration",
		BudgetTokens: 256,
	})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if len(view.IncludedSeqs) == 0 {
		t.Fatal("expected entries from tape head to be included")
	}
}
