# Story 2.3: Paged thumbnail grid with rating and reject badges

Status: done

<!-- Ultimate context engine analysis completed — comprehensive developer guide created. -->
<!-- 2026-04-15: Party mode create 1/2 — AC6 + UX-DR18 guardrail + Risks/DoD + rejected-list task; sprint synced done. -->
<!-- 2026-04-15: Party mode dev1/2 — UX-DR18: hide grid `Scroll` when count is zero or count query fails (Review + Rejected); no faux grid viewport under empty state. -->
<!-- 2026-04-15: Party mode dev2/2 — `syncGridScrollVisible`; Rejected count-error escape hatch (`Back to Review`); `thumbGen` drops stale thumbnail `fyne.Do` after `reset`/`invalidatePages` (closed-DB test panic fix). -->
<!-- 2026-04-15: Party mode create 2/2 — epic §2.3 AC parity (pending vs failed + zero-row grid); `reviewGridListRowCount` + test (UX-DR18 automation). -->
<!-- 2026-04-15: BMAD create-story merge — AC aligned to epics.md Story 2.3; Dev Notes refreshed (architecture §3.8.1); Status preserved. -->

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **an image-forward grid that still shows rating and reject at a glance**,  
So that **photos stay primary while I triage quickly**.

**Implements:** FR-07 (display tags/ratings context); supports FR-08/FR-29 display; UX-DR3, UX-DR16.

## Acceptance Criteria

1. **Given** filtered assets, **when** the grid loads, **then** thumbnails load incrementally (paged or windowed) without loading all pixmaps at once (architecture §3.8 **Grid performance**).
2. **Given** an asset with rating or reject state, **when** shown in grid, **then** badges reflect DB state within the PRD **1 second** guideline for local single-user use (FR-10, SC-3 / FR-07 baseline for display). *Implementation note:* after a filter change or initial load, visible cells must show correct `rating` / `rejected` from the same query row without stale placeholders beyond ~1s under normal local SQLite + decode latency; keyboard rating persistence is Story **2.4**, but grid **display** must match DB.
3. **Given** decode failure, **when** a cell renders, **then** the user sees failed-decode affordance (placeholder + explanation) per **UX-DR3** (component strategy: icon + caption; not a silent blank).
4. **And** at default density, **thumbnail imagery** is the **largest** element in the cell; nonessential chrome **defers** to **hover/focus** where feasible (**UX-DR3**, **UX-DR16**).
5. **And** **minimum** thumbnail **readability** at **1024×768** and **1920×1080** reference layouts is **recorded** in Story **2.11** / NFR-01 evidence (numeric thresholds—not ad hoc).
6. **Given** a visible cell whose thumbnail is still decoding, **when** it renders, **then** the **pending** state is **visually distinct** from **failed** decode (UX-DR3): non-blank pending affordance (e.g. `theme.MediaPhotoIcon()` as `Resource`) versus **failed** path (error icon + explanation). Silent empty image regions are not acceptable for pending.

### Extended guardrails (UX tone / list bind failures)

Epics state AC1–5 above (**AC6** is story-level QA traceability for pending vs failed decode); the following matches **architecture §4.2–4.3** and Story **2.12** error-tone expectations for grid **list/bind** paths:

- **Given** a paged grid query or row-bind failure (SQLite / I/O), **when** the UI must react, **then** cells use **short, non-technical** copy (no driver/SQL fragments); details go to structured **`slog`**.
- **UX-DR17:** Decode and heavy IO off the UI thread; apply Fyne mutations on the main thread (`fyne.Do` or project standard).
- **UX-DR18 (shell coordination):** When the filtered **count is zero**, the Review body shows the **empty state** block (`review.go`), the grid list exposes **no faux rows**, and the **grid scroll viewport is hidden** so empty copy is not stacked above inert chrome; same for **Rejected** when hidden count is zero. When the **count query fails**, the scroll is hidden (count line carries the error). **Rejected:** on count failure, **Back to Review** stays available (medium emphasis) so the panel is not a dead-end. Full empty/loading/error/populated matrix remains Story **2.12**; this story only pins **grid cell** and **page/bind** behavior.

### Risks & Definition of Done (create)

- **Risks:** (1) **Count vs list drift** if `ReviewFilterWhereSuffix` arg order diverges—mitigated by shared helpers + contract tests. (2) **Stale async thumbnails** on fast scroll—mitigated by `thumbnailBinding` + `fyne.Do`. (3) **Accidental error text in UI**—mitigated by fixed user strings + `TestReviewGridUserFacingMessagesSanitized`. (4) **Rejected surface** must not fork paging/error semantics silently—same `review_grid` + `ListRejectedForReview` path. (5) **Epic vs story AC drift** (stakeholders read `epics.md` first)—mitigated by keeping Story **2.3** and epic §2.3 acceptance criteria aligned on pending/failed decode and zero-count grid coordination.
- **DoD:** `go test ./...` green; store list/count parity tests cover default Review **and** rejected list where applicable; grid user copy contains no SQL/driver tokens; Story **2.11** evidence link remains valid for AC5.

## Tasks / Subtasks

- [x] **Wire `libraryRoot` into Review** (AC: 1, 3)
  - [x] Extend `NewReviewView` / `NewMainShell` so Review receives `libraryRoot` (same as `main.go` → `NewMainShell(win, db, root)`), enabling absolute paths from `assets.rel_path` and `{libraryRoot}/.cache/thumbnails/` per architecture §3.8.
- [x] **Store: paged asset list with filter parity** (AC: 1, 2)
  - [x] Add `ListAssetsForReview` (name may vary) that selects rows needed for grid cells (at minimum: `id`, `rel_path`, `rating`, `rejected`, `mime`, `width`, `height`, `capture_time_unix`—trim only if proven unused).
  - [x] **Rejected/Hidden grid (Story 2.6):** same `ReviewGridRow` shape and paging semantics via `CountRejectedForReview` / `ListRejectedForReview` + `ReviewFilterWhereSuffix` parity; `review_grid` `rejectedMode` must not bypass sanitization or thumbnail cache rules.
  - [x] SQL **must** use `WHERE ` + `store.ReviewBrowseBaseWhere` + suffix from `store.ReviewFilterWhereSuffix(filters)` with **identical argument order** to `CountAssetsForReview` (reuse Story 2.2 contracts; extend `review_query_test.go` if list drifts).
  - [x] `ORDER BY` must be **stable** (recommend `capture_time_unix DESC`, `id DESC` tie-break) so paging is deterministic.
  - [x] Accept `LIMIT`/`OFFSET` (or keyset) parameters; **do not** `SELECT *` the entire library for the grid.
- [x] **Thumbnail pipeline: incremental decode + cache** (AC: 1, 3)
  - [x] Follow architecture §3.8: **disk cache** under `{libraryRoot}/.cache/thumbnails/`; avoid holding full decoded pixmaps for off-screen pages.
  - [x] Decode **off the UI goroutine** when work is non-trivial; apply results with Fyne-safe scheduling (`fyne.Do` after `WriteThumbnailJPEG`).
  - [x] On irrecoverable decode error, surface **UX-DR3** failed state in the cell (icon + label—not raw error dumps).
- [x] **Grid UI: dense, scannable cells** (AC: 1–4, extended guardrails)
  - [x] Replace the Story 2.2 **placeholder** body (`gridHint` in `internal/app/review.go`) with a real grid/list that **virtualizes** or **pages** (Fyne `List` rows of `reviewGridColumns` cells + DB-backed paging in pages of `reviewGridPageSize`).
  - [x] Cell layout aligns with UX **Thumbnail grid cell**: uniform thumbnail area (letterboxed via `canvas.ImageFillContain`), **rating badge**, **reject indicator** (caution semantic when `rejected=1`; **not** color-only—`Hidden` label + `WarningImportance`). **AC4:** thumbnail imagery is the dominant read; defer nonessential chrome to hover/focus where feasible.
  - [x] **Default Review** list excludes `rejected=1` (same as count); reject badge still **implemented** for correct rendering if the cell is reused when `rejected=1` appears in row data later.
  - [x] **Pending decode:** distinguish from failed decode (e.g. placeholder resource such as `theme.MediaPhotoIcon()` while decode runs; **UX-DR3** / architecture §3.8.1 stale-work semantics).
  - [x] **Page/bind failures:** calm user copy + structured `slog`; avoid per-cell log spam (e.g. memoize failed page); no SQL-ish strings in labels.
- [x] **Filter integration** (AC: 1, 2)
  - [x] On any filter strip change, **reload** the grid from page 0 using the same `domain.ReviewFilters` mapping as `buildFilters()` in `review.go` (no second filter definition).
  - [x] Keep **live count** + grid in sync (count + `reset` share one refresh path).
- [x] **NFR-01 / Story 2.11 traceability** (AC: 5)
  - [x] Ensure minimum thumbnail readability at **1024×768** and **1920×1080** is captured in Story **2.11** evidence (`nfr-01-layout-matrix-evidence.md` or successor); grid uses the **same measurement anchor** (window box + lifecycle moment) as documented in architecture §3.8.1.
- [x] **Tests** (AC: 1–6, extended guardrails)
  - [x] Store: table-driven tests for list SQL—same row set as count for small fixtures; paging boundaries (empty page, last page partial).
  - [x] Store: `ListAssetsForReview` rejects invalid `limit` / `offset` (guardrails for callers).
  - [x] Grid: regression that user-facing messages do not leak SQL/driver tokens where automated tests exist.
  - [x] Grid: `reviewGridListRowCount` + `TestReviewGridListRowCount` — zero / negative totals yield **no** Fyne list rows (UX-DR18 / empty-state coordination).
  - [x] Grid: `reviewAssetGrid.thumbGen` — async thumbnail completions no-op after `reset` / `invalidatePages` (avoids stale `fyne.Do` touching recycled cells; headless `TestRejectedView_closedDB_*` stability).
  - [x] Optional: lightweight test for thumbnail error path (invalid bytes / missing file) if injectable without heavy GUI.
  - [x] `go test ./...` remains green.

### Review Findings

<!-- BMAD code review (2026-04-15); headless run — patch/decision items none; defers recorded. -->

- [x] [Review][Defer] Duplicated `ListCollections` + option/`collectionIDs` reconciliation between `refreshReviewData` and `refreshRejectedData` — maintenance/drift risk if one panel fixes selection logic and the other lags [`internal/app/review.go`, `internal/app/rejected.go`] — deferred, pre-existing pattern extended in this diff

- [x] [Review][Defer] `newMainShell` now **panics** when any primary nav key maps to a nil panel — intentional fail-fast for wiring mistakes; softer failure would need constructor/API design [`internal/app/shell.go`] — deferred
- [x] [Review][Defer] `gotoReview` prelude calls `clearReviewUndoIfLeftReview` before `setNavSelection` / `selectPanel` — behavior change vs prior ordering; no automated test asserts undo-stack parity across Rejected → Review navigation [`internal/app/shell.go`] — deferred, manual QA / follow-up test

## Dev Notes

### Technical requirements

- **Predicate parity is non-negotiable:** compose list SQL with `ReviewBrowseBaseWhere` + `ReviewFilterWhereSuffix`; do not duplicate `rejected` / `deleted_at_unix` clauses by hand.
- **Story scope:** **Grid + badges + paging + failed decode UX + distinct pending-decode affordance (AC6) + image-first cell density (AC4) + Story 2.11 / NFR-01 readability evidence (AC5)**; **loupe**, **keyboard 1–5 rating**, and **reject/restore actions** are **not** required here as product flows (Epic Stories **2.4**–**2.6**), though **Rejected/Hidden** must **reuse** this grid without forking paging, cache, or error sanitization. Click-to-open may be stubbed only if needed to avoid false FR-09 claims.
- **Tags in cells:** FR-07 also mentions tags; this story’s epic title emphasizes **rating + reject badges**. Omit tag chips until **2.5** unless a **non-deceptive** static hint is trivial.

### Architecture compliance

- **§3.8:** Paged DB queries + on-disk thumbnail cache; custom layout acceptable for grid cell.
- **§3.8.1 Layout ↔ async coherence (image-first):** Any numeric layout budget in Story **2.11** must name the **same box** and **lifecycle moment** as the grid implementation; **UX-DR17** (heavy work off UI thread, `fyne.Do` for UI); **stale async work** prevented via epoch/cancellation so recycled rows do not show wrong thumbnails; reserve stable geometry where Dev Notes specify so **UX-DR18** does not fight the pipeline.
- **§3.12 step 5 / §4.5:** Review list respects default browse rules via shared `ReviewBrowseBaseWhere`.
- **§5.2:** No SQL inside Fyne widgets—queries live in `internal/store`; app binds results to view models.

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3** — prefer standard widgets + `canvas.Image` or `widget.Icon` for thumbnails; follow existing theme roles from `internal/app/theme.go` where applicable.

### File structure requirements

- **Extend:** `internal/app/review.go` (grid body, `libraryRoot` parameter); `internal/app/shell.go` (pass `libraryRoot` into `NewReviewView`).
- **Extend / add:** `internal/store/review_query.go` (+ tests) for paged list; `internal/app/review_grid.go`, `internal/app/thumbnail_disk.go` (+ tests) for grid + cache (no Fyne imports in `store`).

### Testing requirements

- Integration-style store tests with temp SQLite + known `assets` rows (pattern: `review_query_test.go`).
- If headless Fyne grid tests are costly, prioritize **store contracts** + **thin UI glue** tests; document manual QA for scroll/paging in Dev Agent Record.

### Previous story intelligence (Story 2.2)

- **`ReviewBrowseBaseWhere` + `ReviewFilterWhereSuffix` + contract tests** already guard count/list drift—extend them when adding list.
- **`NewReviewView` / `newReviewViewWithoutDB`:** preserve **honest** behavior when `db == nil` (strip + count label rules); grid region should degrade gracefully (e.g. message, not panic).
- **Filter callbacks** already run on Fyne thread; grid refresh must follow the same pattern.
- **Party-mode resolution:** list query **must** reuse **`ReviewBrowseBaseWhere`**; Story 2.2 explicitly deferred the thumbnail grid to this story.

### Git intelligence summary

- Recent history is sparse in this clone; rely on **Story 2.2** artifact and `internal/store/review_query.go` as the source of truth for filter SQL.

### Latest technical information

- **SC-3 / FR-10:** “within1 second” targets **local single-user** review feedback; avoid synchronous full-library decode on the UI thread.
- **Fyne:** prefer loading images asynchronously for large directories; verify widget refresh happens on the main thread.

### Project structure notes

- No `project-context.md` found in repo; use architecture + PRD + this story as the contract.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 2, Story 2.3]
- [Source: _bmad-output/planning-artifacts/PRD.md — SC-3 (item 3), FR-07, FR-10]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.8 Desktop UI / grid performance, §3.8.1 layout ↔ async coherence, §4.2–4.3 errors/logging, §4.5 default queries, §5.1–§5.2 boundaries]
- [Source: _bmad-output/planning-artifacts/initiative-fyne-image-first-phase1-party-2026-04-15.md — Phase 1 UX-DR16–DR19 ↔ implementation synthesis (if present)]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — §2.5 grid uniform thumbnail; Component Strategy **Thumbnail grid cell** (rating badge, reject indicator, pending/failed decode)]
- [Source: _bmad-output/implementation-artifacts/2-2-filter-strip.md — filter parity, `ReviewBrowseBaseWhere`, degraded DB behavior]
- [Source: _bmad-output/implementation-artifacts/2-11-layout-display-scaling-gate.md — NFR-01 evidence / AC5 readability recording]
- [Source: _bmad-output/implementation-artifacts/nfr-01-layout-matrix-evidence.md — numeric layout evidence]
- [Source: internal/store/review_query.go — `ReviewBrowseBaseWhere`, `ReviewFilterWhereSuffix`, `CountAssetsForReview`]
- [Source: internal/app/review.go — filter strip + grid]
- [Source: internal/app/shell.go — `NewMainShell`, Review wiring]
- [Source: internal/store/migrations/001_initial.sql — `assets.rejected`, paths]
- [Source: internal/store/migrations/003_review_filters.sql — `assets.rating`]

## Change Log

- 2026-04-15: Party mode **dev** session **2/2** (2.3) — simulated **Amelia / Sally / Winston / John** (dev hook deepen). **Amelia** pushed past scroll-only UX-DR18: **DRY** `syncGridScrollVisible` + **Rejected** count-error path hid **Back to Review** (dead-end shell). **Winston** countered Fyne **List** has no **Disable** in v2.7.3 — dropped Enable/Disable, kept scroll staging. **Sally** insisted **escape hatch** over perfect empty-state minimalism. **John** accepted as **FR-29 navigation hygiene** (not a new numbered AC). **Amelia** then surfaced **latent flake**: stale thumbnail goroutines after `reset` could **`SetText` into torn-down cells** (`TestRejectedView_closedDB` panic) — fixed with **`thumbGen`** on `reset`/`invalidatePages`. `go test ./...` green; story/sprint remain **done**.
- 2026-04-15: Party mode **dev** session **1/2** (2.3) — simulated **Amelia / Sally / Winston / John**; **Amelia** challenged prior “zero Fyne rows” closure: `widget.List` length 0 still left a **visible `Scroll` slab** under empty-state copy (**Review** + **Rejected**). **Sally** noted layout shift when transitioning0→1 assets; **Winston** required **parity** on both shells. **John** framed as UX-DR18 **stage ownership** (empty block owns attention). Applied `gridScroll.Hide/Show` in `refreshReviewData` / `refreshRejectedData` + count-error path; `go test ./...` green; story/sprint remain **done**.
- 2026-04-15: BMAD **dev-story** workflow (Composer) — re-verified AC1–AC6 and extended guardrails against `internal/app/review_grid.go`, `internal/store/review_query.go`, and tests; `go test ./...` and `go build .` green; no code changes; story Status **done** and sprint **`2-3-thumbnail-grid-rating-badges`** unchanged **done**.
- 2026-04-15: Party mode **create** session **2/2** (2.3) — simulated **Mary / John / Sally / Amelia**; challenged session **1/2** “paper” **UX-DR18** guardrail: no automated check that **zero count ⇒ zero Fyne rows**, and **epics §2.3** still lacked **AC6** / zero-row language read by PMs. Added **`reviewGridListRowCount`** + **`TestReviewGridListRowCount`**, synced **`epics.md`** Story **2.3** ACs, story **Risk (5)** epic drift; `go test ./...` green; sprint **`2-3-thumbnail-grid-rating-badges`** remains **done**.
- 2026-04-15: Party mode **create** session **1/2** (2.3) — simulated **John / Paige / Sally / Winston**; challenged prior **dev-only** closure: epic ACs never numbered **pending vs failed** decode or **Rejected** list parity. Added **AC6**, **UX-DR18** shell coordination guardrail, **Risks & DoD**, rejected-list task bullet; sprint **`2-3-thumbnail-grid-rating-badges`** set **done** to match story **Status** (implementation already satisfied AC6).
- 2026-04-15: BMAD **create-story** workflow — merged spec: AC **1–5** aligned to `epics.md` Story 2.3; **AC5** → tasks + refs for Story **2.11** / NFR-01; architecture **§3.8.1** + initiative doc reference; extended **list/bind** guardrails; **Status** remains **done** per merge rules; sprint **`2-3-thumbnail-grid-rating-badges`** set **ready-for-dev** for BMAD tracking.
- 2026-04-14: Party mode **dev** session **2/2** (2.3, deepen) — simulated roundtable; challenged session **1/2** focus on **failure** paths: **pending decode** still used an empty `canvas.Image`, blurring UX-DR3 **pending** vs **failed**. Applied **`theme.MediaPhotoIcon()`** as `Resource` while `WriteThumbnailJPEG` runs; clear `Resource` before binding cached `File`; **`slog.Error`** for page queries now includes **`page`** + **`rejected_mode`**; `go test ./...` green; story remains **done**.
- 2026-04-14: Party mode **dev** session **1/2** (2.3) — simulated roundtable; **cached failed grid page** in `reviewAssetGrid` so a broken list query logs **once** per page and does not re-hit SQLite on every row bind/scroll; `bindGridRow` no longer duplicates `slog.Error`; sprint comment added; story remains **done**.
- 2026-04-14: BMAD dev-story workflow (re-run) — `go test ./...` and `go build .` green; acceptance criteria and tasks re-verified against codebase; no code changes required; story/sprint remain **done**.
- 2026-04-13: Dev-story workflow — `go test ./...` and `go build .` verified green; all acceptance criteria and tasks confirmed satisfied in codebase; status set to **review** for BMAD handoff (sprint synced).
- 2026-04-13: Party mode session **2/2** (dev hook, deepen) — uniform **page-query** failure across row cells, **theme.ErrorIcon** + shared user copy, **one** `slog.Error` per failed fetch, `TestReviewGridUserFacingMessagesSanitized`; status **done**, sprint synced.

## Dev Agent Record

### Agent Model Used

**2026-04-15:** BMAD dev-story workflow — verification pass (Composer): full test suite + root build, AC spot-check vs grid/store/thumbnail code paths; story already **done** with all tasks complete.

Party mode (simulated headless, automated): **Amelia (Dev)**, **Sally (UX)**, **Winston (Architect)**, **John (PM)** — round9 (**dev** session **2/2**, 2026-04-15). **Amelia (Dev)**, **Sally (UX)**, **Winston (Architect)**, **John (PM)** — round8 (**dev** session **1/2**, 2026-04-15). **Mary (Analyst)**, **John (PM)**, **Sally (UX)**, **Amelia (Dev)** — round7 (**create** session **2/2**, 2026-04-15). **John (PM)**, **Paige (Tech Writer)**, **Sally (UX)**, **Winston (Architect)** — round6 (**create** session **1/2**, 2026-04-15). Earlier rounds: **John**, **Sally**, **Winston**, **Mary** — round 1; **John**, **Sally**, **Winston**, **Murat (TEA)** — round 2. Session **2/2** deepen: **Amelia (Dev)**, **Sally**, **Winston**, **Murat** — round 3. Session **1/2** (2026-04-14) dev hook: **Amelia**, **Sally**, **Winston**, **Mary** — round 4. Session **2/2** (2026-04-14) dev hook deepen: **Amelia**, **Sally**, **Winston**, **John** — round 5; roster from `_bmad/_config/agent-manifest.csv`. Implementation: Composer agent.

**2026-04-14:** BMAD dev-story workflow executed on this artifact; verification-only pass (Composer).

### Debug Log References

### Completion Notes List

- **2026-04-15 (party dev 2/2):** `reviewAssetGrid.syncGridScrollVisible` centralizes scroll show/hide; **Rejected** count-query failure keeps **Back to Review** (medium); **`thumbGen`** invalidates async thumbnail completions on `reset` / `invalidatePages`; `TestRejectedView_closedDB_degradedSuffix_ordersCollectionsBeforeTags` asserts escape hatch.
- **2026-04-15 (party dev 1/2):** UX-DR18 — hide thumbnail **`Scroll`** when `n==0` or count query error (`internal/app/review.go`, `internal/app/rejected.go`); empty / error messaging is not paired with inert grid viewport.
- **2026-04-15 (BMAD dev-story, Composer):** Confirmed all Tasks/Subtasks **[x]**; `go test ./...` and `go build .` green; AC1 paging/async thumbs + cache, AC2 row data for badges, AC3/AC6 pending (`theme.MediaPhotoIcon`) vs failed (`ErrorIcon` + copy), AC4 image-first cell layout, AC5 trace via `_bmad-output/implementation-artifacts/nfr-01-layout-matrix-evidence.md`, UX-DR18 `reviewGridListRowCount(total<=0)==0`, store list/count parity tests present; no implementation gaps found — Status remains **done**.
- **2026-04-15 (create party 2/2):** Epic §2.3 AC parity (`epics.md`); **`reviewGridListRowCount`** + **`TestReviewGridListRowCount`** for UX-DR18 zero-row contract; story **Risk (5)**; `go test ./...` green.
- **2026-04-15 (create party 1/2):** Spec-only alignment — **AC6**, **Risks & DoD**, rejected-list task, **UX-DR18** coordination note; sprint **`2-3-thumbnail-grid-rating-badges`** → **done**; `go test ./...` green.
- `ListAssetsForReview` + `ReviewGridRow` with stable sort and LIMIT/OFFSET paging; tests assert parity with `CountAssetsForReview` and paging edges.
- Review body uses `widget.List` with **four thumbnail cells per row** (virtualized rows, not a single column), DB page cache (`reviewGridPageSize`), thumbnails via `WriteThumbnailJPEG` + `ThumbnailCachePath`, async completion guarded with `sync.Map` + `fyne.Do`.
- Failed decode / missing file: user-facing copy in cell (`Can't preview — …`), not raw errors.
- **Session 2 hardening:** grid **page/SQL** failures use sanitized cell copy + `slog.Error`; store tests cover invalid list pagination args; code comment documents **main-thread** paged query during list bind (small `LIMIT` tradeoff).
- **Session 2/2 deepen:** page fetch errors apply to **all** cells in the visible row (not column-0 only); **ErrorIcon** + shared constants for decode vs page copy; **single** log line per bind row on query failure; `TestReviewGridUserFacingMessagesSanitized` guards against SQL-ish leakage in UI strings.
- **Session 2/2 (2026-04-14) dev hook — pending vs failed:** While thumbnails decode, cells show **`MediaPhotoIcon`** (Fyne `Resource`) instead of a blank image region; success path clears `Resource` before `File = cacheAbs`. Page-query `slog.Error` adds **`page`** and **`rejected_mode`** for supportability without UI leaks.
- **Session 1/2 (2026-04-14) dev hook:** **page-level** failure cache (`pageFailed` + `errReviewGridPageFailed`) so scroll/recycle does not **re-query** a known-bad page or spam logs; real driver error logged **once** in `ensurePageLocked`; UI still uses sanitized `reviewGridMsgPageLoadFail`.
- **Manual QA:** scroll a large library; confirm cache fills under `.cache/thumbnails/` and filter changes reset the grid.
- **2026-04-13 (Composer):** `go test ./...` and `go build .` green after session 2/2; story **done**, sprint **done**.
- **2026-04-14 (Composer):** Dev-story workflow — full test suite + root build green; AC1–AC4 and task list confirmed in `internal/store/review_query.go`, `internal/app/review_grid.go`, `internal/app/thumbnail_disk.go`; status unchanged **done**.

### File List

- `internal/store/review_query.go` — `ReviewGridRow`, `ListAssetsForReview`
- `internal/store/review_query_test.go` — list/count parity + paging tests
- `internal/app/review.go` — `NewReviewView(..., libraryRoot)`, grid wiring, grid scroll visibility (empty / count error)
- `internal/app/review_grid.go` — grid list, cells, badges, async thumbs
- `internal/app/review_grid_test.go` — badge helpers, list row-count regression (UX-DR18)
- `internal/app/thumbnail_disk.go` — cache path + JPEG resize/write
- `internal/app/thumbnail_disk_test.go` — cache path, error path, round-trip
- `internal/app/shell.go` — pass `libraryRoot` into Review
- `internal/app/review_test.go` — updated `NewReviewView` signature

### Party mode — round 1 (create focus, simulated)

**John (PM):** AC2's "1 second" is about *perceived* correctness, not benchmarking every decode. Ship with synchronous SQLite page fetch on the UI thread for each visible window, async only for pixmap work—otherwise we scope-creep into perf stories.

**Sally (UX):** Reject must not be color-only; pair **"Hidden"** with `WarningImportance`. Failed decode needs a **sentence**, not an error code—photographers should know *what* failed without developer jargon.

**Winston (Architect):** List SQL must reuse `ReviewBrowseBaseWhere` + `ReviewFilterWhereSuffix` with identical arg order to count. Thumbnail cache path stays under `.cache/thumbnails/` with sharding by hash prefix; bind async completions with a **generation map** keyed by the cell image pointer so recycled list rows don't flash wrong assets.

**Mary (Analyst):** Risks: (1) filter/count vs grid drift—mitigate with parity tests; (2) goroutine leaks—none if Fyne callbacks are short; (3) Windows `rel_path`—use `filepath.FromSlash` at join sites only, don't mutate store rows.

**Orchestrator synthesis:** Implement store list + tests first, then disk thumbnail helper + tests, then Fyne list with paging and `fyne.Do` completion. Update story checkboxes and sprint status to **done** after `go test ./...` is green.

### Party mode — round 2 (create focus, simulated — challenges session 1)

**John (PM):** Round 1 under-specified **failure UX** for the *other* thumbnail path: broken **page fetch** isn’t the same as broken **decode**. AC3 covered decode only; we need an explicit AC or task so QA doesn’t accept SQL fragments in the grid.

**Sally (UX):** I’m with John — `err.Error()` in a label is a **trust killer**. Photographers shouldn’t see `database disk image is malformed`. Pair a calm sentence with **logging** for support.

**Winston (Architect):** I’ll push back on Sally slightly: **main-thread** `ListAssetsForReview` during `List` bind is still the right MVP tradeoff — prefetch would need cancellation and a second cache. But we should **document** that choice beside `ensurePageLocked` so nobody “fixes” it naively.

**Murat (Test Architect):** Two gaps: (1) **no test** that invalid `LIMIT`/`OFFSET` fail fast — public store API should be self-defending; (2) **no automated assertion** on the SQL error string absence in UI — acceptable if we fix the code path and log instead.

**Orchestrator synthesis:** Add **AC4** (sanitized grid error + slog). Add `TestListAssetsForReview_invalidLimitOrOffset`. Replace `bindError` copy + `slog.Error`. Comment the UI-thread page fetch. Re-run `go test ./...`; keep sprint/story **done**.

### Party mode — round 3 (session 2/2, dev hook — challenges round 2 closure)

**Amelia (Dev):** Round 2 closed on “sanitized string + slog,” but the **grid** still looked broken: on `ListAssetsForReview` failure we cleared three cells and stuffed the apology into **cell0 only**. That reads like a layout bug, not intentional UX. Log **once per row bind** and fan out UI state to every cell in the row.

**Sally (UX):** UX-DR3 asked for **failed decode**: icon **or** tooltip; we had caption but **no** icon. Photographers scan shapes — add **ErrorIcon** so the failure state matches the spec’s “affordance,” not just a paragraph in the stack.

**Winston (Architect):** Pushback on Sally: Fyne 2.7.3 here does not use a **tooltip** API elsewhere in the app; **icon + label** is the honest MVP. Skip ad-hoc hover chrome until we standardize a pattern.

**Murat (TEA):** Agree with Amelia on **log spam** and column-0-only failure. Add a **regression** that the two canonical user strings never contain `sqlite` / `near ` / `syscall` — cheap guardrail since we will not parse Fyne trees in CI.

**Orchestrator synthesis:** Refactor to `showUserFailure(msg)` + constants; stack **ErrorIcon** + label; on page error **log once** in `bindGridRow` then call `showUserFailure` for **every** cell in the row; add `TestReviewGridUserFacingMessagesSanitized`; mark story/sprint **done** after green `go test ./...`.

### Party mode — round 4 (session 1/2, dev hook — challenges round 3)

**Amelia (Dev):** Round 3 fixed **column coverage** and duplicate `slog` between cells in one bind, but **`ensurePageLocked` still re-queries** whenever `pages[pageIdx]` is empty. Fyne **re-binds** rows on scroll; you **hammer SQLite** and can still get **log noise** if anything logs per bind. Move **one** `slog.Error` to the **first** query failure and **memoize failure** per `pageIdx` until `reset` / `invalidatePages`.

**Sally (UX):** Users won’t read logs — they need the grid to **stay** in a calm failure state without **flickering** or feeling like it is constantly “retrying.” Caching failure matches **stable** error tone (UX-DR18 spirit).

**Winston (Architect):** Do not persist the raw `error` in view state — that invites accidental leaks in dumps. Track **`pageFailed`** + sentinel `errReviewGridPageFailed` for control flow; keep the **first** underlying error **only** in the log line.

**Mary (Analyst):** Risk: a **transient** I/O blip looks **permanent** until the user changes filters or something calls `invalidatePages`. That is acceptable if copy promises **user-triggered recovery** (“try changing the filter”); note the tradeoff in code.

**Orchestrator synthesis:** Add `pageFailed`, sentinel error, log once in `ensurePageLocked`, remove duplicate log from `bindGridRow`, clear `pageFailed` on `reset` / `invalidatePages`; run `go test ./...` + `go build .`; story/sprint stay **done**.

### Party mode — round 5 (session 2/2, 2026-04-14 — challenges rounds 3–4)

**Amelia (Dev):** Rounds 3–4 hardened **error** and **SQLite churn**; the **happy path** still left **`canvas.Image` empty** until the goroutine finished. Fast scroll reads as **flicker** or “missing photo.” Bind **`theme.MediaPhotoIcon()`** as `Resource` for the pending window; **`Resource = nil`** before applying **`File`** on success so Fyne doesn’t keep the placeholder.

**Sally (UX):** I’ll nitpick **image-first**: an icon is **chrome**. **`MediaPhotoIcon`** is the lightest-weight **pending** signal we get without a second widget layer—better than a **dead** rectangle that users confuse with **failed decode** (**ErrorIcon** path stays distinct).

**Winston (Architect):** Fyne **`File` vs `Resource`** is exclusive in practice; the success branch must **zero `Resource`** first. **`clear()` / `showUserFailure`** already nil out both—no new state machine.

**John (PM):** This doesn’t change AC text—it closes the **UX-DR3 pending** gap the numbered ACs imply but didn’t spell out. **Observability:** add **`page`** to the page-query log so support can correlate **scroll position** with failures without opening the UI.

**Orchestrator synthesis:** Implement pending **`MediaPhotoIcon`** in `bindRow`, structured **`slog`** fields in `ensurePageLocked`, re-run **`go test ./...`**; story/sprint remain **done**.

### Party mode — round 6 (session 1/2, 2026-04-15 — **create** hook, challenge dev closure)

**John (PM):** The epic’s five ACs are the contract with stakeholders. I’m fine extending the **story** with QA-facing clarity, but we should not imply **new scope**: **AC6** must describe behavior we **already** shipped in the **pending icon** fix—this is documentation debt, not a reopen.

**Paige (Technical Writer):** I’ll push back on John slightly: without a **numbered** pending-vs-failed criterion, **manual QA scripts** still anchor on epic text and miss the distinction. **AC6** plus a **Risks & DoD** block gives auditors a single landing page—epics stay lean, story carries the traceability burden.

**Sally (UX):** **UX-DR18** is mostly **Story 2.12**, but the grid must not **lie** when count is zero—**zero rows**, empty block upstack. That’s not “more empty-state work”; it’s a **coordination guardrail** so we don’t regress into placeholder tiles.

**Winston (Architect):** The biggest spec hole was **Rejected mode**: code paths call **`ListRejectedForReview`**, but tasks read like **default Review** only. Future refactors could drop parity. Explicit **store + grid** bullet ties **2.6** integration to **2.3** performance rules without duplicating SQL.

**Orchestrator synthesis:** Add **AC6** (pending vs failed), **UX-DR18** coordination line, **Risks & DoD**, rejected-list task bullet; set sprint **`2-3-thumbnail-grid-rating-badges`** to **done** to match story **Status**; run **`go test ./...`** to confirm green (no code change required for this round).

### Party mode — round 7 (create focus, session **2/2**, 2026-04-15 — challenges round 6)

**Mary (Analyst):** Round 6 nailed **story** traceability, but **`epics.md` §2.3** is what execs skim — it still read like **five** ACs with **no** pending-vs-failed line. That is **requirements debt**: QA files bugs against the epic PDF, not our implementation artifact.

**John (PM):** I’ll push back lightly: we shouldn’t **inflate** the epic every time a story adds nuance. **Unless** the nuance is **user-visible** and **shippable** — pending vs failed **is**. Zero-row grid when count=0 is the same: it prevents “blank grid” **false affordance**. Epic gets **two short bullets**, not a novel.

**Sally (UX):** Session 1/2 wrote **UX-DR18** in prose; without a test, the next refactor can “simplify” row math and bring back **one empty list row** of dead cells. **Automate the invariant:** `total≤0 ⇒ list length 0`.

**Amelia (Dev):** `widget.NewList` closure duplicated `(total+columns-1)/columns` next to a `total==0` guard. Extract **`reviewGridListRowCount`**, unit-test **0, negative, partial last page**. No Fyne tree walk. **`internal/app/review_grid.go`**.

**Orchestrator synthesis:** Sync **`epics.md`** Story **2.3** with **AC6-class** pending/failed wording + **zero-count grid** bullet; add **`reviewGridListRowCount`** + **`TestReviewGridListRowCount`**; extend story **Risks** with epic-drift item; append task checkbox; **`go test ./...`**; sprint stays **done**.

### Party mode — round 9 (dev focus, session **2/2**, 2026-04-15 — challenges round 8)

**Amelia (Dev):** Round 8 fixed the **scroll slab**, but we duplicated **four** `if gridScroll != nil` branches and **Rejected** still **hid** `backToReview` when `CountRejectedForReview` failed — that’s a **trap** if shell nav is overlooked. Centralize **`syncGridScrollVisible`** and **show** the back button on count error.

**Sally (UX):** I’ll fight **Winston** on “just use nav”: photographers in a bad DB state are already stressed — **Back to Review** must read as an **obvious exit**, not a Easter egg in the rail.

**Winston (Architect):** I tried **`list.Disable()`** for UX-DR19 — **Fyne 2.7.3 `widget.List` has no `Disable`**. Don’t fake a11y with APIs that don’t exist; **hidden scroll** is the real containment.

**John (PM):** Not a new epic AC — it’s **recovery** inside the same FR-29 surface. Ship the hatch; keep stakeholder-facing **epics** unchanged.

**Orchestrator synthesis:** Implement **`syncGridScrollVisible`**; **Rejected** `qerr` branch: **`backToReview.Show`**, medium importance, **`onGotoReview`**; extend **closed-DB** test. While validating, **`TestRejectedView_closedDB`** panicked — root cause **stale async thumbnail** after **`reset`**; add **`thumbGen`** + check in **`fyne.Do`**. **`go test ./...`** green.

### Party mode — round 8 (dev focus, session **1/2**, 2026-04-15 — challenges round 7 / list-only UX-DR18)

**Amelia (Dev):** **`reviewGridListRowCount(0)==0`** is necessary but not sufficient: Fyne still paints a **`Scroll`** frame under the empty-state block. That is a **false grid affordance** and wastes vertical space at NFR-01 min sizes. Hide the scroll when **`n==0`** and when **`Count*ForReview` errors**; wire **`gridScroll`** before the first **`refreshAll`** so the initial frame is correct.

**Sally (UX):** I’ll push back on **jank**: hiding the scroll when the user goes **1 → 0** will **collapse** the body and shift controls. Acceptable if we only hide when the **empty row** is visible — just avoid odd layout flicker. Also ensure **Rejected** matches; asymmetry reads as a product bug.

**Winston (Architect):** Keep **one** scroll object per surface; drive visibility only from the **same refresh** that owns **`grid.reset`**. No new goroutines—this is synchronous shell state.

**John (PM):** No new numbered AC: epic **2.3** already says the shell owns messaging at zero count. This is **implementation debt** closing the gap between “no rows” and “no grid stage.”

**Orchestrator synthesis:** Add **`var gridScroll *container.Scroll`**, assign **`container.NewScroll(grid.canvasObject())`** before **`refreshAll()`**, **`Hide()`** when **`qerr`** or **`n==0`**, **`Show()`** when **`n>0`** in **`review.go`** and **`rejected.go`**; extend story **UX-DR18** guardrail prose; **`go test ./...`**; sprint **`2-3-thumbnail-grid-rating-badges`** stays **done**.
