# Story 1.5: Desktop upload flow with collection confirm and receipt

Status: in-progress

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **to pick files, optionally assign an upload collection, and confirm before anything is created**,  
So that **I get predictable organization without surprise albums**.

**Implements:** FR-04, FR-05, FR-06; NFR-04; UX-DR6.

## Acceptance Criteria

1. **Given** the user selects multiple images via file picker, **when** import completes successfully, **then** each new file is ingested per Story 1.3 and the UI shows an **operation receipt** with added / duplicate / failed counts (UX-DR6, FR-03, NFR-04).
2. **Given** the upload flow offers collection assignment, **when** the user has not confirmed, **then** no new collection and no links are persisted (FR-06).
3. **Given** the default collection name pattern `Upload YYYYMMDD`, **when** the user confirms, **then** the collection is created/updated and all batch assets are linked as specified (FR-04, FR-05).
4. **Given** the user clears or renames the collection name before confirm, **when** they confirm, **then** the persisted name matches their input.

## Tasks / Subtasks

- [x] **Fyne upload surface** (AC: 1–4)
  - [x] Multi-select file dialog (`dialog.FileOpen` or equivalent) filtered to supported image types; collect absolute paths.
  - [x] Wire library bootstrap: `config.ResolveLibraryRoot`, `EnsureLibraryLayout`, `store.Open` — reuse patterns from `main.go`; keep **no SQL in widgets** (architecture §5.2).
- [x] **Ingest batch** (AC: 1)
  - [x] Call `ingest.Ingest(db, libraryRoot, paths)` after selection; capture `domain.OperationSummary`.
  - [x] **UX order (mandatory):** Match [_bmad-output/planning-artifacts/ux-design-specification.md — Journey A]: hash/dedupe/write assets **before** collection confirm; receipt comes **after** the batch finishes (including the no-collection and cancel-collection branches). Only **collections + `asset_collections` rows** are blocked until confirm (FR-06).
- [x] **Resolve asset IDs for the batch** (AC: 3)
  - [x] For every selected file, determine the **canonical asset id** after ingest: **new row** → id for inserted hash; **skipped duplicate** → id of existing row for that `content_hash`. Today `FindAssetByContentHash` only returns `bool`; extend `internal/store` (e.g. `AssetIDByContentHash`) or extend `internal/ingest` to return per-file outcomes — avoid ambiguous N+1 guessing.
  - [x] Link set must include **all batch files that resolved to an asset** (added + duplicate), so duplicates still join the album per FR-04.
- [x] **Collection gate** (AC: 2–4)
  - [x] Optional assignment UI: checkbox or “Skip collection” vs “Assign to collection” per Journey A; if assigning, show name field defaulting to `Upload YYYYMMDD` where **YYYYMMDD** is the **local calendar date at batch start** (or first ingest kickoff — pick one, document in code; PRD FR-05).
  - [x] **Confirm** (primary): if user chose assignment and name is non-empty after trim → `store.CreateCollection` + `store.LinkAssetsToCollection` with collected asset IDs. If name cleared / user skipped assignment / user cancels collection step → **no** `CreateCollection`, **no** links.
  - [x] **Cancel** on collection step: no new collection and no links; ingested files and assets remain (Journey A branch `G|No| I`).
- [x] **Receipt panel** (AC: 1, UX-DR6)
  - [x] Plain labels: **Added**, **Skipped duplicate** (or “Duplicate” in UI copy if shorter — must map one-to-one to `OperationSummary` fields), **Failed**; show `Updated` if non-zero for forward compatibility.
  - [x] Same field names and meanings as CLI will use (`internal/domain/summary.go` JSON tags).
- [x] **Tests / verification**
  - [x] Prefer `fyne test` or lightweight harness if available; minimum: document manual QA checklist in Dev Agent Record (multi-file, all-duplicate batch, confirm vs skip vs cancel, renamed default collection).
  - [x] `go test ./...` stays green; no new Fyne imports under `internal/ingest`.

## Dev Notes

### Technical requirements

- **Prerequisites:** Story 1.3 (`ingest.Ingest`, `OperationSummary`); Story 1.4 (`CreateCollection`, `LinkAssetsToCollection`, schema v2). Coordinate with any in-flight **1.4 review** items (e.g. FK surfacing on link) so upload does not rely on silent `IGNORE` for bad IDs.
- **FR-06 scope:** No `collections` / `asset_collections` writes until the user confirms assignment. Ingest may persist **assets** and files earlier per UX Journey A; do not conflate “no collection” with “no ingest.”
- **FR-04 “created/updated”:** MVP can **create** a new collection on each confirmed assign. If product later requires **updating** an existing collection (same name), define lookup/merge rules; until then, document “always new row on confirm” if that is the chosen behavior to avoid silent merges.
- **Default name:** `Upload YYYYMMDD` — align with `CreateCollection` display-date semantics (calendar date, local) in `internal/store/collections.go`.
- **Errors:** User-visible copy for picker/IO failures; `slog` + `%w` wrapping at boundaries (architecture §4.2–4.3).

### Architecture compliance

- **Boundaries:** Fyne UI in `internal/app` (or root `main.go` only while `internal/app` is not created); call `ingest` + `store` only from UI layer — architecture §5.1–5.2, §5.3 (FR-01–FR-06 → ingest + store).
- **Single pipeline:** Same `ingest.Ingest` and `OperationSummary` as CLI scan/import later (architecture §3.9, §3.12 step 3).
- **Target layout:** Architecture §5.1 recommends `internal/app/` for Fyne shell — prefer placing upload view there when introducing navigation; `main.go` may remain thin bootstrap.

### Library / framework

- **Fyne** v2.7.3; **Go** 1.25.4 per architecture frontmatter.
- **SQLite** via existing `store.Open`; no new DB driver.

### File structure (touch / extend)

- `main.go` — extend with upload UI **or** introduce `internal/app/upload.go` (or `internal/app/…`) and call from `main`.
- `internal/ingest/ingest.go` — optional: richer batch result for asset IDs (if not solved purely in `store`).
- `internal/store/assets.go` — optional: `AssetIDByContentHash` (or rename `FindAssetByContentHash` to return `(id int64, ok bool, err error)`).
- `internal/domain/summary.go` — consume only; do not change contract without NFR-04 alignment.

### Testing requirements

- Table-driven / integration patterns per architecture §4.4 where automation is feasible for non-UI parts (e.g. new store helper tests).
- Document manual Fyne checklist for AC1–4 and Journey A branches (assign yes/no, confirm/cancel).

### Previous story intelligence

- **1.4:** `CreateCollection` trims name, default `display_date` local ISO; `LinkAssetsToCollection` transactional with `ON CONFLICT DO NOTHING` for idempotency; invalid FKs should error (see 1.4 review: prefer FK failures over silent ignore).
- **1.3:** `InsertAsset` does not return `id`; linking after ingest requires an explicit ID resolution strategy. Duplicate path never calls `InsertAsset` — must still resolve existing asset id by hash.

### Scope boundaries

- **In scope:** File-picker upload, receipt, optional collection confirm, single-batch assignment as in AC (not full Story 2.9 CRUD UI).
- **Out of scope:** Drag-and-drop (Story 1.8); full nav shell/themes (Story 2.1); multi-collection picker for one batch unless you fold minimal UI without breaking AC singularity.

### Project structure notes

- Brownfield: `main.go` is a minimal window today — this story substantially grows UI surface; follow architecture §5.1 migration toward `internal/app` when practical.
- Receipt pattern and confirm gate: [_bmad-output/planning-artifacts/ux-design-specification.md — UX-DR6, Journey A, Operation receipt panel].

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Story 1.5, Epic 1, FR coverage map]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-04, FR-05, FR-06, FR-03, NFR-04, Journey A]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.2, §3.9 OperationSummary, §3.12 order, §4.1–4.5, §5.1–5.3]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — Journey A (flow order), UX-DR6 operation receipts, confirm-before-collection]
- [Source: internal/ingest/ingest.go — `Ingest` entry]
- [Source: internal/domain/summary.go — `OperationSummary`]
- [Source: internal/store/collections.go — `CreateCollection`, `LinkAssetsToCollection`]
- [Source: internal/store/assets.go — dedup / insert helpers]
- [Source: _bmad-output/implementation-artifacts/1-3-core-ingest.md — ingest behavior, summary contract]
- [Source: _bmad-output/implementation-artifacts/1-4-collections-schema.md — store API, migration v2, review findings]

## Dev Agent Record

### Agent Model Used

Composer (Cursor agent)

### Debug Log References

### Completion Notes List

- Implemented `store.AssetIDByContentHash` and `ingest.IngestWithAssetIDs` so each source path maps to a canonical `assets.id` (added or duplicate) without extra hashing passes.
- Fyne’s stock `dialog.FileOpen` is single-file; multi-file selection is **Add images…** repeatedly to accumulate absolute paths (equivalent per story).
- Default collection name date: **`batchStart := time.Now()`** at the start of **Import** (ingest kickoff), local `Upload YYYYMMDD`; `CreateCollection` receives matching `display_date` ISO for that same local calendar day.
- **Manual QA checklist:** (1) Add 2+ JPEGs, Import — receipt counts match expectations. (2) Import same files again — duplicates in receipt, asset IDs still link if assigning. (3) Skip collection + Confirm — no new `collections` row. (4) Assign + default name + Confirm — new collection and links. (5) Assign + rename field + Confirm — stored name matches entry. (6) Assign + clear name + Confirm — no collection. (7) Cancel after import — no collection; assets remain. (8) Mix of good and bad paths — Failed > 0, failed paths not linked.

### File List

- main.go
- internal/app/upload.go
- internal/ingest/ingest.go
- internal/ingest/ingest_test.go
- internal/store/assets.go
- internal/store/store_test.go
- _bmad-output/implementation-artifacts/sprint-status.yaml

### Change Log

- 2026-04-13: Story 1.5 — desktop upload with ingest receipt, optional collection confirm, `AssetIDByContentHash` + `IngestWithAssetIDs`, store/ingest tests; sprint status → review.
- 2026-04-13: BMAD code review — findings below; status → in-progress until patches addressed.

### Review Findings

- [ ] [Review][Patch] Orphan collection when assign succeeds but nothing is linkable — If every selected path fails ingest (or all `lastAssetIDs` filter to empty), `CreateCollection` still runs when the user assigns a non-empty name, `LinkAssetsToCollection` returns immediately on an empty ID list, and `summarizeDoneMessage` still reports that assets were linked. Prefer gating collection creation on at least one resolved asset ID and aligning the completion copy with actual links. [`internal/app/upload.go` ~166–188]

- [ ] [Review][Patch] Import can run again before collection confirm — The Import button stays enabled after the first ingest, so the user can re-run the batch, overwrite the receipt/`batchStart` (shifting default `Upload YYYYMMDD`), and blur Journey A’s “receipt then confirm” step. Prefer disabling Import until Confirm/Cancel (or require clearing paths explicitly). [`internal/app/upload.go` ~135–151, ~159–164]

- [ ] [Review][Patch] Create + link is not atomic — If `CreateCollection` succeeds and `LinkAssetsToCollection` fails, an empty orphan collection remains; only the link error is shown. Prefer a single transactional API (create+link) or compensating delete on link failure. [`internal/app/upload.go` ~171–184]

- [x] [Review][Defer] `isUniqueContentHash` relies on SQLite error substring matching — Acceptable MVP risk; revisit if driver/sqlite wording changes. [`internal/ingest/ingest.go` ~174–183] — deferred, pre-existing pattern extended in this story

---

**Story context:** Ultimate context engine analysis completed — comprehensive developer guide created (BMAD create-story workflow, 2026-04-13).
