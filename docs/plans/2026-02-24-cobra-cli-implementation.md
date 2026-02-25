# Cobra CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `cmd/agent-runtime` 入口迁移到 Cobra，支持 `run` 子命令与 `version` 占位命令，并保持 config 必填语义。

**Architecture:** 在 `cmd/agent-runtime/main.go` 内引入 root/run/version 三层命令结构。`run` 通过 `RunE` 复用现有执行逻辑，配置来源为 `--config` 或位置参数（flag 优先）。测试聚焦 run 命令参数验证，保证行为不回归。

**Tech Stack:** Go, Cobra, Go testing

---

### Task 1: 引入 Cobra 并重构命令入口

**Files:**
- Modify: `go.mod`
- Modify: `cmd/agent-runtime/main.go`
- Modify: `cmd/agent-runtime/main_test.go`

**Step 1: Write the failing test**

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

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/agent-runtime -run TestRunCmdConfigPathRequired -v`
Expected: FAIL（`newRunCmd` 未定义）

**Step 3: Write minimal implementation**

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

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/agent-runtime -run TestRunCmdConfigPathRequired -v`
Expected: PASS

**Step 5: Commit**

```bash
git add go.mod cmd/agent-runtime/main.go cmd/agent-runtime/main_test.go
git commit -m "feat: migrate agent-runtime cmd to cobra"
```

### Task 2: 增加 version 占位命令并回归验证

**Files:**
- Modify: `cmd/agent-runtime/main.go`
- Modify: `cmd/agent-runtime/main_test.go`

**Step 1: Write the failing test**

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

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/agent-runtime -run TestVersionCmdPrintsPlaceholder -v`
Expected: FAIL（`newVersionCmd` 未定义）

**Step 3: Write minimal implementation**

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

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/agent-runtime -run TestVersionCmdPrintsPlaceholder -v && go test ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/agent-runtime/main.go cmd/agent-runtime/main_test.go
git commit -m "feat: add cobra version command placeholder"
```
