# Story 1.5: Desktop upload flow with collection confirm and receipt

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **to pick files, optionally assign an upload collection, and confirm before anything is created**,  
So that **I get predictable organization without surprise albums**.

**Implements:** FR-04, FR-05, FR-06; NFR-04; UX-DR6, UX-DR17.

## Acceptance Criteria

1. **Given** the user selects multiple images via file picker, **when** import completes successfully, **then** each new file is ingested per Story 1.3 and the UI shows an **operation receipt** with added / duplicate / failed counts (UX-DR6, FR-03, NFR-04).
2. **Given** the upload flow offers collection assignment, **when** the user has not confirmed, **then** no new collection and no links are persisted (FR-06).
3. **Given** the default collection name pattern `Upload YYYYMMDD`, **when** the user confirms, **then** a **new** collection row is created (MVP: no merge into an existing same-named collection; see Dev Notes) and all batch assets are linked as specified (FR-04, FR-05).
4. **Given** the user clears or renames the collection name before confirm, **when** they confirm, **then** the persisted name matches their input.
5. **Given** a multi-file pick, **when** the confirm step is shown, **then** the UI surfaces **large previews** of the batch (image-first **ingest** stage per UX **Direction E**); the **operation receipt** remains readable and may be **collapsed** after the user has learned the pattern (UX spec **feedback patterns**).
6. **Given** ingest work runs off the UI thread, **when** results are applied, **then** **Fyne** widgets update on the **main thread** (UX-DR17).

## Tasks / Subtasks

- [x] **Fyne upload surface** (AC: 1, 5, 6)
  - [x] Multi-select file dialog (`dialog.FileOpen` or equivalent) filtered to supported image types; collect absolute paths (stock dialog is single-file — **Add images…** accumulates paths; equivalent per story).
  - [x] Wire library bootstrap: `config.ResolveLibraryRoot`, `EnsureLibraryLayout`, `store.Open` — reuse patterns from `main.go`; keep **no SQL in widgets** (architecture §5.2).
  - [x] **Direction E / AC5:** **Batch preview** strip (horizontal scroll, **≥140×140** min cell, **first 6** files + **“+ N more…”** copy) **above** the collapsible receipt; **“Files in this batch: N”** retained for scale-at-a-glance.
- [x] **Ingest batch** (AC: 1, 6)
  - [x] Call `ingest.IngestWithAssetIDs` after selection; capture `domain.OperationSummary` and per-path asset IDs.
  - [x] **UX order (mandatory):** Match [_bmad-output/planning-artifacts/ux-design-specification.md — Journey A]: hash/dedupe/write assets **before** collection confirm; receipt comes **after** the batch finishes (including the no-collection and cancel-collection branches). Only **collections + `asset_collections` rows** are blocked until confirm (FR-06).
  - [x] **UX-DR17:** Production path runs ingest in a worker goroutine and applies UI updates with `fyne.Do`; `UploadViewOptions.SynchronousIngest` for `fyne test` (driver does not serialize `fyne.Do` across `Tap` boundaries).
- [x] **Resolve asset IDs for the batch** (AC: 3)
  - [x] For every selected file, determine the **canonical asset id** after ingest: **new row** → id for inserted hash; **skipped duplicate** → id of existing row for that `content_hash` (`store.AssetIDByContentHash`, `ingest.IngestWithAssetIDs`).
  - [x] Link set must include **all batch files that resolved to an asset** (added + duplicate), so duplicates still join the album per FR-04.
- [x] **Collection gate** (AC: 2–4)
  - [x] Optional assignment UI: “Skip collection” vs “Assign to collection” per Journey A; name field defaulting to `Upload YYYYMMDD` where **YYYYMMDD** is the **local calendar date** at **`batchStart := time.Now()`** when **Import** starts (FR-05).
  - [x] **Confirm** (primary): if user chose assignment and name is non-empty after trim → `store.CreateCollectionAndLinkAssets` with collected asset IDs. If name cleared / user skipped assignment → **no** collection, **no** links.
  - [x] **Cancel** on collection step: no new collection and no links; ingested files and assets remain (Journey A branch `G|No| I`).
- [x] **Receipt panel** (AC: 1, 5, UX-DR6)
  - [x] Plain labels: **Added**, **Skipped duplicate** (or shorter UI copy — must map one-to-one to `OperationSummary` fields), **Failed**; show **Updated** if non-zero.
  - [x] Same field names and meanings as CLI (`internal/domain/summary.go` JSON tags; NFR-04).
- [x] **Window close / re-entrancy** (AC: 2, 6)
  - [x] `SetCloseIntercept` while `importInFlight` **or** `awaitingPostImportStep` (user must Confirm/Cancel collection before closing); `UploadViewOptions.DisableImportCloseIntercept` when another owner manages close.
- [x] **Tests / verification**
  - [x] `go test ./...` green; upload flow tests in `internal/app/upload_fr06_flow_test.go`, `e2e_shell_journeys_test.go` (`collectLabelsDeep` walks `Accordion` for receipt labels).
  - [x] **Dev party 2/2:** `TestUpload_flow_assignRadioButNameCleared_noCollection`, `TestUpload_flow_receiptShowsFilesInBatchCount`.
  - [x] Manual QA checklist in Dev Agent Record (multi-file, all-duplicate batch, confirm vs skip vs cancel, renamed default collection).
- [x] **AC5 closure (product / UX):** **Preview strip** on confirm (`internal/app/upload.go`: `uploadPreviewStripMaxItems`, `uploadPreviewThumbMin`); regression tests `TestUpload_flow_confirmStep_showsBatchPreviewStrip`, `TestUpload_flow_previewStrip_capsAtSixWithOverflowLabel`.

## Risks / open edges (BMAD create party 2026-04-15)

**Session 1/2**

1. **Decode hitch:** Thumbnails load from **source paths** on the UI goroutine when the receipt step appears; pathological huge images could stall — if profiling shows jank, move decode to a worker and swap in resources via `fyne.Do` (keep cap at six).
2. **Source vs library bytes:** Previews show **picked files**, not post-ingest library paths; if a user deletes a source file before Confirm, a tile may fail to render — acceptable rare edge; a later improvement is cache/thumb from library `rel_path`.
3. **“Collection updated” vs created:** **Update-existing-by-name** remains **out of scope**; each confirm **creates** a new collection row (see Dev Notes). PM/architecture story if product needs merge semantics.

**Session 2/2 (challenge prior round)**

4. **AC3 / epic wording:** “Created/updated” implied merge semantics we explicitly deferred — **AC3** and **epics Story 1.5** now say **new row + MVP no same-name merge** to match implementation and Dev Notes.
5. **Window close during collection step:** Previously only `importInFlight` blocked close; users could exit mid-step without the same explicit branch as **Cancel**. **Mitigation:** close intercept now covers **`awaitingPostImportStep`** (parity with import guard). **Headless caveat:** Fyne `test.Window.Close()` does **not** invoke `SetCloseIntercept` — **policy** is covered by **`uploadImportCloseBlocked`** unit tests (`TestUpload_importCloseBlocked_policy`); full close UX remains **manual QA**.
6. **AC5 “large” on huge displays:** The strip uses a **fixed minimum edge** (140dp), not responsive scaling; acceptable MVP bound per Direction E; revisit if UX research says thumbnails read too small on 4K+.

**Session 2/2 — dev party (hook dev, 2026-04-15)**

7. **Assign without a usable name:** Session 1/2 focused on **close intercept**; the complementary FR-06 edge is **“Assign to collection”** with **whitespace-only** (or cleared) name — must **not** create a collection. **Mitigation:** `TestUpload_flow_assignRadioButNameCleared_noCollection`.
8. **Batch count label untested:** **“Files in this batch: N”** is part of the confirm-step receipt chrome; headless coverage now locks it to the picked file count via **`TestUpload_flow_receiptShowsFilesInBatchCount`**.
9. **Preview decode policy in code:** Session 1 **deferred** worker offload; session 2 **pins** the rationale in **`upload.go`** next to **`updateBatchPreview`** (bounded strip first; profile before async decode).

## Definition of Done (1.5)

- AC1–6 each have **automated** coverage **or** an explicit **manual** row in Dev Agent Record with pass/fail.
- **AC5:** Preview strip visible after successful batch ingest; receipt remains collapsible; **NFR-04** labels unchanged vs CLI.
- **UX-DR17:** Production ingest uses a worker + `fyne.Do`; `SynchronousIngest` only in tests.

## Dev Notes

### Technical requirements

- **Prerequisites:** Story 1.3 (`ingest.Ingest` / `IngestWithAssetIDs`, `OperationSummary`); Story 1.4 (`CreateCollectionAndLinkAssets`, schema v2). Upload must not rely on silent `IGNORE` for bad asset IDs — prefer surfaced FK errors.
- **FR-06 scope:** No `collections` / `asset_collections` writes until the user confirms assignment. Ingest may persist **assets** and files earlier per UX Journey A; do not conflate “no collection” with “no ingest.”
- **FR-04 “created/updated”:** MVP may **create** a new collection on each confirmed assign. If product later requires **updating** an existing collection (same name), define lookup/merge rules separately.
- **Default name:** `Upload YYYYMMDD` — align with `CreateCollection` display-date semantics (calendar date, local) in `internal/store/collections.go`.
- **Errors:** User-visible copy for picker/IO failures; `slog` + `%w` wrapping at boundaries (architecture §4.2–4.3).
- **AC5 vs implementation:** Direction E **large previews** satisfied by a **bounded** strip (six tiles, **140dp** min edge) **above** the receipt; overflow batch size is surfaced in copy + path list. Further polish (library-based thumbs, async decode) is optional if QA finds hitch on large files.

### Architecture compliance

- **Boundaries:** Fyne UI in `internal/app`; call `ingest` + `store` only from UI layer — architecture §5.1–5.2, §5.3 (FR-01–FR-06 → ingest + store).
- **Single pipeline:** Same `ingest` entry points and `OperationSummary` as CLI scan/import (architecture §3.9, §3.12).
- **Async UI:** architecture §3.8.1 — heavy work off UI thread; **`fyne.Do`** (or equivalent) for widget mutations (**UX-DR17**).

### Library / framework

- **Fyne** v2.7.3; **Go** 1.25.4 per `go.mod` / architecture.
- **SQLite** via existing `store.Open`; no new DB driver.

### File structure (touch / extend)

- `internal/app/upload.go` — primary upload flow, receipt, collection gate, `UploadViewOptions`.
- `main.go` — thin bootstrap; wires `NewUploadView` / shell.
- `internal/ingest/ingest.go` — `IngestWithAssetIDs` batch result.
- `internal/store/assets.go` — `AssetIDByContentHash` (or equivalent).
- `internal/domain/summary.go` — consume only; do not change contract without NFR-04 alignment.

### Testing requirements

- Table-driven / integration patterns per architecture §4.4 for store/ingest; Fyne headless tests use `SynchronousIngest` and label-walk helpers that recurse into `Accordion`.
- Manual Fyne checklist for AC1–6 and Journey A branches.

### Previous story intelligence

- **1.4:** `CreateCollection` trims name; `CreateCollectionAndLinkAssets` transactional; `LinkAssetsToCollection` idempotent; invalid FKs should error.
- **1.3:** Duplicate path never calls `InsertAsset` — must still resolve existing asset id by hash for linking.

### Scope boundaries

- **In scope:** File-picker upload, receipt, optional collection confirm, single-batch assignment as in AC (not full Story 2.9 CRUD UI).
- **Out of scope:** Drag-and-drop (Story 1.8); full nav shell/themes (Story 2.1).

### Project structure notes

- Brownfield: prefer `internal/app/` for Fyne per architecture §5.1; `main.go` stays thin.
- Receipt pattern: [_bmad-output/planning-artifacts/ux-design-specification.md — UX-DR6, Journey A, Operation receipt panel].

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Story 1.5, Epic 1, FR coverage map]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-03, FR-04, FR-05, FR-06, NFR-04, Journey A]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.8.1 UX-DR17, §3.9 OperationSummary, §3.12 ingest order, §4.1–4.5, §5.1–5.3]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — Journey A, Direction E, UX-DR6 operation receipts, feedback patterns]
- [Source: internal/app/upload.go — upload flow, `fyne.Do`, receipt Accordion, `UploadViewOptions`]
- [Source: internal/ingest/ingest.go — `IngestWithAssetIDs`]
- [Source: internal/domain/summary.go — `OperationSummary`]
- [Source: internal/store/collections.go — `CreateCollectionAndLinkAssets`]
- [Source: internal/store/assets.go — dedup / `AssetIDByContentHash`]
- [Source: _bmad-output/implementation-artifacts/1-3-core-ingest.md — ingest behavior, summary contract]
- [Source: _bmad-output/implementation-artifacts/1-4-collections-schema.md — store API, migration v2]

## Dev Agent Record

### Agent Model Used

Composer (Cursor agent)

### Debug Log References

### Completion Notes List

- **2026-04-15 (party mode dev session 2/2 — automated):** Simulated **Amelia / Sally / Winston / John** (dev hook). **Amelia:** session 1/2 over-indexed on close policy; add **assign + empty/whitespace name** regression (FR-06 parity with skip). **Sally:** confirm step **batch count** line is user-facing scale feedback — assert it. **Winston:** don’t async-decode previews without profiling; **document** bounded-strip + UI-thread decode next to **`updateBatchPreview`**. **John:** disagrees with hiding assign when name clears — keep radio explicit, gate on **trimmed** name only (current behavior). **Synthesis:** two tests + comment in `upload.go`; sprint/story **review** unchanged.
- **2026-04-15 (party mode dev session 1/2 — automated):** Simulated **Amelia / Sally / Winston / John** on **dev** hook: challenged create-session risks without repeating verbatim — **close intercept** testability vs async preview decode cost. **Synthesis:** extract **`uploadImportCloseBlocked`** (import vs collection-step copy + precedence when both flags set), wire **`SetCloseIntercept`** through it, add **`TestUpload_importCloseBlocked_policy`**; defer moving **`updateBatchPreview`** off-UI-thread (would reorder UI vs receipt for little gain until profiling). `go test ./...` green; story/sprint stay **review**.
- **2026-04-15 (dev-story / Composer):** Re-ran full validation: `go test ./...` and `go build .` green. Confirmed sprint `1-5-upload-confirm-receipt` and story **Status** both **review**; all Tasks/Subtasks remain **[x]**; implementation in `internal/app/upload.go` still satisfies AC1–AC6 (ingest receipt, preview strip, FR-06 collection gate, `fyne.Do` after worker ingest, close intercept for `importInFlight` and `awaitingPostImportStep`).
- **2026-04-15 (party mode create session 2/2):** Simulated **Mary / John / Sally / Winston** — challenged session 1/2: **epic+AC3 “created/updated”** vs MVP create-only; **close window during collection step** (unsettled UX vs FR-06). **Synthesis:** pin AC3/epic text to **new row / no merge**; extend **`SetCloseIntercept`** to **`awaitingPostImportStep`**; document **fyne test** close-intercept gap; add risk on **140dp vs large monitors**.
- **2026-04-15 (party mode create session 1/2):** Simulated **John / Sally / Winston / Paige** on **AC5** and spec debt: aligned on **bounded preview strip** (six × ≥140dp) before receipt, overflow copy, risks (sync decode, source-path previews); added story **Risks**, **DoD**, closed AC5 task; sprint `1-5` → **review** to match implementation + spec.
- **2026-04-14 (party mode dev session 2/2):** Challenged session 1/2: (1) **Window close vs async ingest** — `SetCloseIntercept` blocks close while `importInFlight` so a worker cannot schedule `fyne.Do` after teardown; `UploadViewOptions.DisableImportCloseIntercept` for special cases (document: chain if shell adds its own intercept). (2) **UX backlog (epics delta)** — receipt wrapped in **Fyne `Accordion`** (default open, user-collapsible) + **“Files in this batch: N”** line for batch-at-a-glance without full thumbnail grid yet. (3) **AC4 / FR-04 tests** — `TestUpload_flow_renameFromDefault_persistsCollectionName`, `TestUpload_flow_duplicatePathsInBatch_singleLinkRow`. (4) **Label walk** — `collectLabelsDeep` recurses into `Accordion` so upload copy smoke tests still find receipt placeholders.
- **2026-04-14 (party mode dev session 1/2):** Challenged UX-DR17 on Upload: ingest no longer blocks the Fyne event thread in production — `go` + `ingest.IngestWithAssetIDs` + `fyne.Do` for receipt/collection UI; `importInFlight` + “Importing…” label; drop blocked while import runs. Headless tests set `UploadViewOptions.SynchronousIngest` because the fyne test driver does not serialize `fyne.Do` after the Import handler returns (avoids Tap(Confirm) racing background ingest).
- **2026-04-14 (dev-story close):** Confirm path uses `store.CreateCollectionAndLinkAssets` so collection insert and membership writes commit or roll back together. Orphan-collection and “no linkable assets” gates remain: assignment runs only when `len(link) > 0`; completion copy uses `summarizeDoneMessage` with `actuallyLinked`. Import / Add / Clear stay disabled while `awaitingPostImportStep` (regression: `TestUpload_flow_duringCollectionStep_importAddClearDisabled`).
- Implemented `store.AssetIDByContentHash` and `ingest.IngestWithAssetIDs` so each source path maps to a canonical `assets.id` (added or duplicate) without extra hashing passes.
- Fyne’s stock `dialog.FileOpen` is single-file; multi-file selection is **Add images…** repeatedly to accumulate absolute paths (equivalent per story).
- Default collection name date: **`batchStart := time.Now()`** at the start of **Import** (ingest kickoff), local `Upload YYYYMMDD`; `CreateCollection` receives matching `display_date` ISO for that same local calendar day.
- **Manual QA checklist:** (1) Add 2+ JPEGs, Import — receipt counts match expectations. (2) Import same files again — duplicates in receipt, asset IDs still link if assigning. (3) Skip collection + Confirm — no new `collections` row. (4) Assign + default name + Confirm — new collection and links. (5) Assign + rename field + Confirm — stored name matches entry. (6) Assign + clear name + Confirm — no collection. (7) Cancel after import — no collection; assets remain. (8) Mix of good and bad paths — Failed > 0, failed paths not linked. (9) After Import, try to close the window before Confirm/Cancel — **blocked** with “Collection step pending”; then Cancel — window can close; no collection.

### File List

- main.go
- internal/app/upload.go (batch preview strip: `uploadPreviewStripMaxItems`, `uploadPreviewThumbMin`, `updateBatchPreview`; `uploadImportCloseBlocked` + close intercept)
- internal/app/upload_fr06_flow_test.go (`TestUpload_flow_confirmStep_showsBatchPreviewStrip`, `TestUpload_flow_previewStrip_capsAtSixWithOverflowLabel`, `TestUpload_importCloseBlocked_policy`, `countCanvasImagesDeep`)
- internal/app/e2e_shell_journeys_test.go (`collectLabelsDeep` Accordion walk)
- internal/ingest/ingest.go
- internal/ingest/ingest_test.go
- internal/store/assets.go
- internal/store/store_test.go
- _bmad-output/implementation-artifacts/sprint-status.yaml

### Change Log

- 2026-04-15: **party mode dev session 2/2 (1-5)** — `TestUpload_flow_assignRadioButNameCleared_noCollection`, `TestUpload_flow_receiptShowsFilesInBatchCount`; `upload.go` preview-decode note; story risks (Session 2/2 dev party); remains **review**.
- 2026-04-15: **party mode dev session 1/2 (1-5)** — `uploadImportCloseBlocked` + table test for close policy; risk #5 notes unit vs manual QA; story remains **review**.
- 2026-04-15: **dev-story (1-5)** — Validation only: `go test ./...` + `go build .` green; Dev Agent Record completion note; story remains **review**.
- 2026-04-15: **BMAD party mode create 2/2 (1-5)** — AC3/epic **create-only** wording; **close intercept** for collection step + story risks/QA; sprint note.
- 2026-04-15: **BMAD party mode create 1/2 (1-5)** — AC5 **preview strip** + tests; story **Risks** / **DoD**; tasks closed; sprint tracker `1-5` → review.
- 2026-04-15: **BMAD create-story** — Acceptance criteria merged to epics Story 1.5 (six BDD bullets); tasks remapped to AC 1–6; AC5 Direction E gap documented; References refreshed; **Status preserved:** review.
- 2026-04-14: Party mode dev session 2/2 (1-5) — import close intercept, receipt accordion + batch count label, rename/duplicate-batch tests, collectLabelsDeep accordion; status → review.
- 2026-04-14: Party mode dev session 1/2 — UX-DR17 async ingest + fyne.Do, importInFlight/drop guard, SynchronousIngest for fyne_test; status → review.
- 2026-04-14: Story 1-5 dev-story — review follow-up (transactional create+link, import gate test); status → review.
- 2026-04-13: Story 1-5 — desktop upload with ingest receipt, optional collection confirm, `AssetIDByContentHash` + `IngestWithAssetIDs`, store/ingest tests; sprint status → review.
- 2026-04-13: BMAD code review — findings below; status → in-progress until patches addressed.

### Review Findings

- [x] [Review][Patch] Orphan collection when assign succeeds but nothing is linkable — Gated: collection path runs only when `wantedAssign && len(link) > 0`; completion copy uses `actuallyLinked` (`summarizeDoneMessage`). Covered by `TestUpload_flow_allFailedIngest_assignDoesNotCreateCollection`.

- [x] [Review][Patch] Import can run again before collection confirm — `awaitingPostImportStep` disables Import, Add images…, and Clear list until Confirm or Cancel. Covered by `TestUpload_flow_duringCollectionStep_importAddClearDisabled`.

- [x] [Review][Patch] Create + link is not atomic — Confirm uses `store.CreateCollectionAndLinkAssets` (single transaction; rollback on link failure).

- [x] [Review][Defer] `isUniqueContentHash` relies on SQLite error substring matching — Acceptable MVP risk; revisit if driver/sqlite wording changes. [`internal/ingest/ingest.go` ~174–183] — deferred, pre-existing pattern extended in this story

- [ ] [Review][Patch] Overflow label test couples to a loose "+ 2 more" substring — `TestUpload_flow_previewStrip_capsAtSixWithOverflowLabel` can pass on unrelated copy or fail if wording changes; assert the full formatted overflow line using the same rule as production (`n - uploadPreviewStripMaxItems` and the format string at `internal/app/upload.go` ~157). [`internal/app/upload_fr06_flow_test.go` ~678–686]

- [x] [Review][Defer] AC5 minimum thumbnail edge (140dp via `uploadPreviewThumbMin`) is not asserted in automated tests — implementation satisfies Direction E; add a size/theme-scaled assertion only if QA wants a hard gate. [`internal/app/upload.go` ~27–29, ~149–153] — deferred, test gap

---

**Story context:** Ultimate context engine analysis completed — comprehensive developer guide created (BMAD create-story workflow, 2026-04-13). **Refreshed 2026-04-15** per create-story merge: epics AC parity, sprint tracker set to `ready-for-dev` for spec handoff.
