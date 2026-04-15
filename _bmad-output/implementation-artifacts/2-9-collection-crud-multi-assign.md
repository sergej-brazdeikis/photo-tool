# Story 2.9: Collection CRUD, multi-assign, and safe collection delete

Status: in-progress

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- create-story workflow (2026-04-13): spec from epics §Story 2.9, PRD FR-18–FR-20, UX form patterns, brownfield store (CreateCollection, LinkAssetsToCollection, DeleteCollection), Story 2.8 CollectionsView + loupe shell. -->
<!-- Party mode create 1/2 (2026-04-13): AC2 ambiguity resolved; display-date validation + risks/DoD; DeleteCollection missing-id → ErrCollectionNotFound in store + test. -->
<!-- Party mode create 2/2 (2026-04-13): AC3 atomic create+link; Unlink idempotency contract; focus-order / stale-dialog risks; store `CreateCollectionAndLinkAssets` + rollback tests. -->
<!-- Party mode dev 1/2 (2026-04-13): Loupe FK/user copy via `albumStoreErrText`; single-flight “New album” save; `CreateCollection` godoc matches validation; `TestAlbumStoreErrText`. -->
<!-- Party mode dev 2/2 (2026-04-13): Shared `collectionStoreErrText` (+ `ErrCollectionNotFound`); Collections create/edit single-flight + same copy; `TestCollectionStoreErrText`. -->

## Story

As a **photographer**,  
I want **to create, rename, and delete collections without orphaning rules wrong**,  
So that **metadata stays trustworthy**.

**Implements:** FR-18–FR-20.

## Acceptance Criteria

1. **Create collection (FR-18):** **Given** a create-collection flow (from the **Collections** list and/or from **single-photo / loupe** per AC3), **when** the user saves, **then** **name** is **required** after trim; empty/whitespace-only name is rejected with a clear validation message (UX spec: errors next to field or summary; preserve input). **Display date** is **optional**; if omitted (after trim), the stored value follows FR-18 / PRD: use **collection creation calendar date** as **`YYYY-MM-DD`** in the operator’s **local** timezone (same rule as `store.CreateCollection` today — not photo capture time). **If** the user enters a **non-empty** display date, **then** it must be a **valid calendar** ISO date `YYYY-MM-DD` (parse/validate in `internal/store` or a shared helper); **invalid** values are rejected with a clear message (preserve **name** input). **Display date** does not change member assets’ capture times or paths (PRD).
2. **Rename / edit collection (FR-18):** **Given** an existing collection, **when** the user saves edits, **then** **name** remains required (same trim/validation as AC1). **Display date** validation matches AC1 for non-empty input. **Cleared / omitted display date on edit (locked rule):** when the user clears the optional field (empty after trim), **recompute** and persist **`YYYY-MM-DD`** from the row’s **`created_at_unix`** using **`time.Unix(created_at_unix, 0).In(time.Local)`** — i.e. the same *semantic* default as FR-18 “creation calendar date,” not “keep whatever was there before.” Encode in **`UpdateCollection`** (or equivalent) and **unit test** the clear path (schema **`display_date` stays NOT NULL**).
3. **Single-photo multi-assign + inline create (FR-19):** **Given** **Review loupe** (`openReviewLoupe` / `review_loupe.go`) for an asset, **when** the user manages collection membership, **then** they can **assign the current photo to more than one collection** (many-to-many persists in `asset_collections`). The UI must make current memberships visible and allow **adding** and **removing** membership without leaving loupe (exact control pattern: checklist, multi-select list, or equivalent — match existing Fyne density). **And** the user can **create a new collection from this surface** and have the **current asset linked** to it in the same flow. **Atomicity (locked):** the **inline create-from-loupe** path must **not** persist a new `collections` row unless the **current asset** link succeeds — implement via **`store.CreateCollectionAndLinkAssets`** (single transaction) or equivalent; **no** orphan empty albums after a failed FK/link. **Out of scope:** thumbnail **hover** assign (FR-08) and **filter-workflow bulk** assign (FR-17) — those are **Story 2.10**. *(Upload batch still uses separate create/link in `upload.go`; orphan risk there is tracked under Story 1.5 review — not required in 2.9.)*
4. **Safe collection delete (FR-20):** **Given** a delete action on a collection, **when** the user has **not** confirmed, **then** no delete runs. **When** confirmed, **then** all **`asset_collections`** rows for that collection are removed **before** / as part of removing the **`collections`** row (DB already uses **`ON DELETE CASCADE`** on `asset_collections.collection_id` — implementation must call `store.DeleteCollection` or equivalent and remain correct). **Store contract:** if the collection id **no longer exists** (stale UI, race), **`DeleteCollection`** returns an error satisfying **`errors.Is(err, store.ErrCollectionNotFound)`** so the UI can match Story 2.8 “missing album” handling. **UI:** use **destructive** styling for the committing control and a **confirm** dialog with explicit consequence copy (UX-DR5 pattern; align with Story 2.7 delete confirm tone). After delete, **Collections** list/detail refresh; if the deleted id was open in detail, behavior matches Story 2.8 AC9 (no panic; return to safe surface + message).
5. **No orphan semantics:** **Given** any successful membership or delete operation, **when** inspected in SQLite, **then** there are **no** `asset_collections` rows pointing at a missing `collections.id` (FK integrity maintained).
6. **Regression — browse predicates:** **Given** collection detail and Review filters, **when** CRUD or loupe assign runs, **then** default asset visibility still respects **`ReviewBrowseBaseWhere`** where applicable (Story 2.8 / 2.3 precedent); collection membership queries remain parameterized in **`internal/store`** (architecture §5.2 — no SQL in widgets).

## Tasks / Subtasks

- [x] **Store: read + update APIs** (AC: 1, 2, 5)
  - [x] **`CreateCollection`**: non-empty **`display_date`** must parse as **`YYYY-MM-DD`** in local TZ; **`TestCreateCollection_invalidDisplayDate`**. *(Party create 1/2 — 2026-04-13.)*
  - [x] **`CreateCollectionAndLinkAssets`**: insert + junction writes in **one transaction**; failed link rolls back collection insert; **`TestCreateCollectionAndLinkAssets_success`**, **`TestCreateCollectionAndLinkAssets_rollsBackOnBadAsset`**. *(Party create 2/2 — 2026-04-13.)*
  - [x] **`DeleteCollection`**: missing id returns **`ErrCollectionNotFound`** (wrapped); **`TestDeleteCollection_notFound`** (`internal/store/store_test.go`). *(Party create 1/2 — 2026-04-13.)*
  - [x] Add **`GetCollection`** (or extend list row type) to load **`name`**, **`display_date`**, **`created_at_unix`** for edit dialogs.
  - [x] Add **`UpdateCollection(db, id, name, displayDateISO)`** (or split rename vs date) with **trim**, **non-empty name**, **`display_date` ISO validation** when non-empty, **`ErrCollectionNotFound`** when id missing — for “cleared” optional date on edit, **recompute** from **`created_at_unix`** per AC2 (local calendar `YYYY-MM-DD`).
  - [x] Add **`ListCollectionIDsForAsset(db, assetID)`** (ordered, stable) and **`UnlinkAssetFromCollection(db, assetID, collectionID)`** (or batch replace helper) for loupe membership edits; preserve **`LinkAssetsToCollection`** idempotency for adds. **`UnlinkAssetFromCollection`** must be **idempotent**: deleting a non-existent link is a **no-op** (no error); only **real** DB failures surface — document in godoc and test the no-op path.
  - [x] Unit tests: update validation, **not-found** id, **multi-assign** (asset in 2+ collections), **unlink** leaves other links, **delete** clears junction rows. *(Junction cleanup covered by existing `TestCollections_createLinkIdempotentDelete`.)*
- [x] **Collections list / detail UI: CRUD chrome** (AC: 1, 2, 4, 6)
  - [x] **`internal/app/collections.go`:** expose **New album**, **Rename/Edit**, **Delete** (or context/long-press pattern) without violating **full-page** browsing (UX-DR10 — no modal **stack** for **browsing** the grid; **dialogs** for forms/confirm are OK per UX “modals for confirm delete”).
  - [x] Wire create/edit **form**: name + optional display date (`widget.Entry` / `DateEntry` if available — otherwise ISO text with validation message); on success, **`reloadCollectionRows`** and refresh any dependent UI.
  - [x] Delete: **confirm** → **`store.DeleteCollection`** → refresh list; if deleted id == **`detailCollectionID`**, **`ResetToList`** + short message (AC4).
  - [x] Replace placeholder copy in empty list state that still says albums appear “in a later update” once CRUD ships (Story 2.8 list empty state).
- [x] **Review loupe: collection membership strip** (AC: 3, 5, 6)
  - [x] **`internal/app/review_loupe.go`:** add a **scroll-safe** subsection (below chrome / side column — follow NFR-01 / UX-DR4 spacing) showing **all collections** with **toggles** for current asset; persist adds via **`LinkAssetsToCollection`**, removes via **`UnlinkAssetFromCollection`**; debounce or single-flight DB writes if needed.
  - [x] **Create new collection** from loupe: small dialog → **`CreateCollectionAndLinkAssets`** with **`[]int64{currentAssetID}`** (not separate create+link); refresh loupe membership controls and **filter strip** collection source if cached in Review.
  - [x] Ensure **`onReviewDataChanged`** (or new callback) refreshes grid/filter strip when membership changes so filters don’t show stale “in collection” state.
- [x] **Cross-surface refresh** (AC: 3, 4, 6)
  - [x] After collection **rename/delete/create**, any UI that caches **`ListCollections`** (Review filter strip, loupe) must **reload** — grep for **`ListCollections`** / collection picker state.
- [x] **Tests** (AC: 1–6)
  - [x] Prefer **`internal/store`** tests for all new SQL; **`internal/app`** tests only where headless-safe (pure helpers / state predicates).
  - [x] **`CreateCollectionAndLinkAssets`** happy path + **rollback** when link targets a missing asset (session 2/2).

## Dev Notes

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **FR-18 semantics** | Name required; display date optional with **local calendar** default on create; display date is **metadata only** (PRD FR-18). |
| **FR-19 scope** | **Loupe only** for multi-assign + inline create in this story; **not** FR-08 / FR-17 (2.10). |
| **FR-20** | **`DeleteCollection`** in `collections.go` documents CASCADE; UI must still **confirm** before call. |
| **Data model** | `002_collections.sql`: `collections(name, display_date NOT NULL, created_at_unix)`; `asset_collections` composite PK + FK CASCADE. |
| **Errors** | Use **`errors.Is(..., store.ErrCollectionNotFound)`** for missing collection id — no driver string matching (Story 2.8 precedent). |

### Existing implementation to extend (do not reinvent)

- **`store.CreateCollection`**, **`CreateCollectionAndLinkAssets`** (loupe inline create), **`LinkAssetsToCollection`**, **`DeleteCollection`**, **`ListCollections`**, **`CollectionExists`** — `internal/store/collections.go`.
- **`CollectionsView`** list/detail state machine — `internal/app/collections.go` (Story 2.8).
- **Loupe** entrypoint **`openReviewLoupe`** — `internal/app/review_loupe.go`; grid **`reviewAssetGrid`** for asset id + refresh callbacks.
- **Upload post-import** already creates collections — `internal/app/upload.go` (reuse validation **patterns**, not duplicate business rules inconsistently).

### Architecture compliance

- **SQLite + migrations:** No new migration unless a column/index is required; current schema supports FR-18–FR-20.
- **`app → domain, store`:** SQL and transactions in **`internal/store`**; Fyne in **`internal/app`** (architecture §5.2).

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3** (`go.mod`): `dialog`, `widget`, `container` for forms and confirm; destructive button styling via theme (**`theme.ErrorColor()`** or project theme roles from Story 2.1).
- **SQLite:** `modernc.org/sqlite` — use parameterized queries; transactions for multi-step membership replaces if implemented.

### File structure requirements

| Area | Files (expected touch) |
|------|-------------------------|
| Store | `internal/store/collections.go`, new tests in `internal/store/*_test.go` |
| Collections UI | `internal/app/collections.go`, `internal/app/collections_test.go` |
| Loupe | `internal/app/review_loupe.go`, possibly `internal/app/review.go` for refresh wiring; inline create uses **`CreateCollectionAndLinkAssets`** |
| Review filters | Files that populate collection dropdown / list from DB (grep **`ListCollections`**) |

### Testing requirements

- **Store-first:** CRUD edge cases (empty name, bad id, multi-link, unlink one of many, delete collection with many members).
- **Concurrency:** Single-user MVP; still avoid nested **`Begin`** without need; document if loupe fires overlapping toggles.

### Previous story intelligence (2.8)

- **Full-page detail** stays **non-modal**; dialogs only for **forms** and **confirm** (2.8 AC1, UX-DR10).
- **`ResetToList` / AC9:** Reuse **`CollectionExists`** and transient list messages when ids go stale after delete.
- **Shell:** **`collectionsNavShouldResetToList`** (2.8 AC12) may interact with new toolbar — retest nav re-select after CRUD.

### Project context reference

- No `project-context.md` in repo root; rely on PRD, architecture, and this file.

### Risks & mitigations (create sessions 1–2)

| Risk | Mitigation |
|------|------------|
| **Loupe membership toggles** fired faster than SQLite commits → UI/DB drift | **Single-flight** or disable the toggling control while a write is in flight; refresh list from DB after success. |
| **Stale collection id** after another surface deletes or renames | Use **`ErrCollectionNotFound`** / **`CollectionExists`** patterns from Story 2.8; refresh **`ListCollections`** caches after CRUD. |
| **Duplicate collection names** (schema allows) | **In scope:** no uniqueness constraint unless PRD changes; **UX:** consider showing **display_date** in loupe list rows if names collide. |
| **Timezone / “creation day”** for default from **`created_at_unix`** | Always use **`time.Local`** when formatting defaults (document in `UpdateCollection` / `CreateCollection` comments). |
| **Create dialog open while collection deleted elsewhere** (session 2/2) | On save, surface **not-found** like Story 2.8; **do not** assume id still valid after long idle. |
| **Loupe membership strip keyboard / focus** (session 2/2) | Align with **UX-DR15** where practical: toggles reachable after loupe chrome; no trap focus in scroll sub-region. |
| **Unlink called twice or on stale toggle state** (session 2/2) | **`UnlinkAssetFromCollection`** idempotent no-op prevents spurious errors when UI races. |

### Definition of Done

- All AC1–6 pass in manual QA plus **store unit tests** for **`GetCollection`**, **`UpdateCollection`**, **`ListCollectionIDsForAsset`**, **`UnlinkAssetFromCollection`** (including **idempotent unlink**), **`CreateCollectionAndLinkAssets`**, display-date validation, and AC2 “clear date” path.
- **No raw SQL** in Fyne widgets; membership and CRUD go through **`internal/store`**.
- Collections list empty-state copy no longer promises albums “in a later update” once create ships.
- **Loupe** membership strip and **filter strip** stay in sync after membership changes (**grep** `ListCollections` / collection caches).

### Party mode — implementation pre-work (session 1/2)

- **`internal/store/collections.go`:** `DeleteCollection` returns **`fmt.Errorf(..., ErrCollectionNotFound)`** when **`RowsAffected == 0`**.
- **`internal/store/collections.go`:** `CreateCollection` rejects **non-empty** display dates that do not parse as **`YYYY-MM-DD`** in **`time.Local`** (AC1).
- **`internal/store/store_test.go`:** **`TestDeleteCollection_notFound`**, **`TestCreateCollection_invalidDisplayDate`**.

### Party mode — simulated round (session 2/2, create hook)

_Roster: manifest path missing — personas per BMAD defaults._

- **John (PM):** Session 1 nailed display-date semantics, but AC3 still allowed a **two-step** create+link. That repeats the orphan-collection failure mode called out in **1-5** upload review. Lock **atomicity** for loupe: one transaction or explicit rollback — don’t ship “create then link” again.
- **Sally (UX):** I’m fine with atomicity. Where I push back is **error recovery**: if the transaction fails, the dialog should keep **name + date** filled and show a **single** clear message (FK vs validation), not two stacked toasts. For membership toggles, plan **focus order** into the strip (UX-DR15); dense checklists are useless if keyboard users can’t reach them.
- **Winston (Architect):** Agree on **`CreateCollectionAndLinkAssets`**. Refactor **`LinkAssetsToCollection`** to share a **`tx`** helper so we don’t fork insert logic. For **`UnlinkAssetFromCollection`**, decide now: **idempotent delete** (0 rows OK) avoids toggle races; reserve **`ErrCollectionNotFound`** for **collection** missing, not for “link already gone.”
- **Murat (Test Architect):** Add **`TestCreateCollectionAndLinkAssets_rollsBackOnBadAsset`** — proves no row in **`collections`** when FK blows. For **`Unlink`**, test double-unlink and “never linked.” Session 1 already covered invalid display dates; add **leap-year** sanity only if we see parser drift — Go’s **`ParseInLocation`** is strict enough.

**Orchestrator synthesis:** Encode **AC3 atomicity** + **`CreateCollectionAndLinkAssets`** in the story; implement store API + rollback test; require loupe UI task to call it; specify **`Unlink`** idempotency + UX risks (stale dialog, focus). Upload stays out of scope for 2.9.

### Party mode — dev session 1/2 (automated headless)

_Roster: `_bmad/_config/agent-manifest.csv` missing — personas per BMAD defaults._

- **Murat (Test Architect):** Store coverage already hits rollback, unlink idempotency, and AC2 clear-date. What remains for **dev** is **UX truthfulness**: when a link fails with SQLite `FOREIGN KEY`, the loupe still showed the raw driver string, which conflicts with the story expectation of a clear message for non-validation failures. Add a small mapper plus a test; do not re-audit AC paragraphs.

- **Sally (UX):** Apply the same treatment in the **toggle** path — same stale-ID scenario as inline create. One sentence, no stacked dialogs. Double-tap **Save** on “New album” can still race two creates; **single-flight** the save handler.

- **Amelia (Dev):** `CreateCollection` godoc still said non-empty dates are “stored as-is”; code validates `YYYY-MM-DD`. Fix the comment or the next dev might “optimize away” validation. Skip new migrations and avoid scope creep into upload (1.5).

- **Winston (Architect):** Keep hints in **`internal/app`**, not store — preserves layering. Detect **`FOREIGN KEY`** substring rather than importing `sqlite` error types; stable across driver builds.

**Orchestrator synthesis (applied):** Implement `albumStoreErrText` + `TestAlbumStoreErrText`; use for loupe New album inline errors and membership-toggle `ShowError`; guard New album Save with `atomic.Bool`; correct `CreateCollection` documentation in `collections.go`; record in sprint log.

### Party mode — dev session 2/2 (automated headless)

_Roster: `_bmad/_config/agent-manifest.csv` missing — personas per BMAD defaults. Session 2 challenges session 1: string-matching FK is not enough for stale-id semantics._

- **Winston (Architect):** Session 1 only rewrote errors when `FOREIGN KEY` appeared in the text. **`ErrCollectionNotFound`** is the contract from the store; the UI should map **`errors.Is`** first and only then fall back to driver substrings. Otherwise a wrapped not-found still surfaces as `update collection 99: collection not found`, which violates the “clear message” bar next to the loupe’s friendlier copy.

- **Murat (Test Architect):** Agree on **`errors.Is`**. Add a unit test that wraps **`ErrCollectionNotFound`** — prove we don’t regress to raw sentinel text. Also **`CollectionsView.showAlbumForm`** had no single-flight guard: double **Save** on **New album** from the list can still race two inserts; loupe was fixed in 1/2 but parity matters.

- **Sally (UX):** One vocabulary: **Collections** inline form and **loupe** should show the same **stale album** sentence where we don’t already open the information dialog. Users shouldn’t learn two failure dialects on the same feature.

- **Amelia (Dev):** Centralize the mapper in **`internal/app`** (new small file), call it from **`review_loupe`** and **`collections`**. Keep store free of Fyne; keep substring fallback for FK only in the app layer.

**Orchestrator synthesis (applied):** Add **`collectionStoreErrText`** with **`errors.Is(…, store.ErrCollectionNotFound)`** then FK substring; use in **loupe** (replaces inline helper); use for **Collections** create/edit field errors; **`atomic.Bool` single-flight** on **list/detail album form Save**; extend tests → **`TestCollectionStoreErrText`**.

## Dev Agent Record

### Agent Model Used

Cursor agent (Claude) — dev-story workflow 2026-04-13.

### Debug Log References

### Completion Notes List

- Implemented **`GetCollection`**, **`UpdateCollection`** (AC2 clear-date → `created_at_unix` local day), **`ListCollectionIDsForAsset`**, **`UnlinkAssetFromCollection`** (idempotent) in `internal/store/collections.go` with **`internal/store/store_test.go`** coverage.
- **CollectionsView:** New album, picker-based Rename/Delete on list, Edit/Delete on album detail, form validation via store errors, destructive delete confirm, updated empty-state copy.
- **Review loupe:** scrollable album checklists + **`CreateCollectionAndLinkAssets`** dialog; **`refreshReviewData`** reloads **`ListCollections`** into the filter strip; **`suspendColSelectRefresh`** prevents **`SetSelected`** recursion/stack overflow.
- **Party dev 1/2:** Loupe surfaces user-facing text for FK failures (`albumStoreErrText`); New album Save single-flight; `CreateCollection` godoc aligned with date validation; **`TestAlbumStoreErrText`**.
- **Party dev 2/2:** Shared **`collectionStoreErrText`** (`ErrCollectionNotFound` + FK); Collections album form single-flight + same copy; **`collection_store_err_text.go`**, **`TestCollectionStoreErrText`**.

### File List

- `internal/store/collections.go`
- `internal/store/store_test.go`
- `internal/app/collection_store_err_text.go`
- `internal/app/collections.go`
- `internal/app/review.go`
- `internal/app/review_loupe.go`
- `internal/app/review_loupe_test.go`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/2-9-collection-crud-multi-assign.md`

### Change Log

- 2026-04-13: Story 2.9 implementation — store CRUD helpers, Collections + loupe UI, Review collection strip refresh; sprint status → review.
- 2026-04-13: Party mode dev 1/2 — loupe FK-friendly errors, single-flight new album save, `CreateCollection` doc fix, `review_loupe_test.go`.
- 2026-04-13: Party mode dev 2/2 — centralized collection err copy + not-found via `errors.Is`, Collections form single-flight, renamed/extended app tests.

## References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Story 2.9, Epic 2]
- [Source: `_bmad-output/planning-artifacts/PRD.md` — FR-18, FR-19, FR-20]
- [Source: `_bmad-output/planning-artifacts/ux-design-specification.md` — Form patterns (Collection create/edit), §Modals vs collection browsing]
- [Source: `_bmad-output/planning-artifacts/architecture.md` — §Schema `collections` / `asset_collections`, §5.2 layering]
- [Source: `internal/store/migrations/002_collections.sql` — schema + CASCADE]
- [Source: `internal/store/collections.go` — `CreateCollection`, `CreateCollectionAndLinkAssets`, `DeleteCollection`, `LinkAssetsToCollection`]
- [Source: `_bmad-output/implementation-artifacts/2-8-collections-list-detail.md` — prior CollectionsView + loupe constraints]

### Review Findings

_Code review (BMAD adversarial layers), 2026-04-14. Headless run: patches left as action items._

- [ ] [Review][Patch] Review filter strip stays stale after album CRUD on Collections — `NewReviewView` only calls `refreshAll` at construction and when filters change; `NewMainShell` does not refresh Review when switching panels. After creating/renaming/deleting an album on the Collections tab, the Review “Collection” and bulk-assign dropdowns can list outdated albums until some other refresh runs (e.g. loupe membership change). Violates story AC6 / cross-surface refresh task. [`internal/app/shell.go` nav + `internal/app/review.go` `refreshReviewData`]

- [x] [Review][Defer] Duplicate album names make Fyne `Select` / checklist labels ambiguous (first matching label wins) — called out in story risks and `review.go` comment; no uniqueness constraint in schema. [`internal/app/review.go`, `internal/app/collections.go`, `internal/app/review_loupe.go`] — deferred, pre-existing product/data-quality choice

- [x] [Review][Defer] `LinkAssetsToCollection` with empty `assetIDs` returns success without verifying `collectionID` exists — documented API footgun for callers, not an end-user path. [`internal/store/collections.go`]
