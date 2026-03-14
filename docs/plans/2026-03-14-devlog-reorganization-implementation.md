# Devlog Reorganization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reorganize `docs/plans/` into `docs/devlog/{component}` with component-based naming, domain-aligned wording, and updated internal references.

**Architecture:** Migrate the existing planning documents into per-component devlog directories, rewriting titles and surrounding prose so each component reads as a coherent domain narrative instead of a dated snapshot. Update all repository references that still point at `docs/plans/`, then leave behind a thin `docs/plans/README.md` compatibility note rather than keeping duplicated redirect files.

**Tech Stack:** Markdown documentation, repository-local skill prompt files under `.agents/skills/`, `rg`, `git`, and `go test`

---

### Task 1: Create the devlog structure and migrate `cobra-cli` + `connectrpc`

**Files:**
- Create: `docs/devlog/cobra-cli/`
- Create: `docs/devlog/connectrpc/`
- Move + Rewrite: `docs/plans/2026-02-24-cobra-cli-design.md` -> `docs/devlog/cobra-cli/design.md`
- Move + Rewrite: `docs/plans/2026-02-24-cobra-cli-implementation.md` -> `docs/devlog/cobra-cli/implementation.md`
- Move + Rewrite: `docs/plans/2026-02-24-connectrpc-design.md` -> `docs/devlog/connectrpc/design.md`
- Move + Rewrite: `docs/plans/2026-02-24-connectrpc-architecture-implementation.md` -> `docs/devlog/connectrpc/architecture-implementation.md`

**Step 1: Create the target component directories**

Run: `mkdir -p docs/devlog/cobra-cli docs/devlog/connectrpc`
Expected: directories exist with no output

**Step 2: Rewrite and move the `cobra-cli` design doc**

Update the title, summary, and section wording so the file reads in `cobra-cli` domain language (`command entry`, `config`, `runtime mode`), then save it to:

`docs/devlog/cobra-cli/design.md`

**Step 3: Rewrite and move the `cobra-cli` implementation doc**

Update the intro and task framing so it reads as a `cobra-cli` implementation devlog, then save it to:

`docs/devlog/cobra-cli/implementation.md`

**Step 4: Rewrite and move the `connectrpc` design doc**

Update terminology toward `transport layer`, `protocol`, and `service boundary`, then save it to:

`docs/devlog/connectrpc/design.md`

**Step 5: Rewrite and move the `connectrpc` architecture implementation doc**

Keep the implementation intent but align the wording to `connectrpc` domain language, then save it to:

`docs/devlog/connectrpc/architecture-implementation.md`

**Step 6: Verify the new files exist and old files are gone**

Run: `test -f docs/devlog/cobra-cli/design.md && test -f docs/devlog/cobra-cli/implementation.md && test -f docs/devlog/connectrpc/design.md && test -f docs/devlog/connectrpc/architecture-implementation.md && test ! -f docs/plans/2026-02-24-cobra-cli-design.md && test ! -f docs/plans/2026-02-24-connectrpc-design.md`
Expected: success with exit code 0

**Step 7: Commit**

```bash
git add docs/devlog/cobra-cli docs/devlog/connectrpc
git commit -m "docs: reorganize cobra and connectrpc devlogs"
```

### Task 2: Create `message-bus` devlog and normalize message bus language

**Files:**
- Create: `docs/devlog/message-bus/`
- Move + Rewrite: `docs/plans/2026-02-24-message-bus-bot-design.md` -> `docs/devlog/message-bus/bot-design.md`
- Move + Rewrite: `docs/plans/2026-02-24-message-bus-refinement-design.md` -> `docs/devlog/message-bus/refinement-design.md`
- Move + Rewrite: `docs/plans/2026-02-24-message-bus-runtime-implementation.md` -> `docs/devlog/message-bus/runtime-implementation.md`
- Move + Rewrite: `docs/plans/2026-02-24-message-bus-discord-telegram-implementation.md` -> `docs/devlog/message-bus/discord-telegram-implementation.md`

**Step 1: Create the message bus directory**

Run: `mkdir -p docs/devlog/message-bus`
Expected: directory exists with no output

**Step 2: Rewrite and move the bot design doc**

Rewrite the intro and section wording so it consistently uses `ingress`, `router`, and `platform integration` language, then save to:

`docs/devlog/message-bus/bot-design.md`

**Step 3: Rewrite and move the refinement design doc**

Align the file to `message-bus` domain terms and save it to:

`docs/devlog/message-bus/refinement-design.md`

**Step 4: Rewrite and move the runtime implementation doc**

Adjust the narrative so it reads as the runtime-focused message bus implementation record, then save it to:

`docs/devlog/message-bus/runtime-implementation.md`

**Step 5: Rewrite and move the Discord/Telegram implementation doc**

Use platform adapter terminology consistently and save it to:

`docs/devlog/message-bus/discord-telegram-implementation.md`

**Step 6: Verify the message bus migration**

Run: `test -f docs/devlog/message-bus/bot-design.md && test -f docs/devlog/message-bus/refinement-design.md && test -f docs/devlog/message-bus/runtime-implementation.md && test -f docs/devlog/message-bus/discord-telegram-implementation.md`
Expected: success with exit code 0

**Step 7: Commit**

```bash
git add docs/devlog/message-bus
git commit -m "docs: reorganize message bus devlogs"
```

### Task 3: Migrate `tape-service`, `executor`, and `agent-skills`

**Files:**
- Create: `docs/devlog/tape-service/`
- Create: `docs/devlog/executor/`
- Create: `docs/devlog/agent-skills/`
- Move + Rewrite: `docs/plans/2026-03-14-tape-service-design-refinement.md` -> `docs/devlog/tape-service/design-refinement.md`
- Move + Rewrite: `docs/plans/2026-03-14-tape-service-implementation.md` -> `docs/devlog/tape-service/implementation.md`
- Move + Rewrite: `docs/plans/2026-03-14-executor-script-planner-design.md` -> `docs/devlog/executor/script-planner-design.md`
- Move + Rewrite: `docs/plans/2026-03-14-executor-script-planner-implementation.md` -> `docs/devlog/executor/script-planner-implementation.md`
- Move + Rewrite: `docs/plans/2026-03-14-agent-skills-handoff-design.md` -> `docs/devlog/agent-skills/handoff-design.md`
- Move + Rewrite: `docs/plans/2026-03-14-agent-skills-handoff-implementation.md` -> `docs/devlog/agent-skills/handoff-implementation.md`

**Step 1: Create the runtime-centric component directories**

Run: `mkdir -p docs/devlog/tape-service docs/devlog/executor docs/devlog/agent-skills`
Expected: directories exist with no output

**Step 2: Rewrite and move the tape service docs**

Normalize the wording around `facts`, `anchor`, `handoff`, and `view`, then save the two docs to:

- `docs/devlog/tape-service/design-refinement.md`
- `docs/devlog/tape-service/implementation.md`

**Step 3: Rewrite and move the executor docs**

Normalize the wording around `planner`, `script builder`, and `executor`, then save the two docs to:

- `docs/devlog/executor/script-planner-design.md`
- `docs/devlog/executor/script-planner-implementation.md`

**Step 4: Rewrite and move the agent skills docs**

Normalize the wording around `agent native skills`, `built-in handoff`, `skill bundle`, and `preload`, then save the two docs to:

- `docs/devlog/agent-skills/handoff-design.md`
- `docs/devlog/agent-skills/handoff-implementation.md`

**Step 5: Verify the runtime-centric migrations**

Run: `test -f docs/devlog/tape-service/design-refinement.md && test -f docs/devlog/executor/script-planner-design.md && test -f docs/devlog/agent-skills/handoff-design.md`
Expected: success with exit code 0

**Step 6: Commit**

```bash
git add docs/devlog/tape-service docs/devlog/executor docs/devlog/agent-skills
git commit -m "docs: reorganize runtime devlogs"
```

### Task 4: Update repository references from `docs/plans` to `docs/devlog`

**Files:**
- Modify: `docs/architecture.md`
- Modify: `.agents/skills/brainstorming/SKILL.md`
- Modify: `.agents/skills/writing-plans/SKILL.md`
- Modify: `.agents/skills/subagent-driven-development/SKILL.md`
- Modify: moved docs under `docs/devlog/**` that still reference `docs/plans/...`

**Step 1: Update `docs/architecture.md` references**

Replace any direct `docs/plans/...` links so they point at the new `docs/devlog/{component}/...` paths.

**Step 2: Update brainstorming skill prompts**

Replace the default design-doc output path in:

`.agents/skills/brainstorming/SKILL.md`

so it reflects the new `docs/devlog/{component}` layout.

**Step 3: Update writing-plans skill prompts**

Replace the default implementation-plan output path in:

`.agents/skills/writing-plans/SKILL.md`

so it reflects the new `docs/devlog/{component}` layout.

**Step 4: Update subagent-driven-development references**

Replace any example plan paths still pointing at `docs/plans/...`.

**Step 5: Verify there are no stale hardcoded paths**

Run: `rg -n "docs/plans/" docs .agents`
Expected: only allowed compatibility references remain

**Step 6: Commit**

```bash
git add docs/architecture.md .agents/skills/brainstorming/SKILL.md .agents/skills/writing-plans/SKILL.md .agents/skills/subagent-driven-development/SKILL.md docs/devlog
git commit -m "docs: repoint devlog references"
```

### Task 5: Add the compatibility note and run final validation

**Files:**
- Create: `docs/plans/README.md`
- Remove: remaining migrated Markdown files from `docs/plans/`

**Step 1: Add the compatibility README**

Create `docs/plans/README.md` explaining that planning and design history now lives under:

`docs/devlog/{component}/`

**Step 2: Verify the old `docs/plans/` content is fully migrated**

Run: `find docs/plans -maxdepth 1 -type f -name '*.md' | sort`
Expected: only `README.md` and intentionally retained migration docs remain

**Step 3: Run formatting and stale-reference checks**

Run: `git diff --check && rg -n "docs/plans/" docs .agents`
Expected: no whitespace errors and only allowed compatibility mentions

**Step 4: Run the repository test suite**

Run: `go test ./...`
Expected: all packages PASS

**Step 5: Commit**

```bash
git add docs/plans/README.md docs/devlog docs/architecture.md .agents/skills
git commit -m "docs: migrate plans into component devlogs"
```
