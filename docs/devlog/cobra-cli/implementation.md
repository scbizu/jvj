# Cobra CLI Implementation

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate `cmd/agent-runtime` to Cobra, add a dedicated `run` command plus a placeholder `version` command, and preserve the required-config contract.

**Architecture:** Build a small Cobra command tree around `root -> run -> version`. The runtime mode stays behind `run`, which reuses the existing startup path. Config resolution should come from `--config` or the positional argument, with the flag taking precedence. Verification should focus on command-entry behavior so the CLI surface does not regress.

**Tech Stack:** Go, Cobra, Go testing

---

### Task 1: Introduce Cobra and refactor the command entry

**Files:**
- Modify: `go.mod`
- Modify: `cmd/agent-runtime/main.go`
- Modify: `cmd/agent-runtime/main_test.go`

**Step 1: Write the behavior test**

```go
func TestRunCmdConfigPathRequired(t *testing.T) {
    cmd := newRunCmd()
    cmd.SetArgs([]string{})
    err := cmd.Execute()
    if err == nil {
        t.Fatal("expected error when config path is missing")
    }
}
```

**Step 2: Run the focused test to confirm the gap**

Run: `go test ./cmd/agent-runtime -run TestRunCmdConfigPathRequired -v`
Expected: FAIL (`newRunCmd` is not defined yet)

**Step 3: Implement the `run` command entry**

```go
func newRunCmd() *cobra.Command {
    var configPath string
    cmd := &cobra.Command{
        Use: "run [config-path]",
        RunE: func(cmd *cobra.Command, args []string) error {
            return run(args, configPath)
        },
    }
    cmd.Flags().StringVar(&configPath, "config", "", "config path")
    return cmd
}
```

**Step 4: Re-run the focused test**

Run: `go test ./cmd/agent-runtime -run TestRunCmdConfigPathRequired -v`
Expected: PASS

**Step 5: Commit**

```bash
git add go.mod cmd/agent-runtime/main.go cmd/agent-runtime/main_test.go
git commit -m "feat: migrate agent-runtime cmd to cobra"
```

### Task 2: Add the `version` placeholder command and verify runtime behavior

**Files:**
- Modify: `cmd/agent-runtime/main.go`
- Modify: `cmd/agent-runtime/main_test.go`

**Step 1: Write the behavior test**

```go
func TestVersionCmdPrintsPlaceholder(t *testing.T) {
    cmd := newVersionCmd()
    b := &bytes.Buffer{}
    cmd.SetOut(b)
    if err := cmd.Execute(); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if !strings.Contains(b.String(), "dev") {
        t.Fatal("expected placeholder version")
    }
}
```

**Step 2: Run the focused test to confirm the missing command**

Run: `go test ./cmd/agent-runtime -run TestVersionCmdPrintsPlaceholder -v`
Expected: FAIL (`newVersionCmd` is not defined yet)

**Step 3: Implement the placeholder command**

```go
func newVersionCmd() *cobra.Command {
    return &cobra.Command{
        Use: "version",
        RunE: func(cmd *cobra.Command, args []string) error {
            _, err := fmt.Fprintln(cmd.OutOrStdout(), "dev")
            return err
        },
    }
}
```

**Step 4: Run the CLI regression checks**

Run: `go test ./cmd/agent-runtime -run TestVersionCmdPrintsPlaceholder -v && go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/agent-runtime/main.go cmd/agent-runtime/main_test.go
git commit -m "feat: add cobra version command placeholder"
```
