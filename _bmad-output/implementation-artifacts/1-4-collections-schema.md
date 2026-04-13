# Story 1.4: Collections schema and batch assignment API

Status: in-progress

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **collections to exist in the database before upload confirmation flows**,  
So that **I can attach uploads to albums safely**.

**Implements:** FR-18 (persistence layer for collections), FR-20; prerequisite for FR-04–FR-06.

## Acceptance Criteria

1. **Given** migrations run, **when** the store is opened, **then** `collections` and `asset_collections` tables exist with fields needed for FR-18 (name required, display_date optional with default rule).
2. **Given** a set of asset IDs and a collection ID, **when** linking API runs, **then** relations are created idempotently or per documented rules.
3. **Given** a collection delete request, **when** executed, **then** all asset–collection rows for that collection are removed then the collection row is deleted (FR-20).

## Tasks / Subtasks

- [x] **Migration 002 + migrator upgrade** (AC: 1)
  - [x] Add `internal/store/migrations/002_collections.sql` defining `collections`, `asset_collections`, FKs, and indexes (see Dev Notes for suggested shape).
  - [x] Refactor `internal/store/migrate.go`: today it only runs `001_initial.sql` when `v < 1` and returns early when `v >= 1`, which **blocks** future migrations. Implement a **versioned chain** (apply001 if needed, then 002 if `version < 2`, etc.) and set `schema_meta.version` to **2** after 002 succeeds. Fresh DBs must end at version 2.
  - [x] Update `internal/store/store_test.go` (`TestOpen_migratesFreshLibrary` and any version assertions) so expected `schema_meta.version` is **2** after this story.
- [x] **Store API: collections + linking + delete** (AC: 1–3)
  - [x] `CreateCollection` (or equivalent): `name` non-empty; if `display_date` omitted, persist default = **collection creation calendar date** (PRD FR-18 — same semantics as Story 2.9 UI later; enforce in Go, not silent NULL unless product explicitly wants “unknown”).
  - [x] `LinkAssetsToCollection(db, collectionID, assetIDs []int64)`: batch link; **document idempotency** (e.g. `INSERT OR IGNORE` with `UNIQUE(asset_id, collection_id)`, or skip-if-exists per pair). Invalid `asset_id` or `collection_id` should error clearly.
  - [x] `DeleteCollection(db, collectionID)`: remove all `asset_collections` rows for that collection, then delete the `collections` row (FR-20). May use explicit deletes or `ON DELETE CASCADE` from `asset_collections` → `collections` if documented — behavior must match AC3.
- [x] **Tests** (AC: 1–3)
  - [x] Open store on temp library → assert tables exist and `schema_meta.version == 2`.
  - [x] Insert two assets (reuse patterns from `store_test.go` / ingest fixtures), create collection, link both, assert two junction rows; **second link call** must not duplicate (idempotency).
  - [x] Delete collection → junction rows gone, collection gone, **asset rows unchanged**.

### Review Findings

Code review (BMAD code-review, 2026-04-13). Headless run: findings recorded as action items; no code changes applied during review.

- [ ] [Review][Patch] Replace `INSERT OR IGNORE` for junction inserts — with SQLite conflict algorithm `IGNORE`, foreign-key violations are skipped without error, so invalid `asset_id` / `collection_id` may not surface as errors (violates AC2 and the “FK failures surface” completion note). Prefer `INSERT ... ON CONFLICT(asset_id, collection_id) DO NOTHING` (or plain `INSERT` with constraint handling) so duplicates stay idempotent while bad FKs still fail. [`internal/store/collections.go:51`]
- [ ] [Review][Patch] After changing link semantics, run `go test ./internal/store/...` and adjust tests or completion notes if error/idempotency behavior changes. [`internal/store/store_test.go`]

## Dev Notes

### Technical requirements

- **Prerequisite:** Story 1.3 — `assets` rows and `InsertAsset` exist; use real `asset` IDs in link/delete tests.
- **Scope:** Persistence + `internal/store` API only. **No** Fyne upload UI (Story 1.5), **no** full CRUD/rename flows in UI (Story 2.9) — but schema and defaults must satisfy FR-18 so later stories only add UX.
- **`display_date` representation:** Architecture lists `display_date` without SQL type. Pick one approach and document it in code: e.g. `TEXT` ISO `YYYY-MM-DD` for calendar semantics, or `INTEGER` Unix midnight UTC — **must** align with PRD (“calendar date”, not capture time). `created_at_unix` should stay consistent with `assets.created_at_unix` (integer Unix).
- **Foreign keys:** `open.go` already sets `PRAGMA foreign_keys = ON`; use FK constraints on `asset_collections` to `assets(id)` and `collections(id)` where helpful for integrity.
- **Indexes:** Plan for filter/list queries — e.g. index on `asset_collections(collection_id)`, `asset_collections(asset_id)` (architecture §3.3 mentions collection membership for filters).

### Architecture compliance

- SQLite + embedded migrations under `internal/store/migrations/`, forward-only (architecture §3.3).
- Repository-style functions in `internal/store`; **no SQL in future Fyne** — same functions usable from CLI/GUI later (architecture §3.8 state/repository note).
- Errors wrapped with `%w`; use `log/slog` at call sites when this package gains logging in later stories (architecture §4.2–4.3).

### Library / module

- Continue **`modernc.org/sqlite`** driver; no new DB driver for this story.

### File structure (touch / extend)

- `internal/store/migrations/002_collections.sql` — **new**
- `internal/store/migrate.go` — **extend** (multi-version migration runner)
- `internal/store/open.go` — unchanged unless migration entrypoint changes signature
- `internal/store/collections.go` (or split `assets.go` / naming consistent with repo) — **new** methods
- `internal/store/store_test.go` — extend version + new collection tests

### Testing requirements

- Follow existing patterns: `t.TempDir()`, `config.EnsureLibraryLayout`, `Open(root)`, `t.Cleanup` closing DB.
- Table-driven tests optional; minimum is explicit coverage for AC1–AC3.

### Previous story intelligence (1.3)

- `InsertAsset` / `FindAssetByContentHash` live in `internal/store/assets.go`; migration **001** defines `assets` — reuse for fixture data.
- Story 1.3 review called out **SQLite error detection** (`errors.As` / constraint codes) for ingest — new store code should prefer typed/driver-aware error handling over string matching on `err.Error()` where uniqueness matters.
- `schema_meta.version` is currently expected to be **1** in tests; this story **must** bump to **2** and fix assertions.

### Scope boundaries

- **Out of scope:** Upload confirmation flow (1.5); rename collection (2.9); multi-collection UI (2.9); tags (2.5). Batch “link many assets” is **this** story’s API surface for later callers.

### Project structure notes

- Aligns with architecture §3.3 (`collections`, `asset_collections` many-to-many).
- Epic validation: “DB/migrations only when story needs them” — collections migration belongs here, not in 1.5.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Story 1.4, Epic 1]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.3 Data architecture (collections, asset_collections, indexes)]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-18, FR-20, FR-04–FR-06 (downstream consumers)]
- [Source: _bmad-output/implementation-artifacts/1-3-core-ingest.md — store/asset patterns, schema version 1 baseline, out-of-scope boundary]

## Dev Agent Record

### Agent Model Used

Composer (Cursor agent)

### Debug Log References

### Completion Notes List

- Implemented forward-only migration chain through `targetSchemaVersion` (2): apply `001_initial.sql` when `v < 1`, then `002_collections.sql` when `v < 2`, updating `schema_meta.version` after each step via existing upsert pattern.
- Collections schema: `display_date` as `TEXT` ISO `YYYY-MM-DD` (documented on `CreateCollection`); `created_at_unix` on `collections` and `asset_collections`; junction `PRIMARY KEY (asset_id, collection_id)` plus `ON DELETE CASCADE` on `collection_id` for FR-20.
- `CreateCollection` trims name, rejects empty; default display date = local calendar date when `displayDateISO` is empty.
- `LinkAssetsToCollection` uses `INSERT OR IGNORE` for idempotency; FK failures surface as wrapped errors. `DeleteCollection` deletes the collection row (CASCADE clears junction); errors if id missing.
- Tests: `TestOpen_migratesFreshLibrary` expects v2 and `collections` / `asset_collections` tables; `TestCollections_createLinkIdempotentDelete` covers AC2–AC3; extra tests for empty name and invalid FK links.

### File List

- internal/store/migrations/002_collections.sql
- internal/store/migrate.go
- internal/store/collections.go
- internal/store/store_test.go
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/1-4-collections-schema.md

### Change Log

- 2026-04-13: Story 1.4 — collections schema, versioned migrations, store API (`CreateCollection`, `LinkAssetsToCollection`, `DeleteCollection`), tests; sprint status `1-4-collections-schema` → review.

---

**Story context:** Ultimate context engine analysis completed — comprehensive developer guide created (BMAD create-story workflow).
