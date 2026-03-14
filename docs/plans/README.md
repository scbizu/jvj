# Plans Directory Compatibility Note

Planning and design history now lives under:

`docs/devlog/{component}/`

This `docs/plans/` directory is kept only as a thin compatibility layer for migration-era records.

## What still lives here

- `2026-03-14-devlog-reorganization-design.md`
- `2026-03-14-devlog-reorganization-implementation.md`

These two files describe how the repository moved from date-prefixed plan files to component-scoped devlogs.

For active component work, use the matching files under `docs/devlog/` instead of creating new date-prefixed entries in `docs/plans/`.
