# Scripted Executor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a command-execution pipeline that plans first, compiles a one-shot shell script second, and only then executes it in a sandbox.

**Architecture:** Keep command-like tool calls inside `internal/tools`, but split execution into three parts: `ExecutionPlan`, `ScriptBuilder`, and `ScriptExecutor`. Integrate that flow into the existing policy/registry/core path without introducing multi-session logic or direct raw-shell execution.

**Tech Stack:** Go 1.25, standard library `testing` in BDD-style behavior specs, existing `internal/tools` and `internal/core` packages

---

### Task 1: Define the execution plan model

**Files:**
- Create: `internal/tools/execution_plan.go`
- Create: `internal/tools/execution_plan_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestExecutionPlanCommandRequestRequiresGoalAndSteps(t *testing.T) {
	req := CommandRequest{
		Goal: "fix config",
		Plan: &ExecutionPlan{
			Steps: []PlanStep{{Name: "check", Script: "pwd"}},
		},
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected plan-backed request to be valid: %v", err)
	}
}

func TestExecutionPlanRejectsEmptyStepScript(t *testing.T) {
	plan := ExecutionPlan{
		Goal:  "fix config",
		Steps: []PlanStep{{Name: "check"}},
	}

	if err := plan.Validate(); err == nil {
		t.Fatal("expected empty step script to be rejected")
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/tools -run 'TestExecutionPlanCommandRequestRequiresGoalAndSteps|TestExecutionPlanRejectsEmptyStepScript' -v`

Expected: FAIL with `undefined: ExecutionPlan` or `CommandRequest` validation errors

**Step 3: Write minimal implementation**

```go
type PlanStep struct {
	Name   string
	Script string
}

type ExecutionPlan struct {
	Goal  string
	Steps []PlanStep
}

func (p ExecutionPlan) Validate() error {
	if p.Goal == "" {
		return errors.New("goal is required")
	}
	if len(p.Steps) == 0 {
		return errors.New("at least one step is required")
	}
	for _, step := range p.Steps {
		if step.Script == "" {
			return errors.New("step script is required")
		}
	}
	return nil
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/tools -run 'TestExecutionPlanCommandRequestRequiresGoalAndSteps|TestExecutionPlanRejectsEmptyStepScript' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/execution_plan.go internal/tools/execution_plan_test.go
git commit -m "feat: add execution plan model"
```

### Task 2: Update command policy to require plan-backed command execution

**Files:**
- Modify: `internal/tools/command_policy.go`
- Modify: `internal/tools/command_policy_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestCommandPolicy_RejectsCommandWithoutExecutionPlan(t *testing.T) {
	p := NewCommandPolicy()

	err := p.Validate(CommandRequest{
		Argv: []string{"go", "test", "./..."},
	})
	if err == nil {
		t.Fatal("expected direct argv execution to be rejected")
	}
}

func TestCommandPolicy_AllowsPlanBackedCommandRequest(t *testing.T) {
	p := NewCommandPolicy()

	err := p.Validate(CommandRequest{
		Goal: "run tests",
		Plan: &ExecutionPlan{
			Goal: "run tests",
			Steps: []PlanStep{{Name: "test", Script: "go test ./..."}},
		},
	})
	if err != nil {
		t.Fatalf("expected plan-backed command request to pass: %v", err)
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/tools -run 'TestCommandPolicy_RejectsCommandWithoutExecutionPlan|TestCommandPolicy_AllowsPlanBackedCommandRequest' -v`

Expected: FAIL because direct argv execution is still accepted

**Step 3: Write minimal implementation**

```go
type CommandRequest struct {
	Raw  string
	Argv []string
	Goal string
	Plan *ExecutionPlan
}

func (req CommandRequest) Validate() error {
	if req.Raw != "" {
		return errors.New("raw shell eval is forbidden")
	}
	if req.Plan == nil {
		return errors.New("execution plan is required")
	}
	if err := req.Plan.Validate(); err != nil {
		return err
	}
	return nil
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/tools -run 'TestCommandPolicy_RejectsCommandWithoutExecutionPlan|TestCommandPolicy_AllowsPlanBackedCommandRequest' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/command_policy.go internal/tools/command_policy_test.go
git commit -m "feat: require plan-backed command execution"
```

### Task 3: Build one-shot script artifacts from execution plans

**Files:**
- Create: `internal/tools/script_builder.go`
- Create: `internal/tools/script_builder_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestScriptBuilderBuildsOneShotScriptWithSafetyHeader(t *testing.T) {
	builder := NewScriptBuilder("/tmp")

	artifact, err := builder.Build(ExecutionPlan{
		Goal: "run tests",
		Steps: []PlanStep{{Name: "test", Script: "go test ./..."}},
	})
	if err != nil {
		t.Fatalf("build script: %v", err)
	}

	if !strings.Contains(artifact.Content, "set -euo pipefail") {
		t.Fatal("expected safety header in script")
	}
}

func TestScriptBuilderIncludesPlanStepsInOrder(t *testing.T) {
	builder := NewScriptBuilder("/tmp")

	artifact, err := builder.Build(ExecutionPlan{
		Goal: "fix",
		Steps: []PlanStep{
			{Name: "check", Script: "pwd"},
			{Name: "apply", Script: "go test ./..."},
		},
	})
	if err != nil {
		t.Fatalf("build script: %v", err)
	}

	if strings.Index(artifact.Content, "pwd") > strings.Index(artifact.Content, "go test ./...") {
		t.Fatal("expected script steps to preserve plan order")
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/tools -run 'TestScriptBuilderBuildsOneShotScriptWithSafetyHeader|TestScriptBuilderIncludesPlanStepsInOrder' -v`

Expected: FAIL with `undefined: NewScriptBuilder`

**Step 3: Write minimal implementation**

```go
type ScriptArtifact struct {
	Path    string
	Hash    string
	Content string
}

type ScriptBuilder struct {
	baseDir string
}

func NewScriptBuilder(baseDir string) *ScriptBuilder {
	return &ScriptBuilder{baseDir: baseDir}
}

func (b *ScriptBuilder) Build(plan ExecutionPlan) (*ScriptArtifact, error) {
	var body strings.Builder
	body.WriteString("#!/usr/bin/env bash\nset -euo pipefail\n")
	for _, step := range plan.Steps {
		body.WriteString(step.Script)
		body.WriteString("\n")
	}
	content := body.String()
	return &ScriptArtifact{
		Path:    filepath.Join(b.baseDir, "executor.sh"),
		Hash:    hash(content),
		Content: content,
	}, nil
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/tools -run 'TestScriptBuilderBuildsOneShotScriptWithSafetyHeader|TestScriptBuilderIncludesPlanStepsInOrder' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/script_builder.go internal/tools/script_builder_test.go
git commit -m "feat: add one-shot script builder"
```

### Task 4: Execute temporary scripts and return structured results

**Files:**
- Create: `internal/tools/script_executor.go`
- Create: `internal/tools/script_executor_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestScriptExecutorRunsGeneratedScript(t *testing.T) {
	executor := NewScriptExecutor()

	result, err := executor.Execute(context.Background(), &ScriptArtifact{
		Content: "#!/usr/bin/env bash\nset -euo pipefail\necho hello\n",
	})
	if err != nil {
		t.Fatalf("execute script: %v", err)
	}

	if strings.TrimSpace(result.Stdout) != "hello" {
		t.Fatalf("expected stdout hello, got %q", result.Stdout)
	}
}

func TestScriptExecutorMarksFailuresRetryableWhenScriptExitsNonZero(t *testing.T) {
	executor := NewScriptExecutor()

	result, err := executor.Execute(context.Background(), &ScriptArtifact{
		Content: "#!/usr/bin/env bash\nset -euo pipefail\nexit 2\n",
	})
	if err == nil {
		t.Fatal("expected execution error")
	}

	if !result.Retryable {
		t.Fatal("expected non-policy script failure to be retryable")
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/tools -run 'TestScriptExecutorRunsGeneratedScript|TestScriptExecutorMarksFailuresRetryableWhenScriptExitsNonZero' -v`

Expected: FAIL with `undefined: NewScriptExecutor`

**Step 3: Write minimal implementation**

```go
type ExecutionResult struct {
	ExitCode  int
	Stdout    string
	Stderr    string
	Retryable bool
}

type ScriptExecutor struct{}

func NewScriptExecutor() *ScriptExecutor { return &ScriptExecutor{} }

func (e *ScriptExecutor) Execute(ctx context.Context, artifact *ScriptArtifact) (*ExecutionResult, error) {
	tmpFile, err := os.CreateTemp("", "executor-*.sh")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(artifact.Content); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "bash", tmpFile.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	result := &ExecutionResult{
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Retryable: err != nil,
	}
	return result, err
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/tools -run 'TestScriptExecutorRunsGeneratedScript|TestScriptExecutorMarksFailuresRetryableWhenScriptExitsNonZero' -v`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/tools/script_executor.go internal/tools/script_executor_test.go
git commit -m "feat: add script executor"
```

### Task 5: Wire planner-script-executor flow into registry and agent loop

**Files:**
- Modify: `internal/tools/registry.go`
- Create: `internal/tools/registry_test.go`
- Modify: `internal/core/agent_loop.go`
- Create: `internal/core/agent_loop_test.go`

**Step 1: Write the failing behavior specs**

```go
func TestRegistryBuildsExecutableArtifactFromPlanBackedRequest(t *testing.T) {
	reg := NewRegistry()

	result, err := reg.Execute(context.Background(), CommandRequest{
		Goal: "run tests",
		Plan: &ExecutionPlan{
			Goal: "run tests",
			Steps: []PlanStep{{Name: "test", Script: "echo hello"}},
		},
	})
	if err != nil {
		t.Fatalf("registry execute: %v", err)
	}

	if strings.TrimSpace(result.Stdout) != "hello" {
		t.Fatalf("expected stdout hello, got %q", result.Stdout)
	}
}

func TestAgentLoopUsesRegistryInsteadOfDirectCommandExecution(t *testing.T) {
	router := &Router{}
	reg := tools.NewRegistry()
	loop := NewAgentLoop(router, reg)

	if _, err := loop.Run(context.Background(), "echo hello"); err != nil {
		t.Fatalf("run: %v", err)
	}
}
```

**Step 2: Run the specs to verify the behaviors are not implemented yet**

Run: `go test ./internal/tools ./internal/core -run 'TestRegistryBuildsExecutableArtifactFromPlanBackedRequest|TestAgentLoopUsesRegistryInsteadOfDirectCommandExecution' -v`

Expected: FAIL with `undefined: (*Registry).Execute` or constructor/signature mismatch errors

**Step 3: Write minimal implementation**

```go
type Registry struct {
	policy   *CommandPolicy
	builder  *ScriptBuilder
	executor *ScriptExecutor
}

func NewRegistry() *Registry {
	return &Registry{
		policy:   NewCommandPolicy(),
		builder:  NewScriptBuilder(os.TempDir()),
		executor: NewScriptExecutor(),
	}
}

func (r *Registry) Execute(ctx context.Context, req CommandRequest) (*ExecutionResult, error) {
	if err := r.policy.Validate(req); err != nil {
		return nil, err
	}
	artifact, err := r.builder.Build(*req.Plan)
	if err != nil {
		return nil, err
	}
	return r.executor.Execute(ctx, artifact)
}
```

**Step 4: Run the specs to verify the behaviors now pass**

Run: `go test ./internal/tools ./internal/core -run 'TestRegistryBuildsExecutableArtifactFromPlanBackedRequest|TestAgentLoopUsesRegistryInsteadOfDirectCommandExecution' -v`

Expected: PASS

**Step 5: Run the full suite**

Run: `go test ./...`

Expected: all packages PASS

**Step 6: Commit**

```bash
git add internal/tools/registry.go internal/tools/registry_test.go internal/core/agent_loop.go internal/core/agent_loop_test.go
git commit -m "feat: wire scripted executor flow"
```
