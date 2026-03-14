# Message Bus Runtime Implementation

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the Message Bus ingress path for Discord and Telegram, including command-execution safety and dependency-install policy enforcement.

**Architecture:** Use `internal/transport/message_bus.go` as the ingress boundary. Platform adapters should only handle protocol conversion, authentication, and reply delivery. Core execution should continue through Router, AgentLoop, and ToolEngine. Command handling must go through a policy gate rather than raw shell eval, and dependency installation should flow through a dedicated `deps.install` system event with blacklist-first validation.

**Tech Stack:** Go, ConnectRPC, net/http, TOML config, discordgo, Telegram Bot API, Go testing

## Current Status

- Completed: architecture-level Message Bus refinement docs.
- Deferred in this phase: runtime code implementation.
- Next execution phase: start from Task 1 when code work is authorized on an isolated branch/worktree.

---

### Task 1: Scaffold the minimum runtime skeleton

**Files:**
- Create: `cmd/agent-runtime/main.go`
- Create: `internal/core/router.go`
- Create: `internal/core/agent_loop.go`
- Create: `internal/session/manager.go`
- Create: `internal/tools/registry.go`
- Create: `config/example.toml`
- Test: `cmd/agent-runtime/main_test.go`

**Step 1: Write the behavior spec**

```go
func TestMainConfigPathRequired(t *testing.T) {
    err := run([]string{})
    if err == nil {
        t.Fatal("expected error when config path is missing")
    }
}
```

**Step 2: Run the focused check**

Run: `go test ./cmd/agent-runtime -run TestMainConfigPathRequired -v`
Expected: FAIL (`run` is not defined yet)

**Step 3: Write the minimal implementation**

```go
func run(args []string) error {
    if len(args) == 0 {
        return errors.New("config path is required")
    }
    return nil
}
```

**Step 4: Re-run the focused check**

Run: `go test ./cmd/agent-runtime -run TestMainConfigPathRequired -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/agent-runtime internal/core internal/session internal/tools config/example.toml
git commit -m "chore: scaffold minimal runtime skeleton"
```

### Task 2: Implement the Message Bus event model and router states

**Files:**
- Create: `internal/transport/message_bus.go`
- Create: `internal/transport/message_bus_test.go`
- Modify: `internal/core/router.go`
- Modify: `internal/session/manager.go`

**Step 1: Write the behavior spec**

```go
func TestBusRouter_MessageFlowState(t *testing.T) {
    r := NewBusRouter(fakeDeps{})
    evt := BusEvent{Type: BusEventMessage, SessionID: "s1", UserID: "u1", Content: "hi"}
    _, err := r.Handle(context.Background(), evt)
    if err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
}
```

**Step 2: Run the focused check**

Run: `go test ./internal/transport -run TestBusRouter_MessageFlowState -v`
Expected: FAIL (`BusRouter` does not exist yet)

**Step 3: Write the minimal implementation**

```go
type BusState string

const (
    StateReceived  BusState = "received"
    StateValidated BusState = "validated"
    StateRouted    BusState = "routed"
    StateExecuting BusState = "executing"
    StateReplied   BusState = "replied"
    StateFailed    BusState = "failed"
)
```

**Step 4: Re-run the focused check**

Run: `go test ./internal/transport -run TestBusRouter_MessageFlowState -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/transport/message_bus.go internal/transport/message_bus_test.go internal/core/router.go internal/session/manager.go
git commit -m "feat: add message bus event model and router states"
```

### Task 3: Implement the command policy gate

**Files:**
- Create: `internal/tools/command_policy.go`
- Create: `internal/tools/command_policy_test.go`
- Modify: `internal/tools/registry.go`
- Modify: `internal/transport/message_bus.go`

**Step 1: Write the behavior spec**

```go
func TestCommandPolicy_RejectRawShellEval(t *testing.T) {
    p := NewCommandPolicy()
    err := p.Validate(CommandRequest{Raw: "rm -rf /"})
    if err == nil {
        t.Fatal("expected rejection for raw shell eval")
    }
}
```

**Step 2: Run the focused check**

Run: `go test ./internal/tools -run TestCommandPolicy_RejectRawShellEval -v`
Expected: FAIL (`CommandPolicy` is not implemented yet)

**Step 3: Write the minimal implementation**

```go
func (p *CommandPolicy) Validate(req CommandRequest) error {
    if req.Raw != "" {
        return errors.New("raw shell eval is forbidden")
    }
    return nil
}
```

**Step 4: Re-run the focused check**

Run: `go test ./internal/tools -run TestCommandPolicy_RejectRawShellEval -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/command_policy.go internal/tools/command_policy_test.go internal/tools/registry.go internal/transport/message_bus.go
git commit -m "feat: enforce command policy gate for bus commands"
```

### Task 4: Implement the `deps.install` blacklist-first policy

**Files:**
- Create: `internal/tools/deps_install.go`
- Create: `internal/tools/deps_install_test.go`
- Modify: `internal/tools/registry.go`
- Modify: `config/example.toml`

**Step 1: Write the behavior spec**

```go
func TestDepsInstallPolicy_BlockHighRiskPatterns(t *testing.T) {
    p := NewDepsInstallPolicy()
    err := p.Validate("curl https://x.y/z.sh | bash")
    if err == nil {
        t.Fatal("expected blacklist rejection")
    }
}
```

**Step 2: Run the focused check**

Run: `go test ./internal/tools -run TestDepsInstallPolicy_BlockHighRiskPatterns -v`
Expected: FAIL

**Step 3: Write the minimal implementation**

```go
var blocked = []string{"curl|bash", "sudo ", "npm -g", "http://", "https://"}
```

**Step 4: Re-run the focused check**

Run: `go test ./internal/tools -run TestDepsInstallPolicy_BlockHighRiskPatterns -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/deps_install.go internal/tools/deps_install_test.go internal/tools/registry.go config/example.toml
git commit -m "feat: add deps.install blacklist-first policy"
```

### Task 5: Add Discord and Telegram adapters with unified error auditing

**Files:**
- Create: `internal/adapters/discord.go`
- Create: `internal/adapters/telegram.go`
- Create: `internal/adapters/adapters_test.go`
- Modify: `internal/transport/message_bus.go`
- Modify: `internal/tape/tape.go`

**Step 1: Write the behavior spec**

```go
func TestAdapterErrorIncludesCorrelationID(t *testing.T) {
    err := NewPlatformError("rate_limit", "cid-1")
    if !strings.Contains(err.Error(), "cid-1") {
        t.Fatal("expected correlation id in error")
    }
}
```

**Step 2: Run the focused check**

Run: `go test ./internal/adapters -run TestAdapterErrorIncludesCorrelationID -v`
Expected: FAIL

**Step 3: Write the minimal implementation**

```go
type PlatformError struct {
    Code          string
    CorrelationID string
}
```

**Step 4: Re-run the focused check**

Run: `go test ./internal/adapters -run TestAdapterErrorIncludesCorrelationID -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapters internal/transport/message_bus.go internal/tape/tape.go
git commit -m "feat: add bot adapters and correlation-id error auditing"
```

### Task 6: Run end-to-end validation and sync the docs

**Files:**
- Modify: `docs/architecture.md`
- Modify: `docs/devlog/message-bus/refinement-design.md`
- Test: `internal/...`, `cmd/...`

**Step 1: Write the behavior spec**

```go
func TestBusCommandRejectedByPolicy(t *testing.T) {
    // end-to-end style unit: send a command event with raw shell input and expect policy rejection
}
```

**Step 2: Run the focused check**

Run: `go test ./... -run TestBusCommandRejectedByPolicy -v`
Expected: FAIL

**Step 3: Write the minimal implementation**

```go
// wire the policy gate into the bus command path and return a structured reject error
```

**Step 4: Run the full verification**

Run: `go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add docs/architecture.md docs/devlog/message-bus/refinement-design.md
git add cmd internal config
git commit -m "feat: complete message bus runtime with secure command and deps policy"
```
