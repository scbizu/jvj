---
name: handoff
description: Standardize single-session tape handoff payloads and write them through the runtime handoff bridge.
---

# Built-in Handoff Skill

Use this skill when the agent needs to close the current phase and hand off the next actionable state.

## Expected payload

Produce a structured handoff payload with:

- `summary`
- `next_steps`
- `source_seqs`
- `owner`
- optional `phase_tag`
- optional `open_items`

## Runtime effect

The runtime bridge should translate the payload into a Tape handoff call so the latest anchor can advance without inventing a separate skill runtime.
