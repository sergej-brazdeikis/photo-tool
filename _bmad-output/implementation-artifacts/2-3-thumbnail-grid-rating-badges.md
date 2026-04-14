# Story 2.3: Paged thumbnail grid with rating and reject badges

Status: done

<!-- Ultimate context engine analysis completed — comprehensive developer guide created. -->

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **a dense grid that shows state at a glance**,  
So that **I can triage quickly**.

**Implements:** FR-07 (display tags/ratings context); supports FR-08/FR-29 display; UX-DR3.

## Acceptance Criteria

1. **Given** filtered assets, **when** the grid loads, **then** thumbnails load incrementally (**paged or windowed**) without loading **all** pixmaps for the full result set at once (architecture §3.8 **Grid performance**).
2. **Given** an asset with **rating** or **reject** state in the DB, **when** its cell is shown in the grid, **then** **badges** reflect that DB state within the PRD **1 second** guideline for **single-user, local-library** use (PRD **SC-3** item 3; **FR-10** / **FR-07** display baseline). *Interpretation for this story:* after a filter change or initial load, visible cells must show **correct** `rating` / `rejected` from the same query row without stale placeholders beyond ~1s under normal local SQLite + decode latency; full **keyboard rating** persistence is Story **2.4**, but grid **display** must be truthful to DB.
3. **Given** **decode failure** for an asset file, **when** the cell renders, **then** the user sees a **failed-decode** affordance (**placeholder + short explanation**), not a silent blank—per **UX-DR3** / UX component strategy (**failed decode**: icon + tooltip or caption).
4. **Given** a **paged grid query** error (SQLite / I/O), **when** the list tries to bind a row, **then** the cell shows a **short, non-technical** explanation (no driver/SQL text in the UI); details go to **structured logs** (`slog`) for diagnosis—same spirit as AC3 for “no raw dumps” in the surface.

## Tasks / Subtasks

- [x] **Wire `libraryRoot` into Review** (AC: 1, 3)
  - [x] Extend `NewReviewView` / `NewMainShell` so Review receives `libraryRoot` (same as `main.go` → `NewMainShell(win, db, root)`), enabling absolute paths from `assets.rel_path` and `{libraryRoot}/.cache/thumbnails/` per architecture §3.8.
- [x] **Store: paged asset list with filter parity** (AC: 1, 2)
  - [x] Add `ListAssetsForReview` (name may vary) that selects rows needed for grid cells (at minimum: `id`, `rel_path`, `rating`, `rejected`, `mime`, `width`, `height`, `capture_time_unix`—trim only if proven unused).
  - [x] SQL **must** use `WHERE ` + `store.ReviewBrowseBaseWhere` + suffix from `store.ReviewFilterWhereSuffix(filters)` with **identical argument order** to `CountAssetsForReview` (reuse Story 2.2 contracts; extend `review_query_test.go` if list drifts).
  - [x] `ORDER BY` must be **stable** (recommend `capture_time_unix DESC`, `id DESC` tie-break) so paging is deterministic.
  - [x] Accept `LIMIT`/`OFFSET` (or keyset) parameters; **do not** `SELECT *` the entire library for the grid.
- [x] **Thumbnail pipeline: incremental decode + cache** (AC: 1, 3)
  - [x] Follow architecture §3.8: **disk cache** under `{libraryRoot}/.cache/thumbnails/`; avoid holding full decoded pixmaps for off-screen pages.
  - [x] Decode **off the UI goroutine** when work is non-trivial; apply results with Fyne-safe scheduling (`fyne.Do` after `WriteThumbnailJPEG`).
  - [x] On irrecoverable decode error, surface **UX-DR3** failed state in the cell (icon + label/tooltip text—not raw error dumps).
- [x] **Grid UI: dense, scannable cells** (AC: 1–4)
  - [x] Replace the Story 2.2 **placeholder** body (`gridHint` in `internal/app/review.go`) with a real grid/list that **virtualizes** or **pages** (Fyne `List` rows of `reviewGridColumns` cells + DB-backed paging in pages of `reviewGridPageSize`).
  - [x] Cell layout aligns with UX **Thumbnail grid cell**: uniform thumbnail area (letterboxed via `canvas.ImageFillContain`), **rating badge**, **reject indicator** (caution semantic when `rejected=1`; **not** color-only—`Hidden` label + `WarningImportance`).
  - [x] **Default Review** list excludes `rejected=1` (same as count); reject badge still **implemented** for correct rendering if the cell is reused when `rejected=1` appears in row data later.
- [x] **Filter integration** (AC: 1, 2)
  - [x] On any filter strip change, **reload** the grid from page 0 using the same `domain.ReviewFilters` mapping as `buildFilters()` in `review.go` (no second filter definition).
  - [x] Keep **live count** + grid in sync (count + `reset` share one refresh path).
- [x] **Tests** (AC: 1–4)
  - [x] Store: table-driven tests for list SQL—same row set as count for small fixtures; paging boundaries (empty page, last page partial).
  - [x] Store: `ListAssetsForReview` rejects invalid `limit` / `offset` (guardrails for callers).
  - [x] Optional: lightweight test for thumbnail error path (invalid bytes / missing file) if injectable without heavy GUI.
  - [x] `go test ./...` remains green.

## Dev Notes

### Technical requirements

- **Predicate parity is non-negotiable:** compose list SQL with `ReviewBrowseBaseWhere` + `ReviewFilterWhereSuffix`; do not duplicate `rejected` / `deleted_at_unix` clauses by hand.
- **Story scope:** **Grid + badges + paging + decode failure UX**; **loupe**, **keyboard 1–5 rating**, and **reject action** are **not** required here (Epic Stories **2.4**–**2.6**). Click-to-open may be stubbed only if needed to avoid false FR-09 claims.
- **Tags in cells:** FR-07 also mentions tags; this story’s epic title emphasizes **rating + reject badges**. Omit tag chips until **2.5** unless a **non-deceptive** static hint is trivial.

### Architecture compliance

- **§3.8:** Paged DB queries + on-disk thumbnail cache; custom layout acceptable for grid cell.
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
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.8 Desktop UI / grid performance, §4.5 default queries, §5.1–§5.2 boundaries]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — §2.5 grid uniform thumbnail; Component Strategy **Thumbnail grid cell** (rating badge, reject indicator, pending/failed decode)]
- [Source: _bmad-output/implementation-artifacts/2-2-filter-strip.md — filter parity, `ReviewBrowseBaseWhere`, degraded DB behavior]
- [Source: internal/store/review_query.go — `ReviewBrowseBaseWhere`, `ReviewFilterWhereSuffix`, `CountAssetsForReview`]
- [Source: internal/app/review.go — filter strip + grid]
- [Source: internal/app/shell.go — `NewMainShell`, Review wiring]
- [Source: internal/store/migrations/001_initial.sql — `assets.rejected`, paths]
- [Source: internal/store/migrations/003_review_filters.sql — `assets.rating`]

## Change Log

- 2026-04-13: Dev-story workflow — `go test ./...` and `go build .` verified green; all acceptance criteria and tasks confirmed satisfied in codebase; status set to **review** for BMAD handoff (sprint synced).
- 2026-04-13: Party mode session **2/2** (dev hook, deepen) — uniform **page-query** failure across row cells, **theme.ErrorIcon** + shared user copy, **one** `slog.Error` per failed fetch, `TestReviewGridUserFacingMessagesSanitized`; status **done**, sprint synced.

## Dev Agent Record

### Agent Model Used

Party mode (simulated headless, automated): **John (PM)**, **Sally (UX)**, **Winston (Architect)**, **Mary (Analyst)** — round 1; **John**, **Sally**, **Winston**, **Murat (TEA)** — round 2 (AC4 + store guards). Session **2/2** deepen: **Amelia (Dev)**, **Sally (UX)**, **Winston (Architect)**, **Murat (TEA)** — round 3 below; `_bmad/_config/agent-manifest.csv` absent in repo (roster from BMAD defaults). Implementation: Composer agent.

### Debug Log References

### Completion Notes List

- `ListAssetsForReview` + `ReviewGridRow` with stable sort and LIMIT/OFFSET paging; tests assert parity with `CountAssetsForReview` and paging edges.
- Review body uses `widget.List` with **four thumbnail cells per row** (virtualized rows, not a single column), DB page cache (`reviewGridPageSize`), thumbnails via `WriteThumbnailJPEG` + `ThumbnailCachePath`, async completion guarded with `sync.Map` + `fyne.Do`.
- Failed decode / missing file: user-facing copy in cell (`Can't preview — …`), not raw errors.
- **Session 2 hardening:** grid **page/SQL** failures use sanitized cell copy + `slog.Error`; store tests cover invalid list pagination args; code comment documents **main-thread** paged query during list bind (small `LIMIT` tradeoff).
- **Session 2/2 deepen:** page fetch errors apply to **all** cells in the visible row (not column-0 only); **ErrorIcon** + shared constants for decode vs page copy; **single** log line per bind row on query failure; `TestReviewGridUserFacingMessagesSanitized` guards against SQL-ish leakage in UI strings.
- **Manual QA:** scroll a large library; confirm cache fills under `.cache/thumbnails/` and filter changes reset the grid.
- **2026-04-13 (Composer):** `go test ./...` and `go build .` green after session 2/2; story **done**, sprint **done**.

### File List

- `internal/store/review_query.go` — `ReviewGridRow`, `ListAssetsForReview`
- `internal/store/review_query_test.go` — list/count parity + paging tests
- `internal/app/review.go` — `NewReviewView(..., libraryRoot)`, grid wiring
- `internal/app/review_grid.go` — grid list, cells, badges, async thumbs
- `internal/app/review_grid_test.go` — badge helpers
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
