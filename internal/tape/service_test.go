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
