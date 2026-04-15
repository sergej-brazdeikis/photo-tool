# Story 1.1: Local library foundation (config, layout, database)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **the app to use a clear library location and a reliable local database**,  
So that **my data has a stable home and upgrades apply safely**.

**Implements:** Architecture foundation; enables all FRs that need persistence. [Source: _bmad-output/planning-artifacts/epics.md — Story 1.1]

## Acceptance Criteria

1. **Given** no `PHOTO_TOOL_LIBRARY` env var, **when** the app resolves the library root, **then** it uses an absolute path under the OS user config area (`…/photo-tool/library` per architecture) **and** creating the library succeeds.
2. **Given** `PHOTO_TOOL_LIBRARY` set to a path, **when** the app starts, **then** that path is used (absolute) and standard subdirs exist (`.phototool`, `.trash`, `.cache/thumbnails`).
3. **Given** a fresh library, **when** the store opens, **then** SQLite is created under `.phototool/library.sqlite`, embedded migrations apply successfully, and `schema_meta.version` equals **`store.targetSchemaVersion`** in `internal/store/migrate.go` (currently **7**; must remain authoritative when new migrations are added).
4. **Given** the assets table exists, **when** two active rows would share the same `rel_path`, **then** the database rejects the second insert (**partial unique index** on `rel_path` where `deleted_at_unix IS NULL`).
5. **And** existing implementation in `internal/config`, `internal/paths`, `internal/filehash`, `internal/store` satisfies the above (**regression tests stay green**).

## Tasks / Subtasks

- [x] **Library root:** Confirm `config.ResolveLibraryRoot` — default `filepath.Join(UserConfigDir(), "photo-tool", "library")`, override via `PHOTO_TOOL_LIBRARY` (`config.EnvLibraryRoot`), always absolute (AC: 1, 2).
- [x] **Layout:** Confirm `config.EnsureLibraryLayout` creates `root`, `.phototool`, `.trash`, `.cache/thumbnails` (AC: 2).
- [x] **Store open:** Confirm `store.Open` — path `{libraryRoot}/.phototool/library.sqlite`, driver `modernc.org/sqlite`, `PRAGMA foreign_keys = ON`, `PRAGMA busy_timeout = 5000`, then `migrate` (AC: 3).
- [x] **Migrations:** Confirm embedded SQL under `internal/store/migrations/` through `007_*.sql`; `targetSchemaVersion` matches highest migration; `schema_meta` singleton row updated on each step (AC: 3).
- [x] **Uniqueness:** Confirm partial unique index `idx_assets_rel_path_active` in `001_initial.sql` and behavior covered by tests (AC: 4).
- [x] **Regression:** Run `go test ./internal/config/... ./internal/store/...` (and `go test ./...` if touching shared surfaces) — especially `TestOpen_migratesFreshLibrary`, `TestOpen_sqlitePragmas`, `TestOpen_partialUniqueRelPath`, `TestEnsureLibraryLayout_createsDirs`, `TestResolveLibraryRoot_envOverride`, `TestResolveLibraryRoot_envWhitespaceFallsBackToDefault` (AC: 5).
- [x] **Hardening (party dev 2/2):** `PHOTO_TOOL_LIBRARY` uses **ASCII-only** trim (Unicode spaces like NBSP are not stripped); `store.Open` requires non-empty root and existing `.phototool` dir with actionable error; tests `TestResolveLibraryRoot_envUnicodeSpaceNotTreatedAsASCIIWhitespace`, `TestOpen_requiresPhototoolDir`, `TestOpen_rejectsEmptyLibraryRoot`.

## Dev Notes

### Technical requirements

- **Env:** `PHOTO_TOOL_LIBRARY` is trimmed for leading/trailing **ASCII** whitespace only (`strings.Trim` cutset); empty, unset, or ASCII-whitespace-only falls back to the user config dir path. **Unicode** space characters (e.g. U+00A0 NBSP) are **not** trimmed—value is treated as a literal path (or `filepath.Abs` error). Non-empty values use `filepath.Abs` with errors wrapped naming the env var.
- **Store open:** Callers must run `config.EnsureLibraryLayout` first so `{libraryRoot}/.phototool` exists; `store.Open` fails fast with a clear error if it is missing (avoids opaque SQLite open/create errors).
- **DB file:** Single SQLite file per library; no in-memory prod path. Foreign keys enabled at open.
- **Schema version:** Epics originally stated `schema_meta.version` **1** after first migration; the codebase applies **forward-only** migrations **001–007**. Treat **AC3** as “version equals `targetSchemaVersion`,” not literal `1`, or tests and production will disagree with the spec.
- **`rel_path`:** Relative to library root; uniqueness enforced only for **non–soft-deleted** rows (`deleted_at_unix IS NULL`).

### Architecture compliance

- Library root, canonical layout hooks, SQLite + migrations under `internal/store/migrations/`, SHA-256 and paths elsewhere — [Source: _bmad-output/planning-artifacts/architecture.md — §3.2, §3.3, §3.12 decision order (1), §4.2–4.4 errors/logging, §5.1 `internal/` tree].
- Driver preference: **`modernc.org/sqlite`** (pure Go) per architecture §3.3 — matches `go.mod` and `store.Open`.

### Library / framework

- **Go** `1.25.4`, **Fyne** `v2.7.3`, **SQLite driver** `modernc.org/sqlite` `v1.48.2` — [Source: `go.mod`; architecture frontmatter].

### File structure (primary)

- `internal/config/library.go` — `ResolveLibraryRoot`, `EnsureLibraryLayout`, `EnvLibraryRoot`
- `internal/config/library_test.go` — env override, layout dirs
- `internal/store/open.go` — `Open`, pragmas, delegates to `migrate`
- `internal/store/migrate.go` — embedded migrations, `targetSchemaVersion`, `schema_meta` updates
- `internal/store/migrations/*.sql` — especially `001_initial.sql` (`assets`, `idx_assets_rel_path_active`, `schema_meta`)
- `internal/store/store_test.go` — fresh migrate version, partial unique `rel_path`, idempotent open
- `internal/paths/`, `internal/filehash/` — not owned by this story but cited in epic “existing implementation”; do not break callers (AC: 5)

### Testing requirements

- Table-driven / temp-dir integration tests with real SQLite file (existing pattern in `store_test.go`).
- Must verify: migrated version matches `targetSchemaVersion`; duplicate active `rel_path` fails; soft-deleted row allows same `rel_path` again.

### Previous story intelligence

- **N/A** — first story in Epic 1.

### Git / recent work patterns

- Recent commits emphasize ingest, CLI, collections, and tests on top of this foundation (`internal/store`, `internal/config`, `internal/ingest`). Changes here ripple to upload, scan, import, and review — keep migrations **forward-only** and bump `targetSchemaVersion` when adding `.sql` files.

### Project Structure Notes

- Single Go module `photo-tool`; boundaries per architecture §5.1 — config and store stay free of Fyne imports.
- DB path is always under library `.phototool/`; never place `library.sqlite` at repo root.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 1, Story 1.1]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.2, §3.3, §3.12, §4.x, §5.1]
- [Source: _bmad-output/planning-artifacts/PRD.md — persistence / local library expectations]
- [Source: internal/config/library.go — `EnvLibraryRoot`, resolution and layout]
- [Source: internal/store/open.go — DB path and pragmas]
- [Source: internal/store/migrate.go — `targetSchemaVersion`, migration loop]
- [Source: internal/store/migrations/001_initial.sql — `assets`, partial unique index]
- [Source: internal/store/store_test.go — `TestOpen_migratesFreshLibrary`, `TestOpen_partialUniqueRelPath`]

## Dev Agent Record

### Agent Model Used

Composer (dev-story workflow)

### Debug Log References

### Completion Notes List

- Verified AC1–5 against `internal/config` and `internal/store`: default library root now normalized with `filepath.Abs`; `PHOTO_TOOL_LIBRARY` override unchanged (Abs + env wrap on error).
- Exported `store.TargetSchemaVersion` and asserted migrated `schema_meta.version` against it in tests (avoids drift from hardcoded `7`).
- Added `TestResolveLibraryRoot_defaultIsAbsolute` (empty env → fallback path is absolute).
- Party dev1/2: `PHOTO_TOOL_LIBRARY` trimmed — whitespace-only falls back like unset; `TestOpen_sqlitePragmas` locks `foreign_keys` + `busy_timeout`; epic rollup AC3 wording aligned with `TargetSchemaVersion`.
- Party dev2/2: Challenged session-1 assumption that `strings.TrimSpace` matched “ASCII whitespace” spec — switched to ASCII-only trim + NBSP regression test; challenged “Open always follows EnsureLibraryLayout” — enforced `.phototool` preflight in `store.Open` with tests for missing dir and empty root.
- `go test ./internal/config/... ./internal/store/...`, `go test ./...`, and `go build .` all green.

### File List

- internal/config/library.go
- internal/config/library_test.go
- internal/store/migrate.go
- internal/store/store_test.go
- internal/store/open.go
- _bmad-output/planning-artifacts/epics.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/1-1-library-foundation.md

### Change Log

- 2026-04-14: Party mode dev 1/2 — env trim + pragma test + epics AC3 sync (`epics.md`).
- 2026-04-14: Party mode dev 2/2 — ASCII-only env trim vs `TrimSpace`; `Open` `.phototool` + empty-root guards + tests; story + sprint → **done**.
- 2026-04-14: Story 1.1 implementation review — minor hardening (default root Abs, `TargetSchemaVersion` export + test alignment); sprint `1-1-library-foundation` → review.
