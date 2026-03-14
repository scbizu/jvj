# Tape Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an internal append-only Tape Service with entries, anchors, handoffs, and view assembly, without changing the external `AgentService` protocol.

**Architecture:** Add a new `internal/tape` package that owns fact writes, anchor creation, handoff transactions, and view assembly. Implement the core semantics behind tests first, then connect session/core packages to the new tape primitives through small integration seams instead of broad runtime rewrites.

**Tech Stack:** Go 1.25, standard library `testing`, existing `internal/core` and `internal/session` packages, optional in-memory store implementation for initial coverage

---

### Task 1: Create the tape domain model and append path

**Files:**
- Create: `internal/tape/types.go`
- Create: `internal/tape/service.go`
- Create: `internal/tape/memory_store.go`
- Create: `internal/tape/service_test.go`

**Step 1: Write the failing test**

```go
func TestServiceAppendAssignsMonotonicSeq(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	first, err := svc.Append(context.Background(), "tape-1", AppendInput{
		Kind:    EntryUser,
		Content: "hello",
		Actor:   "user",
	})
	if err != nil {
		t.Fatalf("append first entry: %v", err)
	}

	second, err := svc.Append(context.Background(), "tape-1", AppendInput{
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

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tape -run TestServiceAppendAssignsMonotonicSeq -v`

Expected: FAIL with `undefined: NewInMemoryStore` or `undefined: NewService`

**Step 3: Write minimal implementation**

```go
type AppendInput struct {
	Kind    EntryKind
	Content string
	Actor   string
}

type Service struct {
	store Store
}

func (s *Service) Append(ctx context.Context, tapeID TapeID, in AppendInput) (*Entry, error) {
	seq, err := s.store.NextSeq(ctx, tapeID)
	if err != nil {
		return nil, err
	}
	entry := &Entry{
		TapeID:    tapeID,
		Seq:       seq,
		Kind:      in.Kind,
		Content:   in.Content,
		CreatedAt: time.Now().UTC(),
		Actor:     in.Actor,
	}
	if err := s.store.PutEntry(ctx, entry); err != nil {
		return nil, err
	}
	return entry, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tape -run TestServiceAppendAssignsMonotonicSeq -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tape/types.go internal/tape/service.go internal/tape/memory_store.go internal/tape/service_test.go
git commit -m "feat: add tape append service"
```

### Task 2: Add correction and anchor creation semantics

**Files:**
- Modify: `internal/tape/types.go`
- Modify: `internal/tape/service.go`
- Modify: `internal/tape/memory_store.go`
- Modify: `internal/tape/service_test.go`

**Step 1: Write the failing tests**

```go
func TestServiceAppendCorrectionKeepsOriginalEntry(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	original, _ := svc.Append(context.Background(), "tape-1", AppendInput{
		Kind:    EntryAssistant,
		Content: "wrong",
		Actor:   "agent",
	})

	correction, err := svc.AppendCorrection(context.Background(), "tape-1", original.Seq, AppendInput{
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

func TestServiceCreateAnchorCapturesMinimalState(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	anchor, err := svc.CreateAnchor(context.Background(), "tape-1", CreateAnchorInput{
		Phase:      "discover",
		Summary:    "summary",
		SourceSeqs: []uint64{1, 2},
		State: map[string]any{
			"phase": "discover",
		},
		Owner: "agent",
	})
	if err != nil {
		t.Fatalf("create anchor: %v", err)
	}

	if anchor.Phase != "discover" || anchor.Owner != "agent" {
		t.Fatalf("unexpected anchor: %#v", anchor)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/tape -run 'TestServiceAppendCorrectionKeepsOriginalEntry|TestServiceCreateAnchorCapturesMinimalState' -v`

Expected: FAIL with `undefined: AppendCorrection` or `undefined: CreateAnchorInput`

**Step 3: Write minimal implementation**

```go
type CreateAnchorInput struct {
	Phase      string
	Summary    string
	SourceSeqs []uint64
	State      map[string]any
	Owner      string
}

func (s *Service) AppendCorrection(ctx context.Context, tapeID TapeID, correctsSeq uint64, in AppendInput) (*Entry, error) {
	entry, err := s.Append(ctx, tapeID, in)
	if err != nil {
		return nil, err
	}
	entry.CorrectsSeq = &correctsSeq
	return entry, s.store.PutEntry(ctx, entry)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/tape -run 'TestServiceAppendCorrectionKeepsOriginalEntry|TestServiceCreateAnchorCapturesMinimalState' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tape/types.go internal/tape/service.go internal/tape/memory_store.go internal/tape/service_test.go
git commit -m "feat: add tape correction and anchor support"
```

### Task 3: Implement transactional handoff semantics

**Files:**
- Modify: `internal/tape/service.go`
- Modify: `internal/tape/memory_store.go`
- Modify: `internal/tape/service_test.go`

**Step 1: Write the failing tests**

```go
func TestServiceHandoffWritesEntryAndAnchor(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	anchor, err := svc.Handoff(context.Background(), "tape-1", HandoffInput{
		FromPhase:  "discover",
		ToPhase:    "implement",
		Summary:    "Discovery complete.",
		NextSteps:  []string{"Run migration"},
		SourceSeqs: []uint64{1, 2},
		Owner:      "agent",
	})
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}

	if anchor.Phase != "implement" {
		t.Fatalf("expected implement anchor, got %q", anchor.Phase)
	}
}

func TestServiceHandoffRollsBackWhenAnchorWriteFails(t *testing.T) {
	store := NewFailingAnchorStore()
	svc := NewService(store)

	_, err := svc.Handoff(context.Background(), "tape-1", HandoffInput{
		FromPhase:  "discover",
		ToPhase:    "implement",
		Summary:    "Discovery complete.",
		NextSteps:  []string{"Run migration"},
		SourceSeqs: []uint64{1},
		Owner:      "agent",
	})
	if err == nil {
		t.Fatal("expected transactional failure")
	}

	if got := store.EntryCount("tape-1"); got != 0 {
		t.Fatalf("expected rollback, got %d persisted entries", got)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/tape -run 'TestServiceHandoffWritesEntryAndAnchor|TestServiceHandoffRollsBackWhenAnchorWriteFails' -v`

Expected: FAIL with `undefined: HandoffInput` or rollback assertion failures

**Step 3: Write minimal implementation**

```go
func (s *Service) Handoff(ctx context.Context, tapeID TapeID, in HandoffInput) (*Anchor, error) {
	return s.store.WithTx(ctx, tapeID, func(tx TxStore) (*Anchor, error) {
		handoff, err := tx.AppendEntry(ctx, tapeID, AppendInput{
			Kind:       EntryHandoff,
			Content:    in.Summary,
			Actor:      in.Owner,
			SourceSeqs: in.SourceSeqs,
		})
		if err != nil {
			return nil, err
		}

		anchor := &Anchor{
			TapeID:     tapeID,
			AtSeq:      handoff.Seq,
			Phase:      in.ToPhase,
			Summary:    in.Summary,
			SourceSeqs: in.SourceSeqs,
			Owner:      in.Owner,
		}
		if err := tx.PutAnchor(ctx, anchor); err != nil {
			return nil, err
		}
		return anchor, nil
	})
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/tape -run 'TestServiceHandoffWritesEntryAndAnchor|TestServiceHandoffRollsBackWhenAnchorWriteFails' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tape/service.go internal/tape/memory_store.go internal/tape/service_test.go
git commit -m "feat: add tape handoff transactions"
```

### Task 4: Build view assembly and explanation support

**Files:**
- Create: `internal/tape/view.go`
- Modify: `internal/tape/service.go`
- Modify: `internal/tape/memory_store.go`
- Modify: `internal/tape/service_test.go`

**Step 1: Write the failing tests**

```go
func TestBuildViewUsesLatestRelevantAnchor(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	_, _ = svc.CreateAnchor(context.Background(), "tape-1", CreateAnchorInput{
		Phase:      "discover",
		Summary:    "done",
		SourceSeqs: []uint64{1},
		State:      map[string]any{"phase": "discover"},
		Owner:      "agent",
	})

	view, err := svc.BuildView(context.Background(), ViewRequest{
		TapeID:       "tape-1",
		Task:         "implement migration",
		BudgetTokens: 512,
	})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if len(view.AnchorPath) == 0 {
		t.Fatal("expected anchor path to be populated")
	}
}

func TestExplainViewReturnsIncludedAndOmittedRanges(t *testing.T) {
	store := NewInMemoryStore()
	svc := NewService(store)

	expl, err := svc.ExplainView(context.Background(), ViewRequest{
		TapeID:       "tape-1",
		Task:         "implement migration",
		BudgetTokens: 256,
	})
	if err != nil {
		t.Fatalf("explain view: %v", err)
	}

	if expl == nil || len(expl.Reasons) == 0 {
		t.Fatal("expected explanation reasons")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/tape -run 'TestBuildViewUsesLatestRelevantAnchor|TestExplainViewReturnsIncludedAndOmittedRanges' -v`

Expected: FAIL with `undefined: BuildView` or `undefined: ExplainView`

**Step 3: Write minimal implementation**

```go
type ViewExplanation struct {
	IncludedSeqs  []uint64
	OmittedRanges [][2]uint64
	Reasons       []string
}

func (s *Service) BuildView(ctx context.Context, req ViewRequest) (*View, error) {
	anchor, err := s.store.ResolveLatestAnchor(ctx, req.TapeID, req.PreferredFrom)
	if err != nil {
		return nil, err
	}
	return &View{
		TapeID:       req.TapeID,
		AnchorPath:   []string{anchor.ID},
		IncludedSeqs: anchor.SourceSeqs,
		Provenance:   anchor.SourceSeqs,
	}, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/tape -run 'TestBuildViewUsesLatestRelevantAnchor|TestExplainViewReturnsIncludedAndOmittedRanges' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tape/view.go internal/tape/service.go internal/tape/memory_store.go internal/tape/service_test.go
git commit -m "feat: add tape view assembly"
```

### Task 5: Integrate tape ownership into session and core packages

**Files:**
- Modify: `internal/session/manager.go`
- Create: `internal/session/manager_test.go`
- Modify: `internal/core/agent_loop.go`
- Create: `internal/core/agent_loop_test.go`

**Step 1: Write the failing tests**

```go
func TestManagerCreateSessionInitializesTapeID(t *testing.T) {
	mgr := NewManager()

	session := mgr.Create("session-1")
	if session.TapeID == "" {
		t.Fatal("expected tape id")
	}
}

func TestAgentLoopRunAppendsUserEntry(t *testing.T) {
	store := tape.NewInMemoryStore()
	svc := tape.NewService(store)
	router := &Router{}
	loop := NewAgentLoop(router, svc)

	if _, err := loop.Run(context.Background(), "tape-1", "hello"); err != nil {
		t.Fatalf("run: %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/session ./internal/core -run 'TestManagerCreateSessionInitializesTapeID|TestAgentLoopRunAppendsUserEntry' -v`

Expected: FAIL with `undefined: Session`, `undefined: Create`, or signature mismatch errors

**Step 3: Write minimal implementation**

```go
type Session struct {
	ID     string
	TapeID tape.TapeID
}

func (m *Manager) Create(id string) Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := Session{ID: id, TapeID: tape.TapeID(id)}
	m.sessions[id] = s
	return s
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/session ./internal/core -run 'TestManagerCreateSessionInitializesTapeID|TestAgentLoopRunAppendsUserEntry' -v`

Expected: PASS

**Step 5: Run the full suite**

Run: `go test ./...`

Expected: all packages PASS

**Step 6: Commit**

```bash
git add internal/session/manager.go internal/session/manager_test.go internal/core/agent_loop.go internal/core/agent_loop_test.go internal/tape
git commit -m "feat: integrate tape service into core flow"
```
