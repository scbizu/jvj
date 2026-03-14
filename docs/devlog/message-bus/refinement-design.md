# Message Bus Refinement Design

## Background

With Message Bus ingress and Discord/Telegram platform integration chosen as the mainline architecture, the next design pass needs to tighten two areas:

1. command-execution safety
2. dependency-install policy

## Goal and Scope

- Promote Message Bus from a conceptual entry point into a first-class module design (`3.2`).
- Define event types, router states, error layers, and audit expectations.
- Stay at architecture level instead of drifting into implementation-heavy pseudocode.

## Design Decisions

### 1. Module layout

- Add `3.2 Message Bus Ingress`.
- Shift the old `3.2~3.7` modules to `3.3~3.8`.
- Keep platform ingress responsibilities such as normalization, validation, and routing separate from the runtime core.

### 2. Command policy

- Reject raw shell-string execution as a primary path.
- Use the flow: `BusEvent(command)` -> `Router.Parse` -> `ToolRegistry.SchemaValidate` -> `Sandbox Executor(argv)`.
- Enforce fixed cwd, non-root execution, timeout, resource quotas, output caps, and required audit fields.

### 3. Dependency install policy

- Route `deps.install` through a `system` event class instead of the normal user-command path.
- Allow standard ecosystem flows such as go/npm/pip.
- Reject high-risk patterns such as:
  - `curl|bash`
  - `sudo`
  - `npm -g`
  - direct execution of remote script URLs
- Installation failures must be reported explicitly instead of silently downgraded.

### 4. Event and error model

- Event types: `message | command | callback | system`
- Router state flow: `received -> validated -> routed -> executing -> replied/failed`
- Error layers:
  1. Platform Error
  2. Policy Reject
  3. Execution Error
- Every error must carry a `correlation_id` and be written into Tape.

## Documentation Impact

- `docs/architecture.md`
  - add and reorder the module structure in chapter 3
  - strengthen section `6.2` around Message Bus protocol semantics
  - extend the TOML config section with command/deps policy fields

## Acceptance Criteria

- The docs contain a standalone `3.2 Message Bus Ingress` module.
- The docs explicitly ban direct eval and define a blacklist-first dependency install policy.
- Section `6.2` includes event types, router states, and layered error handling.
