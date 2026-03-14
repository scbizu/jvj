# Cobra CLI Design

## Goal

Reshape `cmd/agent-runtime` into a Cobra-based command entry so the runtime starts from a consistent CLI surface while preserving the current `config path is required` behavior.

## Command Layout

1. `main.go` should delegate startup through `Execute()`.
2. The root command should host the runtime-facing subcommands:
   - `run` for normal runtime execution
   - `version` as a placeholder metadata command
3. The `run` command should accept config input from:
   - positional argument: `run <config-path>`
   - flag: `--config`
   - if both are present, `--config` wins

## Runtime Mode Semantics

- `run` remains the command entry for loading config and starting the agent runtime.
- The CLI should not invent implicit defaults for config resolution.
- The `version` command can return a fixed placeholder value until release metadata is wired in.

## Error Handling

- If neither a positional config path nor `--config` is provided, return `config path is required`.
- Do not silently recover from missing config input.

## Validation Notes

- Add behavior coverage around `newRunCmd()`:
  - missing config should fail
  - `--config` should allow execution to proceed
