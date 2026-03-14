package tape

import "time"

type EntryKind string

const (
	EntryUser      EntryKind = "user"
	EntryAssistant EntryKind = "assistant"
	EntryToolCall  EntryKind = "tool_call"
	EntryToolResult EntryKind = "tool_result"
	EntrySystem    EntryKind = "system"
	EntryCorrection EntryKind = "correction"
	EntryHandoff   EntryKind = "handoff"
)

type Tape struct {
	SessionID string
	HeadSeq   uint64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Anchor struct {
	ID           string
	SessionID    string
	AtSeq        uint64
	PrevAnchorID string
	PhaseTag     string
	Summary      string
	State        map[string]any
	SourceSeqs   []uint64
	CreatedAt    time.Time
	Owner        string
}

type Entry struct {
	Seq         uint64
	Kind        EntryKind
	Content     string
	Metadata    map[string]any
	CorrectsSeq *uint64
	CreatedAt   time.Time
	Actor       string
}

type AppendInput struct {
	Kind     EntryKind
	Content  string
	Metadata map[string]any
	Actor    string
}

type CreateAnchorInput struct {
	PhaseTag   string
	Summary    string
	SourceSeqs []uint64
	State      map[string]any
	Owner      string
	AtSeq      uint64
}

type HandoffInput struct {
	Summary    string
	NextSteps  []string
	SourceSeqs []uint64
	Owner      string
	PhaseTag   string
	StateDelta map[string]any
}
