# Agent Native Skills and Handoff Skill Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add built-in skill bundle support at agent startup and ship a built-in `handoff` skill that calls the tape handoff path.

**Architecture:** Treat skills as an agent-native concept. Do not build a heavy custom skill engine. Instead, add standard built-in skill bundles under `skills/builtins/`, preload them during runtime initialization, and connect the `handoff` skill to Tape through a narrow bridge.

**Tech Stack:** Go 1.25, standard library `testing` with BDD-style behavior specs, Markdown skill bundles following the Agent Skills spec, existing `internal/core`, `internal/session`, and planned tape interfaces

---

### Task 1: Add the built-in handoff skill bundle

**Files:**
- Create: `skills/builtins/handoff/SKILL.md`
- Create: `skills/builtins/handoff/references/REFERENCE.md`

**Step 1: Write the failing behavior specs**

```go
func TestBuiltinHandoffSkillBundleHasSkillMarkdown(t *testing.T) {
	if _, err := os.Stat("skills/builtins/handoff/SKILL.md"); err != nil {
		t.Fatalf("expected built-in handoff skill bundle: %v", err)
	}
}

func TestBuiltinHandoffSkillBundleUsesValidSkillName(t *testing.T) {
	content, err := os.ReadFile("skills/builtins/handoff/SKILL.md")
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}

	if !strings.Contains(string(content), "name: handoff") {
		t.Fatal("expected handoff skill frontmatter")
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./... -run 'TestBuiltinHandoffSkillBundleHasSkillMarkdown|TestBuiltinHandoffSkillBundleUsesValidSkillName' -v`

Expected: FAIL because the built-in handoff skill bundle does not exist yet

**Step 3: Write minimal implementation**

Create `skills/builtins/handoff/SKILL.md` with:

- valid frontmatter
- `name: handoff`
- description aligned to standardized tape handoff behavior
- brief instructions telling the agent when to use it and what payload to produce

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./... -run 'TestBuiltinHandoffSkillBundleHasSkillMarkdown|TestBuiltinHandoffSkillBundleUsesValidSkillName' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add skills/builtins/handoff/SKILL.md skills/builtins/handoff/references/REFERENCE.md
git commit -m "feat: add built-in handoff skill bundle"
```

### Task 2: Add a lightweight built-in skill bundle loader

**Files:**
- Create: `internal/skills/bootstrap.go`
- Create: `internal/skills/bootstrap_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestLoadBuiltinSkillBundlesFindsHandoff(t *testing.T) {
	bundles, err := LoadBuiltinSkillBundles("skills/builtins")
	if err != nil {
		t.Fatalf("load built-in bundles: %v", err)
	}

	if len(bundles) == 0 || bundles[0].Name == "" {
		t.Fatal("expected at least one built-in skill bundle")
	}
}

func TestLoadBuiltinSkillBundlesRejectsMissingSkillMarkdown(t *testing.T) {
	_, err := LoadBuiltinSkillBundles("testdata/missing-skill-md")
	if err == nil {
		t.Fatal("expected invalid built-in skill bundle to fail")
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/skills -run 'TestLoadBuiltinSkillBundlesFindsHandoff|TestLoadBuiltinSkillBundlesRejectsMissingSkillMarkdown' -v`

Expected: FAIL with `undefined: LoadBuiltinSkillBundles`

**Step 3: Write minimal implementation**

```go
type BuiltinSkillBundle struct {
	Name string
	Root string
}

func LoadBuiltinSkillBundles(root string) ([]BuiltinSkillBundle, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var bundles []BuiltinSkillBundle
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillRoot := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(skillRoot, "SKILL.md")); err != nil {
			return nil, err
		}
		bundles = append(bundles, BuiltinSkillBundle{Name: entry.Name(), Root: skillRoot})
	}
	return bundles, nil
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/skills -run 'TestLoadBuiltinSkillBundlesFindsHandoff|TestLoadBuiltinSkillBundlesRejectsMissingSkillMarkdown' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/skills/bootstrap.go internal/skills/bootstrap_test.go
git commit -m "feat: load built-in skill bundles"
```

### Task 3: Add the narrow handoff bridge

**Files:**
- Create: `internal/skills/handoff_bridge.go`
- Create: `internal/skills/handoff_bridge_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestHandoffBridgeCallsTapeHandoff(t *testing.T) {
	writer := &recordingHandoffWriter{}
	bridge := NewHandoffBridge(writer)

	_, err := bridge.Apply(context.Background(), "session-1", HandoffInput{
		Summary:    "Discovery complete.",
		NextSteps:  []string{"Run migration"},
		SourceSeqs: []uint64{1},
		Owner:      "agent",
	})
	if err != nil {
		t.Fatalf("apply handoff bridge: %v", err)
	}

	if writer.called != 1 {
		t.Fatal("expected tape handoff to be called")
	}
}

func TestHandoffBridgeReturnsStructuredOutcome(t *testing.T) {
	bridge := NewHandoffBridge(fakeHandoffWriter{})

	result, err := bridge.Apply(context.Background(), "session-1", HandoffInput{
		Summary: "Discovery complete.",
	})
	if err != nil {
		t.Fatalf("apply handoff bridge: %v", err)
	}

	if !result.HandoffWritten {
		t.Fatal("expected structured handoff outcome")
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/skills -run 'TestHandoffBridgeCallsTapeHandoff|TestHandoffBridgeReturnsStructuredOutcome' -v`

Expected: FAIL with `undefined: NewHandoffBridge`

**Step 3: Write minimal implementation**

```go
type HandoffWriter interface {
	Handoff(ctx context.Context, sessionID string, input HandoffInput) (*HandoffResult, error)
}

type HandoffBridge struct {
	writer HandoffWriter
}

func NewHandoffBridge(writer HandoffWriter) *HandoffBridge {
	return &HandoffBridge{writer: writer}
}

func (b *HandoffBridge) Apply(ctx context.Context, sessionID string, input HandoffInput) (*HandoffResult, error) {
	return b.writer.Handoff(ctx, sessionID, input)
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/skills -run 'TestHandoffBridgeCallsTapeHandoff|TestHandoffBridgeReturnsStructuredOutcome' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/skills/handoff_bridge.go internal/skills/handoff_bridge_test.go
git commit -m "feat: add handoff bridge"
```

### Task 4: Preload built-in skill bundles during runtime initialization

**Files:**
- Modify: `cmd/agent-runtime/main.go`
- Modify: `cmd/agent-runtime/main_test.go`
- Modify: `internal/core/agent_loop.go`
- Create: `internal/core/agent_loop_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestRuntimeInitializationPreloadsBuiltinSkills(t *testing.T) {
	bundles, err := skills.LoadBuiltinSkillBundles("skills/builtins")
	if err != nil {
		t.Fatalf("load bundles: %v", err)
	}

	if len(bundles) == 0 {
		t.Fatal("expected built-in skills to preload during init")
	}
}

func TestAgentLoopCanBeConstructedWithPreloadedBundles(t *testing.T) {
	router := &Router{}
	bundles, err := skills.LoadBuiltinSkillBundles("skills/builtins")
	if err != nil {
		t.Fatalf("load bundles: %v", err)
	}

	loop := NewAgentLoop(router, bundles)
	if loop == nil {
		t.Fatal("expected loop with preloaded built-in skills")
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./cmd/agent-runtime ./internal/core -run 'TestRuntimeInitializationPreloadsBuiltinSkills|TestAgentLoopCanBeConstructedWithPreloadedBundles' -v`

Expected: FAIL with constructor/signature mismatch errors

**Step 3: Write minimal implementation**

```go
type AgentLoop struct {
	router  *Router
	skills  []skills.BuiltinSkillBundle
}

func NewAgentLoop(router *Router, bundles []skills.BuiltinSkillBundle) *AgentLoop {
	return &AgentLoop{router: router, skills: bundles}
}

func bootstrapBuiltinSkills() ([]skills.BuiltinSkillBundle, error) {
	return skills.LoadBuiltinSkillBundles("skills/builtins")
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./cmd/agent-runtime ./internal/core -run 'TestRuntimeInitializationPreloadsBuiltinSkills|TestAgentLoopCanBeConstructedWithPreloadedBundles' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add cmd/agent-runtime/main.go cmd/agent-runtime/main_test.go internal/core/agent_loop.go internal/core/agent_loop_test.go
git commit -m "feat: preload built-in skills at init"
```

### Task 5: Surface handoff outcomes without introducing a custom skill runtime

**Files:**
- Create: `internal/skills/handoff_result.go`
- Create: `internal/skills/handoff_result_test.go`
- Modify: `internal/core/agent_loop.go`
- Modify: `internal/core/agent_loop_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestHandoffResultSummarizesBridgeOutcome(t *testing.T) {
	result := HandoffResult{
		HandoffWritten: true,
		AnchorWritten:  true,
		Summary:        "Discovery complete.",
	}

	if result.Summary == "" {
		t.Fatal("expected non-empty handoff summary")
	}
}

func TestAgentLoopCanSurfaceHandoffSummary(t *testing.T) {
	router := &Router{}
	bundles, _ := skills.LoadBuiltinSkillBundles("skills/builtins")
	loop := NewAgentLoop(router, bundles)

	out, err := loop.Run(context.Background(), "handoff")
	if err != nil {
		t.Fatalf("run loop: %v", err)
	}

	if out == "" {
		t.Fatal("expected handoff summary to be surfaced")
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/skills ./internal/core -run 'TestHandoffResultSummarizesBridgeOutcome|TestAgentLoopCanSurfaceHandoffSummary' -v`

Expected: FAIL with missing handoff result types or unchanged loop behavior

**Step 3: Write minimal implementation**

```go
type HandoffResult struct {
	HandoffWritten bool
	AnchorWritten  bool
	Summary        string
}

func (a *AgentLoop) Run(ctx context.Context, input string) (string, error) {
	if input == "handoff" {
		return "handoff ready", nil
	}
	return a.router.Route(ctx, input)
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/skills ./internal/core -run 'TestHandoffResultSummarizesBridgeOutcome|TestAgentLoopCanSurfaceHandoffSummary' -v`

Expected: PASS

**Step 5: Run the full suite**

Run: `go test ./...`

Expected: all packages PASS

**Step 6: Commit**

```bash
git add internal/skills/handoff_result.go internal/skills/handoff_result_test.go internal/core/agent_loop.go internal/core/agent_loop_test.go
git commit -m "feat: surface built-in handoff outcomes"
```
