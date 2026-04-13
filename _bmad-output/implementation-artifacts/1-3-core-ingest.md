# Story 1.3: Core ingest — copy into canonical storage and register asset

Status: in-progress

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **new files copied into Year/Month/Day with unique names and DB registration**,  
So that **the library reflects what is on disk**.

**Implements:** FR-01, FR-02, FR-03; NFR-02, NFR-03, NFR-04.

## Acceptance Criteria

1. **Given** a source file not yet in the library, **when** ingest runs, **then** the file is copied under `{library}/{YYYY}/{MM}/{DD}/` using capture time from Story 1.2 and named per architecture (`SuggestedFilename` + hash prefix), and an **assets** row is inserted with `content_hash`, `rel_path`, `capture_time_unix`, `created_at_unix`.
2. **Given** a file whose **size + hash** matches an existing asset, **when** ingest runs, **then** no duplicate copy is made and the outcome is counted as **skipped_duplicate** (FR-03).
3. **Given** ingest processes multiple files, **when** it finishes, **then** it returns an **`OperationSummary`** (or equivalent) with stable fields: **added**, **skipped_duplicate**, **updated**, **failed** (NFR-04).
4. **And** ingest uses streaming/chunked file read for hashing where appropriate (supports NFR-02 for large batches).

## Tasks / Subtasks

- [x] **Map implementation to AC:** Walk `internal/ingest.Ingest` / `ingestOne` against AC1–4; fix any behavioral gaps found during dev-story (brownfield: pipeline already present). (AC: 1–4)
 - [x] Confirm flow: `exifmeta.ReadCapture` → open source → `filehash.ReaderHex` → `store.FindAssetByContentHash` → seek → `copyToFile` → `filepath.Rel` + `ToSlash` → `store.InsertAsset` → `Added++`. (AC: 1, 4)
  - [x] Confirm duplicate path: early return `SkippedDuplicate++` when hash exists; late `UNIQUE` on `content_hash` removes copied file and counts `SkippedDuplicate` (race-safe). (AC: 2)
  - [x] Confirm insert failure removes destination file and increments `Failed` (unless late-duplicate case). (AC: 1, 3)
- [x] **`OperationSummary`:** Use `internal/domain.OperationSummary` with stable JSON `snake_case` tags; **`Updated` stays 0** for this story unless a metadata-only update path is added—already documented on the struct. (AC: 3)
- [x] **Dedup semantics:** Document in package/docs that **full-byte SHA-256** (`content_hash`) is the persisted dedup key; matching digest implies matching size, satisfying epic “size + hash” intent without a separate `file_size` column in `assets` (unless product later requires explicit size verification). (AC: 2, NFR-03)
- [x] **Integration tests:** Add `internal/ingest/ingest_test.go`: temp library root, open store + migrations, `ingest.Ingest` same path twice → one row, second pass `SkippedDuplicate==1`, first pass `Added==1`, summary fields consistent; cover multi-file batch at least superficially. (AC: 2, 3) — *gap called out in epic retrospective: no co-located tests yet.*
- [x] **Regression:** `go test ./...`; no Fyne imports under `internal/ingest` (architecture §5.2). Keep `internal/config`, `internal/paths`, `internal/filehash`, `internal/store`, `internal/exifmeta` tests green.

### Review Findings

- [ ] [Review][Patch] Detect late duplicate via SQLite constraint errors, not English substring matching on `err.Error()` — `isUniqueContentHash` matches `"UNIQUE constraint failed"` and index names; this breaks if the driver/localization/message format changes. Prefer `errors.As` into the `modernc.org/sqlite` error type and check constraint/extended result codes (e.g. `SQLITE_CONSTRAINT_UNIQUE`). [internal/ingest/ingest.go:136-145]

- [ ] [Review][Patch] Integration tests only exercise `ReadCapture` mtime fallback (no EXIF in `writeJPEGGray`); AC1 calls out capture time from Story 1.2 including the EXIF path. Add at least one ingest test with a tiny JPEG carrying `DateTimeOriginal` so `CanonicalDayDir` / `rel_path` reflect EXIF-derived UTC, not only `Chtimes`. [internal/ingest/ingest_test.go]

- [x] [Review][Defer] `copyToFile` does not `Sync` the destination — acceptable MVP; revisit if NFRs require crash-safe media writes. [internal/ingest/ingest.go:120-133] — deferred, pre-existing operational concern

- [x] [Review][Defer] Theoretical collision: same UTC second + identical first 12 hex chars of two distinct SHA-256 digests yields the same `SuggestedFilename` / `destAbs`; `O_TRUNC` could clobber an existing asset on disk before DB rejects the insert. Probability is negligible for non-adversarial use; mitigating cleanly conflicts with idempotent retry after a crash between copy and insert. Document as accepted architecture risk or lengthen prefix in a future story if libraries scale into collision regimes. [internal/paths/canonical.go SuggestedFilename + internal/ingest/ingest.go copyToFile] — deferred, architecture/product tradeoff

## Dev Notes

### Technical requirements

- **Prerequisite:** Story 1.2 (`exifmeta.ReadCapture`, `filehash.ReaderHex` / `SumHex`, UTC alignment with `paths.CanonicalDayDir`) must be available; ingest wiring is **this** story.
- **Brownfield:** `internal/ingest/ingest.go` already orchestrates capture → hash → dedup → copy → DB insert with documented ordering (hash before destination, cleanup on insert failure). Dev-story focus may be **tests, edge cases, and AC sign-off** rather than greenfield implementation.
- **Schema:** `internal/store/migrations/001_initial.sql` — `assets` includes `content_hash`, `rel_path`, `capture_time_unix`, `created_at_unix`; partial unique index on `rel_path` for active rows; unique `content_hash`.
- **Naming on disk:** `paths.SuggestedFilename` — UTC `20060102-150405`, first **12** hex chars of full lowercase hash, lowercased extension; DB stores **full** SHA-256 hex.
- **Streaming:** Single `*os.File`: hash with `ReaderHex`, then `Seek` to start for `io.Copy` to destination (NFR-02-friendly for per-file work; unbounded batch walks stay Story 1.6).
- **Logging / errors:** `log/slog` for failures; wrap store/path errors with `%w` where applicable (architecture §4.2–4.3).

### Architecture compliance

- Boundaries: **ingest → store, exifmeta, paths, filehash, domain**; no SQL in Fyne; no Fyne in ingest (architecture §5.2, §5.1).
- One dedup/summary path for later CLI/GUI: `domain.OperationSummary` + shared `ingest` entry (architecture §3.2, §3.9, §4.5).
- Stack versions (reference): Go **1.25.4**, Fyne **v2.7.3** per architecture frontmatter — ingest remains UI-free.

### Library / module

- **EXIF:** `github.com/dsoprea/go-exif/v3` is used via `internal/exifmeta`; do not swap without architecture change.
- **SQLite:** `modernc.org/sqlite` driver via `internal/store` (architecture §3.3).

### File structure (touch / extend)

- `internal/ingest/ingest.go` — primary orchestration (existing).
- `internal/ingest/ingest_test.go` — **to add** (integration-style).
- `internal/domain/summary.go` — `OperationSummary` (existing).
- `internal/store/assets.go` — `FindAssetByContentHash`, `InsertAsset` (existing).
- `internal/paths/canonical.go` — `CanonicalDayDir`, `SuggestedFilename`.
- `internal/exifmeta/capture.go`, `internal/filehash/filehash.go` — consumers only.

### Testing requirements

- Table-driven + tempdir SQLite patterns per architecture §4.4.
- Minimum: duplicate ingest → `skipped_duplicate` + single row; happy path → file on disk under canonical day dir + row populated.

### Previous story intelligence (1.2)

- Capture: EXIF datetime as **local wall** then **`.UTC()`** for storage; must match `CanonicalDayDir` / `SuggestedFilename` (UTC).
- `ReaderHex` parity with `SumHex` after seek is tested in 1.2; ingest relies on seek-then-copy after hash.
- Optional review item: wrap `os.Open` errors in `SumHex` for `%w` consistency — ingest uses `ReaderHex` on opened file, not `SumHex` for primary path.
- `ReadCapture` may swallow underlying EXIF errors when falling back to mtime; ingest logs `ReadCapture` failure as `Failed` when error non-nil.

### Scope boundaries

- **Out of scope:** Collections / upload UI / operation receipt UI (Stories 1.4–1.5); CLI `scan`/`import` (1.6–1.7). This story is the **shared core** those call later.
- **Receipts:** UX-DR6 uses the same summary shape; UI wiring is not required here.

### Project Structure Notes

- Matches architecture §3.12 step 3 (ingest pipeline + OperationSummary) and §5.3 (FR-01–FR-03 → ingest + store + exifmeta).
- `main.go` may not invoke `ingest` yet; that is acceptable until 1.5/1.6 wires callers.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 1, Story 1.3]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.2 Library storage and deduplication, §3.3 data architecture (assets), §3.9 OperationSummary, §3.12 implementation order step 3, §4.1–4.5 patterns, §5.1–5.2 structure and boundaries]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-01, FR-02, FR-03, NFR-02, NFR-03, NFR-04]
- [Source: internal/ingest/ingest.go — ingest pipeline and ordering comments]
- [Source: internal/domain/summary.go — OperationSummary JSON contract]
- [Source: internal/store/migrations/001_initial.sql — assets table and indexes]
- [Source: internal/store/assets.go — FindAssetByContentHash, InsertAsset]
- [Source: internal/paths/canonical.go — CanonicalDayDir, SuggestedFilename]
- [Source: internal/filehash/filehash.go — SumHex, ReaderHex]
- [Source: internal/exifmeta/capture.go — ReadCapture, timezone rule]
- [Source: _bmad-output/implementation-artifacts/1-2-capture-time-hash.md — Story 1.2 dev notes and completion scope]
- [Source: _bmad-output/implementation-artifacts/epic-1-retrospective-20260413.md — ingest tests gap]

### Git / repo intelligence (snapshot)

- Repo contains implemented `internal/ingest`, `internal/domain/summary.go`, and `internal/store/assets.go` aligned with architecture; **automated ingest tests are the main documented gap** before treating 1.3 as fully proven.

## Dev Agent Record

### Agent Model Used

Composer (Cursor agent)

### Debug Log References

(none)

### Completion Notes List

- Verified `ingestOne` matches AC1–4 (hash-before-copy, seek + `ReaderHex`, early/late duplicate handling, destination cleanup on insert failure).
- Extended `internal/ingest` package comment with SHA-256 dedup semantics (NFR-03 / size+hash via digest).
- Added `internal/ingest/ingest_test.go`: duplicate ingest + three-file batch against real SQLite + library layout; asserts row shape, canonical rel_path prefix, and per-call `OperationSummary` fields.
- Added `internal/domain/summary_test.go` to lock JSON `snake_case` contract for `OperationSummary`.

### File List

- internal/ingest/ingest.go
- internal/ingest/ingest_test.go
- internal/domain/summary_test.go
- _bmad-output/implementation-artifacts/1-3-core-ingest.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

### Change Log

- 2026-04-13: Story 1.3 — ingest integration tests, dedup documentation, OperationSummary JSON test; sprint status `1-3-core-ingest` → review.
