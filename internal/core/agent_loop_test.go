package core

import (
	"context"
	"testing"

	"github.com/scbizu/jvj/internal/tape"
)

type recordingTapeWriter struct {
	inputs []tape.AppendInput
}

func (r *recordingTapeWriter) Append(_ context.Context, _ string, in tape.AppendInput) (*tape.Entry, error) {
	r.inputs = append(r.inputs, in)
	return &tape.Entry{Seq: uint64(len(r.inputs)), Kind: in.Kind, Content: in.Content, Actor: in.Actor}, nil
}

func TestAgentLoopRunAppendsUserEntry(t *testing.T) {
	writer := &recordingTapeWriter{}
	router := &Router{}
	loop := NewAgentLoop(router, writer)

	if _, err := loop.Run(context.Background(), "session-1", "hello"); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(writer.inputs) == 0 {
		t.Fatal("expected at least one append")
	}
	if writer.inputs[0].Kind != tape.EntryUser {
		t.Fatalf("expected first append to be user, got %s", writer.inputs[0].Kind)
	}
}

func TestAgentLoopRunAppendsAssistantEntry(t *testing.T) {
	writer := &recordingTapeWriter{}
	router := &Router{}
	loop := NewAgentLoop(router, writer)

	out, err := loop.Run(context.Background(), "session-1", "hello")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if out != "hello" {
		t.Fatalf("expected echoed output, got %q", out)
	}
	if len(writer.inputs) != 2 {
		t.Fatalf("expected two appended entries, got %d", len(writer.inputs))
	}
	if writer.inputs[1].Kind != tape.EntryAssistant {
		t.Fatalf("expected second append to be assistant, got %s", writer.inputs[1].Kind)
	}
}
