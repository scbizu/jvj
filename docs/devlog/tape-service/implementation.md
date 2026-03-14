# Tape Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a single-session append-only Tape Service with entries, linear anchors, sequential handoffs, and deterministic view assembly, without changing the external `AgentService` protocol.

**Architecture:** Add a new `internal/tape` package that owns fact writes, linear anchor creation, sequential handoff writes, and deterministic view assembly. Keep the design scoped to one active session per runtime instance: one session owns one tape, recovery always starts from the latest anchor, and no multi-tape or multi-path logic is introduced.

**Tech Stack:** Go 1.25, standard library `testing` used in a BDD-style/spec-oriented way, existing `internal/core` and `internal/session` packages, optional in-memory store implementation for initial coverage

**Testing Approach:** Use BDD-style behavior specs and scenario-oriented assertions. Keep `go test` and the standard library unless the repo later adopts a dedicated BDD framework.

---

### Task 1: Create the single-session tape model and append path

**Files:**
- Create: `internal/tape/types.go`
- Create: `internal/tape/service.go`
- Create: `internal/tape/memory_store.go`
- Create: `internal/tape/service_test.go`

**Step 1: Write the failing behavior spec**

```go
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
```

**Step 2: Run the spec to verify the behavior is not implemented yet**

Run: `go test ./internal/tape -run TestServiceAppendAssignsMonotonicSeq -v`

Expected: FAIL with `undefined: NewInMemoryStore` or `undefined: NewService`

**Step 3: Write minimal implementation**

```go
type AppendInput struct {
	Kind    EntryKind
	Content string
	Actor   string
	Metadata map[string]any
}

type Service struct {
	store Store
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
		CreatedAt: time.Now().UTC(),
		Actor:     in.Actor,
	}
	if err := s.store.PutEntry(ctx, sessionID, entry); err != nil {
		return nil, err
	}
	return entry, nil
}
```

**Step 4: Run the spec to verify the behavior now passes**

Run: `go test ./internal/tape -run TestServiceAppendAssignsMonotonicSeq -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tape/types.go internal/tape/service.go internal/tape/memory_store.go internal/tape/service_test.go
git commit -m "feat: add tape append service"
```

### Task 2: Add correction and linear anchor semantics

**Files:**
- Modify: `internal/tape/types.go`
- Modify: `internal/tape/service.go`
- Modify: `internal/tape/memory_store.go`
- Modify: `internal/tape/service_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestServiceAppendCorrectionKeepsOriginalEntry(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	original, _ := svc.Append(context.Background(), "session-1", AppendInput{
		Kind:    EntryAssistant,
		Content: "wrong",
		Actor:   "agent",
	})

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
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/tape -run 'TestServiceAppendCorrectionKeepsOriginalEntry|TestServiceCreateAnchorBuildsLinearChain' -v`

Expected: FAIL with `undefined: AppendCorrection` or `undefined: CreateAnchorInput`

**Step 3: Write minimal implementation**

```go
type CreateAnchorInput struct {
	PhaseTag   string
	Summary    string
	SourceSeqs []uint64
	State      map[string]any
	Owner      string
	AtSeq      uint64
}

func (s *Service) AppendCorrection(ctx context.Context, sessionID string, correctsSeq uint64, in AppendInput) (*Entry, error) {
	entry, err := s.Append(ctx, sessionID, in)
	if err != nil {
		return nil, err
	}
	entry.CorrectsSeq = &correctsSeq
	return entry, s.store.PutEntry(ctx, sessionID, entry)
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/tape -run 'TestServiceAppendCorrectionKeepsOriginalEntry|TestServiceCreateAnchorBuildsLinearChain' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tape/types.go internal/tape/service.go internal/tape/memory_store.go internal/tape/service_test.go
git commit -m "feat: add tape correction and anchor support"
```

### Task 3: Implement sequential handoff semantics

**Files:**
- Modify: `internal/tape/service.go`
- Modify: `internal/tape/memory_store.go`
- Modify: `internal/tape/service_test.go`

**Step 1: Write the failing behavior specs**

```go
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
}

func TestServiceHandoffKeepsEntryWhenAnchorWriteFails(t *testing.T) {
	store := NewFailingAnchorStore()
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

	if got := store.EntryCount("session-1"); got != 1 {
		t.Fatalf("expected handoff entry to remain, got %d persisted entries", got)
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/tape -run 'TestServiceHandoffWritesEntryAndAnchor|TestServiceHandoffKeepsEntryWhenAnchorWriteFails' -v`

Expected: FAIL with `undefined: HandoffInput` or single-session handoff assertions

**Step 3: Write minimal implementation**

```go
func (s *Service) Handoff(ctx context.Context, sessionID string, in HandoffInput) (*Anchor, error) {
	handoff, err := s.Append(ctx, sessionID, AppendInput{
		Kind:    EntryHandoff,
		Content: in.Summary,
		Actor:   in.Owner,
		Metadata: map[string]any{
			"next_steps": in.NextSteps,
			"phase_tag":  in.PhaseTag,
		},
	})
	if err != nil {
		return nil, err
	}

	anchor, err := s.CreateAnchor(ctx, sessionID, CreateAnchorInput{
		PhaseTag:   in.PhaseTag,
		Summary:    in.Summary,
		SourceSeqs: in.SourceSeqs,
		State:      in.StateDelta,
		Owner:      in.Owner,
		AtSeq:      handoff.Seq,
	})
	if err != nil {
		return nil, err
	}
	return anchor, nil
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/tape -run 'TestServiceHandoffWritesEntryAndAnchor|TestServiceHandoffKeepsEntryWhenAnchorWriteFails' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tape/service.go internal/tape/memory_store.go internal/tape/service_test.go
git commit -m "feat: add sequential tape handoff"
```

### Task 4: Build deterministic view assembly from the latest anchor

**Files:**
- Create: `internal/tape/view.go`
- Modify: `internal/tape/service.go`
- Modify: `internal/tape/memory_store.go`
- Modify: `internal/tape/service_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestBuildViewUsesLatestAnchor(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	first, _ := svc.CreateAnchor(context.Background(), "session-1", CreateAnchorInput{
		PhaseTag:   "discover",
		Summary:    "done",
		SourceSeqs: []uint64{1},
		State:      map[string]any{"phase_tag": "discover"},
		Owner:      "agent",
	})
	latest, _ := svc.CreateAnchor(context.Background(), "session-1", CreateAnchorInput{
		PhaseTag:   "implement",
		Summary:    "latest",
		SourceSeqs: []uint64{2},
		State:      map[string]any{"phase_tag": "implement"},
		Owner:      "agent",
	})

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

	_, _ = svc.Append(context.Background(), "session-1", AppendInput{
		Kind:    EntryUser,
		Content: "hello",
		Actor:   "user",
	})

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
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/tape -run 'TestBuildViewUsesLatestAnchor|TestBuildViewFallsBackToTapeHeadWithoutAnchor' -v`

Expected: FAIL with `undefined: BuildView` or latest-anchor assertions

**Step 3: Write minimal implementation**

```go
func (s *Service) BuildView(ctx context.Context, req ViewRequest) (*View, error) {
	anchor, err := s.store.GetLatestAnchor(ctx, req.SessionID)
	if err == nil {
		return &View{
			SessionID:    req.SessionID,
			AnchorID:     anchor.ID,
			IncludedSeqs: s.store.SeqsFrom(ctx, req.SessionID, anchor.AtSeq),
			Provenance:   anchor.SourceSeqs,
		}, nil
	}

	return &View{
		SessionID:    req.SessionID,
		IncludedSeqs: s.store.SeqsFrom(ctx, req.SessionID, 1),
		Provenance:   s.store.SeqsFrom(ctx, req.SessionID, 1),
	}, nil
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/tape -run 'TestBuildViewUsesLatestAnchor|TestBuildViewFallsBackToTapeHeadWithoutAnchor' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tape/view.go internal/tape/service.go internal/tape/memory_store.go internal/tape/service_test.go
git commit -m "feat: add tape view assembly"
```

### Task 5: Integrate the tape service into the single-session coordinator

**Files:**
- Modify: `internal/session/manager.go`
- Create: `internal/session/manager_test.go`
- Modify: `internal/core/agent_loop.go`
- Create: `internal/core/agent_loop_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestManagerOpenInitializesCurrentSession(t *testing.T) {
	mgr := NewManager()

	session, err := mgr.Open("session-1")
	if err != nil {
		t.Fatalf("open current session: %v", err)
	}
	if session.Tape == nil {
		t.Fatal("expected current session tape")
	}
}

func TestManagerRejectsSecondActiveAttach(t *testing.T) {
	mgr := NewManager()

	_, _ = mgr.Open("session-1")
	if _, err := mgr.Open("session-2"); err == nil {
		t.Fatal("expected single-session rejection")
	}
}

func TestAgentLoopRunAppendsUserEntry(t *testing.T) {
	store := tape.NewInMemoryStore()
	svc := tape.NewService(store)
	router := &Router{}
	loop := NewAgentLoop(router, svc)

	if _, err := loop.Run(context.Background(), "session-1", "hello"); err != nil {
		t.Fatalf("run: %v", err)
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/session ./internal/core -run 'TestManagerOpenInitializesCurrentSession|TestManagerRejectsSecondActiveAttach|TestAgentLoopRunAppendsUserEntry' -v`

Expected: FAIL with `undefined: Session`, `undefined: Open`, or signature mismatch errors

**Step 3: Write minimal implementation**

```go
type Session struct {
	ID       string
	Tape     *tape.Tape
	Attached bool
}

func (m *Manager) Open(id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current != nil && m.current.Attached {
		return nil, ErrActiveSessionExists
	}
	m.current = &Session{
		ID:       id,
		Tape:     &tape.Tape{SessionID: id},
		Attached: true,
	}
	return m.current, nil
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/session ./internal/core -run 'TestManagerOpenInitializesCurrentSession|TestManagerRejectsSecondActiveAttach|TestAgentLoopRunAppendsUserEntry' -v`

Expected: PASS

**Step 5: Run the full suite**

Run: `go test ./...`

Expected: all packages PASS

**Step 6: Commit**

```bash
git add internal/session/manager.go internal/session/manager_test.go internal/core/agent_loop.go internal/core/agent_loop_test.go internal/tape
git commit -m "feat: integrate tape service into core flow"
```
