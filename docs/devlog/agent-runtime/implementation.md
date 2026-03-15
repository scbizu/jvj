# Agent Runtime Entry Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire `cmd/agent-runtime run` into a real single-session interactive runtime loop backed by the existing tape, session, router, and scripted executor components.

**Architecture:** Keep the first runnable runtime intentionally small. `run` should validate the config path, preload built-in skills, construct an in-memory single-session runtime, then process stdin line-by-line through `core.AgentLoop.Run` until EOF or `exit`. This preserves the current single-session architecture while finally making the CLI entrypoint execute real runtime turns.

**Tech Stack:** Go, Cobra, bufio, Go testing

---

### Task 1: Add behavior coverage for the runtime loop entry

- Implemented in `cmd/agent-runtime/main.go` and `cmd/agent-runtime/main_test.go`
- Added runtime loop coverage plus deferred cleanup/context fixes

### Task 2: Enforce startup and shutdown behavior

- Implemented in `cmd/agent-runtime/main.go` and `cmd/agent-runtime/main_test.go`
- Added config existence validation and `exit` shutdown behavior

### Task 3: Run the CLI regression and full suite

- Verify `go test ./cmd/agent-runtime -v`
- Verify `go test ./...`
- Keep docs aligned with the accepted implementation

## Verification

- `go test ./cmd/agent-runtime -v` — PASS
- `go test ./...` — PASS

When done, report:
- What you created/updated
- Test results
- Files changed
- Commit SHA
- Any concerns
