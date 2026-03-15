# Agent Runtime Entry Design

## Goal

Turn `cmd/agent-runtime run` from a startup placeholder into a real, first runnable runtime entry for the single-session architecture.

## Approved First Runtime Shape

The first runnable shape is a **single-session interactive stdin/stdout loop**:

1. `run` still requires an explicit config path.
2. Startup preloads built-in skills before the runtime begins serving input.
3. The runtime creates one in-memory session, one in-memory tape service, one router, and one scripted executor registry.
4. After startup, the process reads one line at a time from stdin.
5. Each non-empty line is sent through `core.AgentLoop.Run`.
6. The resulting assistant output is printed to stdout immediately.
7. The process exits on EOF or when the user enters `exit`.

## Why This Shape

- It is the smallest form that is still a **real runtime**, not just a bootstrap check.
- It matches the repository's current **single-session** assumption.
- It exercises the tape, session manager, router, and executor wiring end-to-end without forcing transport/server design decisions yet.

## Runtime Composition

`run` should build the runtime from the current in-repo components:

- `session.Manager` for the single active session lifecycle
- `tape.Service` backed by `tape.InMemoryStore`
- `core.Router` as the current routing skeleton
- `tools.Registry` for plan-backed command execution
- `core.AgentLoop` for per-turn orchestration

The entrypoint should open exactly one attached session at startup and close it on shutdown.

## I/O Contract

- Input source: `cmd.InOrStdin()`
- Output sink: `cmd.OutOrStdout()`
- Error sink: returned errors from `run`

Interactive behavior:

- blank lines are ignored
- `exit` ends the loop without error
- EOF ends the loop without error

## Error Handling

- Missing config path remains a hard error.
- Built-in skill preload failure aborts startup.
- Session open failure aborts startup.
- A turn failure aborts the current `run` invocation and returns the error instead of silently swallowing it.

## Scope Boundaries

This first step does **not** add:

- transport/server startup
- persistent tape storage
- config schema parsing beyond validating the path exists
- multi-session support
- a smarter router than the current skeleton

## Validation Notes

Behavior coverage should prove:

1. `run` still rejects missing config.
2. `run` fails when the config path does not exist.
3. `run` processes stdin lines through the real runtime loop.
4. `run` stops on `exit`.
5. `run` preloads built-in skills before serving input.
