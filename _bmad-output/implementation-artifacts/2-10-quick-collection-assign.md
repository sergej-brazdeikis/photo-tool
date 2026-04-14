# Story 2.10: Quick collection assign (hover) and filter workflow assign

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- create-story workflow (2026-04-13): spec from epics §Story 2.10, PRD FR-08/FR-17, UX filter strip + thumbnail cell (non-hover duplicates), brownfield `review_grid.go` / `review.go`, store `LinkAssetsToCollection`, Story 2.9 `collectionStoreErrText` + loupe out of scope for this story. -->

## Story

As a **photographer**,  
I want **to assign collections from the grid and from filters**,  
So that **organization stays fast at scale**.

**Implements:** FR-08, FR-17.

## Acceptance Criteria

1. **Thumbnail quick assign (FR-08):** **Given** the main **Review** thumbnail grid (not loupe), **when** the user uses a **quick action** on a cell (e.g. **secondary click / context menu**, **hover-revealed control**, or **small persistent affordance** on the cell—pick the pattern that fits Fyne reliably), **then** they can **choose an existing collection** and **persist** membership for **that asset** via `store.LinkAssetsToCollection` **without** opening the loupe. **And** assignment **survives** refresh: `refreshReviewData` (or equivalent) runs so counts, filter predicates, and thumbnails stay consistent with DB state.
2. **Non-hover duplicate (UX-DR3, UX component “Thumbnail grid cell”):** **Given** the same quick-assign capability, **when** the user does **not** use hover (keyboard, long session, or platform without hover), **then** the **same assign intent** is still reachable from the **filter / bulk** area—see AC3—so collection assign is **not hover-only**.
3. **Filter-workflow bulk assign (FR-17, UX filter strip):** **Given** the **filter strip** context on Review, **when** the user has **one or more assets selected** in the grid (**Cmd/Ctrl+click** bulk selection, Story 2.5 semantics), **then** they can run **assign selection to collection** (dedicated control: e.g. **dropdown of collections + button**, or **menu**—must **not** overload the **Collection** filter dropdown, which is the **browse predicate**, not the assign target). **When** assign runs, **then** **every selected asset id** is linked to the chosen collection in one store call (`LinkAssetsToCollection`) or documented batching with **one transaction per batch** if size limits arise. **Idempotency:** re-assigning an already-linked asset is a **no-op** at the data layer (existing junction idempotency).
4. **Empty / invalid selection:** **Given** bulk assign UI, **when** **no** thumbnails are selected, **then** the assign action is **disabled** or shows a **short inline hint** (mirror bulk tag empty state tone in `review.go`). **Given** a **stale collection id** (deleted elsewhere) or **DB error**, **then** the user sees **`collectionStoreErrText`**-style copy (`internal/app/collection_store_err_text.go`)—**no** raw SQLite strings for **not-found** / **FK** class errors.
5. **Scope boundaries:** **Out of scope for 2.10:** **Loupe** multi-assign and **inline new collection** remain Story **2.9** (`review_loupe.go`, `CreateCollectionAndLinkAssets`). **Upload** batch collection confirm remains **1.x** / `upload.go`. **Rejected / Hidden** (`reviewAssetGrid` with `rejectedMode`): **no** per-cell collection quick-assign and **no** new bulk “assign to collection” strip—this story applies to the **main Review** grid only (avoid silent half-behavior on Rejected).
6. **Regression — layering & predicates:** **Given** assign from grid or bulk strip, **when** complete, **then** SQL stays in **`internal/store`**; Review browse predicates and **`ReviewBrowseBaseWhere`** behavior remain unchanged except for **expected** membership-driven filter results after refresh.
7. **Empty target list / collections read failure:** **Given** the assign UI (cell flow or bulk strip), **when** there are **zero** collections **or** `ListCollections` fails during refresh, **then** assign controls are **disabled** (or hidden) with a **short hint** consistent with other Review empty/error tone—**no** panic and no raw driver text.
8. **Filter coupling & loupe interaction:** **Given** the user assigns while the **browse** filter is **“No assigned collection”** (or any predicate that excludes the asset after link), **when** `refreshReviewData` runs, **then** rows may **disappear from the grid** as an **expected** outcome (not treated as a defect). **Given** bulk assign from the strip, **when** the user triggers assign, **then** follow the same **loupe-dismiss** pattern as **Reject selected** / **Delete selected** (dismiss open loupe before writes) so the grid is the single source of truth for selection.

## Tasks / Subtasks

- [x] **Grid cell quick action** (AC: 1, 2, 5–8)
  - [x] Extend **`internal/app/review_grid.go`** (`reviewGridCell` / `bindRow`): **secondary-click** `widget.ShowPopUpMenuAtPosition` with albums from `ListCollections`; **Guard:** `openCellAssignMenu` only when **`!rejectedMode`** (AC5). **`tapLayer.SecondaryHandler`** added so primary tap/loupe behavior unchanged.
  - [x] **Discoverability:** bulk strip hint documents **right-click** path; secondary-click-only pattern recorded below (Party 2/10 agreed strip is primary non-hover discoverability).
  - [x] **Link** runs off UI thread with **`fyne.Do`** back to main thread for refresh/dialogs; **`refreshReviewData`** on success; loupe dismissed on success (consistent with bulk writes).
  - [x] **`assignCollMu`** serializes collection link writes (quick + bulk) to avoid double-assign races with loupe toggles.
- [x] **Filter-area bulk assign** (AC: 2–4, 6–8)
  - [x] **`internal/app/review.go`:** **Assign selection** row: label + **`assignTargetSel`** (real collections only, no browse sentinel) + **Assign selection to album** button under bulk tag row (after tag actions per UX-DR2 “tags” segment still above in strip).
  - [x] **`suspendAssignTargetRefresh`** when rebuilding assign-target options in **`refreshReviewData`**.
  - [x] Bulk: loupe dismiss, sorted IDs, empty selection disables button + italic hint; **`refreshReviewData`** on success; selection **preserved** (grid does not clear multi-select on data refresh except explicit user action).
  - [x] Zero collections / `ListCollections` error: assign controls disabled + hints (defer in **`refreshBulkTagUI`** + error path in **`refreshReviewData`**).
  - [x] **`collectionStoreErrText`** for link failures.
- [x] **Tests** (AC: 3–8)
  - [x] **`review_test.go`:** filter-strip tab/focus tests now assert **first three** `Select` widgets only (strip); Story 2.10 fourth `Select` lives in bulk row—**UX-DR15** QA should confirm full focus order (strip → tag entry → … → assign target).
  - [x] **Headless regression (Story 2.10):** `TestStory210_RejectedViewOmitsCollectionAssignChrome`, `TestStory210_ReviewViewBulkAssignPresentAndDisabledWithoutAlbums`, `TestStory210_ReviewViewHasAssignTargetSelect`, `TestStory210_AssignTargetOptionsExcludeBrowsePredicate` cover Rejected scope (AC5), zero-album bulk assign disable (AC7), assign-target `Select` presence, and AC3 (assign target never lists the Collection browse sentinel). **Release smoke** still recommended for Fyne popup at pointer and very large multi-select batches.
- [x] **Manual QA matrix** (AC: 1–8)
  - [x] **Interactive / release smoke** (human): quick assign with loupe closed + DB row in `asset_collections`; bulk 2+ mixed idempotency; filter “in collection X” / “No assigned collection” grid outcomes; bulk assign with loupe open; zero collections UX; delete-collection stale message — follow checklist before marking story **done** in sprint. **Structural parity** for Rejected / zero albums is covered by `TestStory210_*` in CI.

## Dev Notes

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **FR-08** | Quick assign on **thumbnail** without loupe; implementation may use **context menu** if hover is flaky in Fyne. |
| **FR-17** | Assign **selected** photos from **filter workflow** = control in Review chrome, not retargeting the filter Collection dropdown. |
| **Data** | `LinkAssetsToCollection` is **idempotent**; uses transaction in `internal/store/collections.go`. |
| **UX** | **Non-hover duplicate** required (epics UX-DR3; UX spec §Thumbnail grid cell + Filter strip “assign selection”). |

### Existing implementation to extend (do not reinvent)

- **`reviewAssetGrid`**, **`SelectedAssetIDs`**, **`toggleSelected`** — `internal/app/review_grid.go`.  
- **Review shell**, **`refreshReviewData`**, bulk tag strip — `internal/app/review.go`.  
- **`store.LinkAssetsToCollection`**, **`ListCollections`** — `internal/store/collections.go`.  
- **`collectionStoreErrText`** — `internal/app/collection_store_err_text.go` (Story 2.9).  

### Architecture compliance

- **§5.2 layering:** Fyne in **`internal/app`**; persistence in **`internal/store`**; no SQL in widgets.  
- **SQLite:** No migration expected for this story (junction table already exists, `002_collections.sql`).  

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3** (`go.mod`): `widget`, `container`, `dialog`, menus/popups as appropriate.  
- **Go** **1.25.4** per architecture / `go.mod`.  

### File structure requirements

| Area | Files (expected touch) |
|------|-------------------------|
| Grid | `internal/app/review_grid.go`, `internal/app/review_grid_test.go` (if any new pure helpers) |
| Review chrome | `internal/app/review.go`, possibly `internal/app/review_test.go` |
| Store | `internal/store/collections.go` **only** if a thin helper is justified (prefer reusing `LinkAssetsToCollection`) |

### Testing requirements

- **Store:** No new requirement unless batch API added.  
- **App:** Headless-safe unit tests only; UI assign flows otherwise **manual QA** with DoD checklist.  

### Previous story intelligence (2.9)

- **Loupe** owns **multi-assign + `CreateCollectionAndLinkAssets`**; do **not** duplicate inline create here unless PM explicitly expands scope.  
- **`suspendColSelectRefresh`** / **`refreshReviewData`** recursion: when repopulating collection **filter** options after assign, use the same **suspend** pattern as membership changes from loupe. **Extend** with **`suspendAssignTargetRefresh`** for the **assign-target** `Select` so its `OnChanged` does not call `refreshAll` mid-rebuild.  
- **Single-flight** saves and **`errors.Is` for `ErrCollectionNotFound`** before driver substring matching.  

### Git / recent patterns (brownfield)

- Bulk operations in **`review.go`** follow: read **`grid.SelectedAssetIDs()`** → **`store.*`** → **`refreshReviewData()`** (tags, reject, delete). **Mirror** for collection assign.  
- Grid selection visual: **`theme.ColorNameSelection`** on cell background when selected (`reviewGridCell.bindRow`).  

### Project structure notes

- Story keys and artifacts live under **`_bmad-output/implementation-artifacts/`**; planning refs under **`_bmad-output/planning-artifacts/`**.  
- No `project-context.md` in repo; this file + PRD + architecture are authoritative for dev-agent runs.  

### Risks & mitigations

| Risk | Mitigation |
|------|------------|
| **Hover-only** UX fatigue | Ship **bulk strip** assign (AC2–3) as the **non-hover** path. |
| **Confusing filter dropdown with assign target** | Separate **browse** Collection select from **target** collection for assign (AC3). |
| **Fyne hover reliability** | **Shipped:** **secondary-click** context menu on thumbnail (no hover dependency). |
| **Stale `ListCollections` cache** after assign | Always **`refreshReviewData`** on success (reloads collection options + grid). |
| **Second `Select` refresh recursion** | **`suspendAssignTargetRefresh`** when rebuilding assign-target options (same class of bug as `suspendColSelectRefresh`). |
| **Large bulk selection** | `LinkAssetsToCollection` holds **one transaction** for all IDs; very large selections may **stall UI**—document; chunk only if measured problem (each chunk = own tx). |
| **Discoverability vs. minimal menu** | Strip hint names right-click; follow-up: optional **⋯** cell button if users miss context menu (Party 2/10: Winston wanted visible affordance; Sally accepted strip + hint for MVP). |
| **Tab order / a11y regression** | Fourth **`Select`** is **not** in the filter strip; automated tests lock **strip-only** tab order; document full chrome order for QA (UX-DR15). |

### Latest tech notes (Fyne)

- Confirm **popup/menu** APIs for **v2.7.3** at implementation time; avoid deprecated APIs—if hover uses **`desktop.Hoverable`**, verify platform support on **tier-1** targets (macOS/Windows).  

### Definition of Done

- AC1–8 satisfied in **manual QA** on Review with a **multi-hundred** asset library smoke test.  
- FR-08 and FR-17 demonstrable without opening loupe for the **quick** path.  
- Errors use **`collectionStoreErrText`** for store-facing failures.  
- **No** new SQL in **`internal/app`**.  

## Dev Agent Record

### Agent Model Used

Composer (automated party mode dev session 1/2). Prior: create session 2/2 + `TestStory210_*` baseline.

### Debug Log References

### Completion Notes List

- **Quick assign:** Right-click / secondary-click thumbnail (main Review only) → album submenu → **dismiss loupe if open (before link)** → `LinkAssetsToCollection` → `refreshReviewData` on success. **`ListCollections` failure** uses the same user-facing line as the bulk assign strip (“Album list is unavailable — …”) for AC7 tone parity.
- **Bulk assign:** Row “Assign selection” with collection `Select` (names only) + **Assign selection to album**; mirrors reject/delete loupe dismiss; IDs sorted ascending; selection retained after success.
- **`rejected.go`:** passes `nil` cell-assign callback; no strip changes in Rejected view.
- **Tests:** `TestReviewFilterStrip_tabFocusOrder_*` assert only the **three filter-strip** selects for Story 2.2 AC4; assign `Select` is excluded from that assertion.
- **Story 2.10 CI:** `TestStory210_RejectedViewOmitsCollectionAssignChrome`, `TestStory210_ReviewViewBulkAssignPresentAndDisabledWithoutAlbums`, `TestStory210_ReviewViewHasAssignTargetSelect`, `TestStory210_AssignTargetOptionsExcludeBrowsePredicate` (helpers `collectButtons`, `findButtonByText`).

### Change Log

- 2026-04-13: Dev-story workflow — added Story 2.10 headless regression tests in `review_test.go`; consolidated Manual QA tasks (CI structural coverage + release smoke note).
- 2026-04-13: Party dev1/2 — `TestStory210_AssignTargetOptionsExcludeBrowsePredicate` (AC3); quick-assign album-list error copy aligned with bulk strip (`review.go`).
- 2026-04-13: Party dev2/2 — AC8: quick-assign (context menu) dismisses loupe **before** link write, matching bulk assign / reject / delete (`review.go`).

### File List

- `internal/app/review_grid.go`
- `internal/app/review.go`
- `internal/app/rejected.go`
- `internal/app/review_test.go`

### Party mode — session 2/2 (create + dev synthesis, headless)

Session 1 already pinned AC7–AC8, `suspendAssignTargetRefresh`, and rejected scope. Session 2 **challenged** those by asking whether tab-order tests should treat the new `Select` as part of the “strip” (no), whether quick-assign must add a visible **⋯** (deferred: strip + right-click + hint), and how loupe dismiss should relate to **per-cell** assign (**final:** dismiss **before** the link write — AC8 parity with bulk; see dev session 2/2 below).

### Party mode — dev session 2/2 (automated headless)

**Roster:** manifest still absent; simulated dev-focused round (challenge prior dev1/2: AC8 wording vs code).

- **Winston (Architect):** Dev 1/2 said “dismiss on success” for quick assign to avoid stale loupe. That contradicts AC8’s explicit **before writes** parity with reject/delete/bulk. Bulk path already clears `dismissLoupe` before the goroutine; per-cell must match or the spec is lying.
- **Sally (UX):** Dismissing before the async link means the loupe can close a moment before the DB confirms — same as bulk assign on failure. Users already accept that for “Assign selection”; right-click assign should feel like the same family of actions.
- **Amelia (Dev):** Move dismiss into the menu item handler (main thread), remove the post-success dismiss block, keep `refreshReviewData` on success only. No new headless test unless we later expose a seam; behavior is parity with existing bulk path.
- **Murat (Test):** We still lack an automated loupe-dismiss ordering assertion; document AC8 parity in code comment and story changelog. Optional follow-up: inject `dismissLoupe` spy in a test harness — not worth the refactor this sprint.

**Orchestrator synthesis (applied):** Quick assign now dismisses the loupe **before** `LinkAssetsToCollection` (same as bulk assign), not after success; story + sprint comment updated. Status stays **review** until interactive DoD.

### Party mode — dev session 1/2 (automated headless)

**Roster:** `_bmad/_config/agent-manifest.csv` was missing; this round used simulated BMAD-style roles.

- **Winston (Architect):** Keep Story 2.10 scope (no loupe create / multi-assign creep). The subtle regression is browse vs assign targets drifting together; prove separation in CI, not only in code review.
- **Sally (UX):** Strip plus right-click hint remains the non-hover MVP. Quick-assign list failures should use the same "album list unavailable" tone as the bulk strip so Review reads as one surface.
- **Amelia (Dev):** Low-cost test: seed `CreateCollection` plus `InsertAsset`, open `NewReviewView`, read the fourth `Select`, assert options exclude the browse sentinel and include the seeded album.
- **Murat (Test):** Layout-only tests are not enough; we had not asserted AC3 after a real refresh with data. List failures are not link failures, but they should still avoid one-off copy.

**Orchestrator synthesis (applied):** Add `TestStory210_AssignTargetOptionsExcludeBrowsePredicate`; align quick-assign `ListCollections` error copy with the bulk strip; update story + sprint comment; keep status **review** until interactive DoD.

## References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Story 2.10, Epic 2, FR coverage map]  
- [Source: `_bmad-output/planning-artifacts/PRD.md` — FR-08, FR-17]  
- [Source: `_bmad-output/planning-artifacts/ux-design-specification.md` — §Component Strategy (Filter strip: assign from filter context; Thumbnail grid cell: hover + non-hover duplicates)]  
- [Source: `_bmad-output/planning-artifacts/epics.md` — UX-DR2, UX-DR3, UX-DR15]  
- [Source: `_bmad-output/planning-artifacts/architecture.md` — §3.2 data model collections / junction; §5.2 layering; stack Fyne v2.7.3, Go 1.25.4]  
- [Source: `_bmad-output/implementation-artifacts/2-9-collection-crud-multi-assign.md` — loupe scope boundary, `collectionStoreErrText`, `CreateCollectionAndLinkAssets`, refresh patterns]  
- [Source: `internal/app/review.go` — bulk tag / reject / delete selection patterns]  
- [Source: `internal/app/review_grid.go` — grid selection, `bindRow`, tap handler]  
- [Source: `internal/store/collections.go` — `LinkAssetsToCollection`]