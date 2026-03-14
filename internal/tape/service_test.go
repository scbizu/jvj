package tape

import (
	"context"
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
