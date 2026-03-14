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
