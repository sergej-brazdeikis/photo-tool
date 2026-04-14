# Story 2.8: Collections list and full-page collection detail

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- create-story workflow (2026-04-13): spec from epics + architecture + UX + brownfield (shell placeholder, review_query, collections store). -->
<!-- Party mode create 1/2 (2026-04-13): AC for empty library, membership-only-visible, invalid id, paging vs grouping; camera_label helper contract; tasks split store/UI/tests. -->
<!-- Party mode create 2/2 (2026-04-13): Paging×grouping boundary contract; nav reset; NULL star ordering; camera trim edge cases; ErrCollectionNotFound; list scope. -->
<!-- Party mode dev1/2 (2026-04-13): AC7 list chrome + AC12 pure predicate + day paging regression test; init order fix for list visibility. -->
<!-- Party mode dev2/2 (2026-04-13): camera-bucket paging regression; per-section scroll/cap doc; ErrCollectionNotFound on lazy grid fetch → list (AC9 mid-scroll). -->

## Story

As a **photographer**,  
I want **to browse collections on their own page**,  
So that **albums feel like first-class places**.

**Implements:** FR-21–FR-24; UX spec — Collections list/detail shell (full-page navigation).

## Acceptance Criteria

1. **Full-page collection detail (FR-21):** **Given** the **Collections** primary area (replacing the Story 2.1 placeholder), **when** the user selects a collection from the list, **then** the main content switches to a **dedicated collection detail surface** that occupies the **same content region** as the list (shell nav remains visible). **Do not** use a modal or blocking popup as the primary container for browsing that collection’s photos (UX spec: modals not for main collection browsing). A **Back** (or equivalent) control returns to the list without leaving the Collections nav item.
2. **Default sort by capture time (FR-22):** **Given** collection detail for a non-empty collection, **when** no grouping mode overrides ordering, **then** assets are ordered by **`capture_time_unix`** descending with **`id`** tie-break (consistent with `ListAssetsForReview`), using the same **default visibility** as Review: **`ReviewBrowseBaseWhere`** (`rejected = 0 AND deleted_at_unix IS NULL`).
3. **Default grouping: star rating (FR-23):** **Given** collection detail loads, **when** grouping is at its default, **then** the UI presents **sections** for **star ratings 5 → 1** only (descending). **Omit** any rating bucket that would be empty. **Within** each non-empty bucket, sort by **`capture_time_unix` DESC, `id` DESC**. **Unrated** assets (`rating IS NULL`): include a final **“Unrated”** section **only when** non-empty; place it **after** the 1-star section (lowest priority). Omit the Unrated section when empty. **SQL/Go ordering contract:** any store query that orders by rating for this mode **must** place **`NULL` ratings after all numeric 1–5** (never interleaved between star buckets); match the section order **5…1 then Unrated**.
4. **Grouping: by calendar day (FR-24):** **Given** the user switches grouping to **by day**, **when** detail refreshes, **then** assets are partitioned by **local calendar day** derived from `capture_time_unix` (document the timezone rule in code: **match** the rule used elsewhere for operator-visible dates, e.g. `time.Local` wall date from Unix seconds). Sections sort **newest day first**; within a day, **`capture_time_unix` DESC, `id` DESC**. Omit empty days.
5. **Grouping: by camera name (FR-24 + FR-26):** **Given** the user switches grouping to **by camera name**, **when** detail refreshes, **then** assets are partitioned by a **deterministic camera label** persisted on the asset row (PRD FR-26 — camera make/model when present). **If** metadata is missing, use a single stable label such as **“Unknown camera”** and **omit** that bucket when empty (same “omit empty groups” rule as FR-23). Within a bucket, **`capture_time_unix` DESC, `id` DESC**. **Label rule (dev must implement in one place — `store` or `domain`):** from nullable `camera_make` / `camera_model` (or a single stored `camera_label`), build display key as **trimmed** strings, **single-space** join when both present, **one field only** when the other is empty, **`NULL`/empty both →** treat as unknown for grouping (unknown bucket still subject to “omit when empty”). **Trim edge case:** a field that is **only whitespace** after trim counts as **empty** (do not emit a stray leading/trailing space segment in the join). **Collapsing duplicate spaces**; **no** locale-sensitive title case or Unicode normalization required for MVP (store/compare UTF-8 as read from EXIF/SQLite). **Prerequisite:** nullable camera field(s) or a single derived `camera_label` column on `assets`, populated on **ingest/import paths** for new files and **readable** by collection queries — extend `internal/exifmeta` (or equivalent) minimally; do **not** claim FR-26 completeness beyond what this story needs for grouping. **Backfill:** existing rows may keep NULL until scan/import touches them; unknown bucket is acceptable for those assets.
6. **Empty collection:** **Given** a collection with **no** linked visible assets, **when** detail opens, **then** show a calm empty state (copy + optional primary hint); **no** crash or bare grid.
7. **Empty collection list (library has zero collections):** **Given** `ListCollections` returns no rows, **when** the user opens **Collections**, **then** show a list empty state (copy distinct from AC6 — e.g. hint that albums appear after upload/assign or future CRUD per Story 2.9); **no** bare screen.
8. **Membership vs visibility:** **Given** a collection whose linked assets are **all** rejected and/or soft-deleted, **when** detail opens, **then** behavior matches AC6 (no visible rows under `ReviewBrowseBaseWhere`) — treat as **empty detail**, not an error.
9. **Invalid or missing collection id:** **Given** detail is requested for an id with **no** `collections` row (deleted elsewhere, stale UI, or bug), **when** the view loads or refreshes, **then** show a short non-panicking message and return focus to the **list** (or equivalent safe fallback); **no** uncaught error dialog loop.
10. **Paging / memory (NFR alignment):** **Given** a collection with many visible assets, **when** the user scrolls detail, **then** thumbnail loading stays **bounded** like Review (Story 2.3): **windowed or paged** fetch; **do not** load full-collection pixmaps eagerly. **Paging × grouping (hard constraint):** naive `LIMIT/OFFSET` on a **flat** ordered stream **without** a stable cursor can **split a group across pages** (bad UX: duplicate section headers or half-empty buckets). The implementation **must** choose and **document** one of: **(A)** a single **keyset/cursor** over the flat `(capture_time_unix DESC, id DESC)` stream with **group labels computed in UI** from the streamed rows (sections grow until the next key change), **(B)** **per-section** queries each with their own cap/window (document ordering + how “load more” works per section), or **(C)** another approach **proved in tests** to never render misleading partial groups. If **(B)**, define how empty omitted buckets stay omitted when paginating. **Document the chosen letter/strategy** beside collection detail code and in a store comment; Epic 2.10 must not require redesign.
11. **Regression / boundaries:** Collection queries and UI **must not** bypass `ReviewBrowseBaseWhere` for default browsing; **no SQL** in Fyne widgets — keep predicates and queries in **`internal/store`** (architecture §5.2).
12. **Primary nav resets drill-in:** **Given** the user opened **collection detail** from the list, **when** they activate the **Collections** primary nav item again (same section, already selected), **then** the UI returns to the **collection list** (clears detail state). **Rationale:** avoids feeling “stuck” inside an album when re-tapping the nav; matches common desktop shell expectations. **Note:** Story 2.9 may refine this if CRUD adds modal flows; for 2.8, list is the stable home.

## Risks & mitigations

| Risk | Mitigation |
|------|------------|
| **FR-26 not yet in schema** blocks “by camera” | Add a **minimal** migration + extraction hook; nullable columns; deterministic label helper in `store` or `domain`. |
| **Grouping × paging** interaction | Prefer **flat ordered query** + **group in Go** for first implementation if SQL `GROUP BY` complicates offsets; or **section-scoped** `LIMIT/OFFSET` with tests proving order — pick one and document (AC10). |
| **Duplicate grid logic** vs Review | Prefer **shared** row type (`ReviewGridRow` or thin alias) and **shared** thumbnail loader patterns (`thumbnail_disk.go`) — extract only if duplication is material. |
| **Timezone ambiguity** for “by day” | Document one rule; use same as operator-facing date elsewhere or `time.Local` from Unix. |
| **Performance** on large collections | Reuse **paged / windowed** loading from Review grid; avoid loading all pixmaps at once (architecture §3.1 item 8, Story 2.3 precedent). |
| **Collection deleted while detail open** | On load/refresh, treat as AC9; list refresh after returning from detail should match store. |
| **Flat `LIMIT/OFFSET` + grouped UI** | Splits groups across pages; AC10 forces cursor, per-section windows, or tested alternative. |
| **`NULL` star order differs between SQL and Go** | Buckets disagree with section headers; AC3 pins `NULL` after 1–5. |
| **Nav re-select leaves user “stuck” in detail** | AC12 resets to list when Collections nav activated again. |

## Definition of Done

- [x] Collections list + detail ship behind **Collections** nav; placeholder text from `shell.go` removed for this section.
- [x] **AC1–12** satisfied with **automated** tests where practical: **store-level** tests for list-in-collection + grouping ordering + **invalid collection id** + **all-linked-assets-invisible** case; **AC7** covered by `internal/app/collections_test.go` (empty library hides list + copy). **AC10:** star-bucket guard in `collection_detail_test.go` plus **day-bucket** paging guard (`TestCollectionDetail_daySectionPagingDoesNotMixDays`) and **camera-bucket** guard (`TestCollectionDetail_cameraSectionPagingDoesNotMixLabels`). **AC12:** unit test for `collectionsNavShouldResetToList` in `shell_test.go` (nav re-tap predicate); manual smoke still recommended for full shell.
- [x] `go test ./...` green; **no** regressions to Review/Rejected predicates.
- [ ] Manual: open collection → full-page detail → switch grouping modes → back to list; verify non-modal flow; **optional:** `Esc` or platform back **if** easy in Fyne without scope creep (not blocking).

## Tasks / Subtasks

- [x] **Schema + metadata minimal FR-26 hook** (AC: 5)
  - [x] Add migration for camera fields needed for grouping (e.g. nullable `camera_make`, `camera_model` TEXT **or** single `camera_label` TEXT).
  - [x] Extend ingest/import registration to **populate** from EXIF when available; document fallback to NULL → **“Unknown camera”** in grouping layer.
  - [x] SQLite tests: row with make/model → expected label; NULL → unknown bucket behavior.
- [x] **Store: collection asset queries** (AC: 2–5, 8–11)
  - [x] New APIs in `internal/store` (e.g. `ListCollectionAssetsForDetail(collectionID, …)` + `CollectionExists`) composing `ReviewBrowseBaseWhere` + membership (`EXISTS` or `JOIN asset_collections` with `collection_id = ?`).
  - [x] Implement grouping **either** in SQL (`GROUP BY` / window functions) **or** fetch ordered rows and group in Go — **justify in code comment** for testability and paging (AC10).
  - [x] Reuse ordering: `capture_time_unix DESC, id DESC` aligned with `review_query.go`; star sections **5 → 1** then **Unrated** last when non-empty.
  - [x] Unit/integration tests: **unknown collection id** returns **`store.ErrCollectionNotFound`** (exported `errors.Is` sentinel) or a typed result **documented** beside the API — **no** UI string-matching of driver errors.
- [x] **Domain: grouping enum** (AC: 3–5)
  - [x] Small type in `internal/domain` for `CollectionGrouping` / modes: `Stars`, `Day`, `Camera` with string persistence for UI if needed.
- [x] **Fyne: Collections view** (AC: 1, 6–7, 9)
  - [x] Replace `NewSectionPlaceholder` for `"collections"` in `internal/app/shell.go` with a real constructor (e.g. `NewCollectionsView`).
  - [x] **List** screen: `ListCollections` + tap/click → navigate to detail state; **AC7** empty state when len=0.
  - [x] **Detail** screen: grouping **selector** (segmented / radio / select — match existing app controls); **section headers** + grid of thumbnails; **Back** to list (toolbar or top row — **not** a modal).
  - [x] Empty states: distinguish **no collections in library** (AC7) vs **collection has no visible assets** (AC6, AC8).
  - [x] **AC9:** handle missing collection row on open/refresh — return to list + user-visible short message.
  - [x] **AC12:** when **Collections** nav is chosen while detail is showing, **pop** to list (RadioGroup may not fire if already selected — use explicit nav callback or duplicate-tap handling as appropriate for Fyne).
- [x] **Thumbnail / grid reuse** (AC: 2–4)
  - [x] Reuse `ReviewGridRow` fields and thumbnail loading from `review_grid.go` / `thumbnail_disk.go` patterns; do not fork decode paths unnecessarily.
- [x] **Tests** (AC: 2–9, 10–12)
  - [x] `internal/store` tests: membership + grouping ordering + empty buckets omitted + **all members rejected/deleted** → zero visible rows + **invalid collection id** behavior.
  - [x] Optional `internal/app` tests: grouping mode string / navigation state if testable without headless Fyne pain.

## Dev Notes

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **Visibility** | Use **`ReviewBrowseBaseWhere`** for assets shown in collection detail (same as default Review — Story 2.3/2.6 precedent). |
| **Capture time** | Sort key is `capture_time_unix` already on `assets` (ingest/backfill from Story 1.x / 1.7). |
| **Collections data** | `collections` + `asset_collections` exist (`002_collections.sql`); `ListCollections`, `LinkAssetsToCollection` in `internal/store/collections.go`. |
| **Navigation** | **Full page** = content swap inside shell, **not** `dialog` for the album grid (UX-DR modals section vs FR-21). |
| **Story 2.9 boundary** | **No** collection create/rename/delete flows required here — list/browse only; CRUD is **2.9**. |

### Technical requirements

- **Shell:** Today `internal/app/shell.go` wires `NewSectionPlaceholder("Collections", …)` — replace with real view; preserve **`clearReviewUndoIfLeftReview`** behavior when switching **away from Review** (unchanged keys: `"review"` vs `"collections"`).
- **Queries:** Parameterize **`collection_id`**; keep SQL out of `internal/app` (architecture §5.2).
- **Grouping default:** **Stars** per FR-23 when detail opens.

### Architecture compliance

- **§3.3 / §3.1:** SQLite + relational collections; indexes on `asset_collections(collection_id)` already exist.
- **§3.1 table:** Collections FR-15–FR-25 → relational model + **grouping queries** — this story implements the **full-page + grouping** slice.
- **§5.2:** `app → domain, store`; grouping modes in **domain**; SQL in **store**.

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3** (`go.mod`): lists, buttons, borders, scroll, standard controls for grouping strip. **`widget.RadioGroup` caveat:** re-selecting the **already-selected** nav label may **not** invoke the callback — AC12 may require an explicit **“list home”** path (e.g. wrap nav behavior in `shell.go`) so “Collections” always returns to list from detail.
- **EXIF:** Existing `github.com/dsoprea/go-exif/v3` usage in `internal/exifmeta` — extend carefully to avoid binary size / perf surprises; read tags needed for make/model only.

### File structure requirements

- **Extend:** `internal/app/shell.go` — swap Collections panel constructor.
- **New (expected):** `internal/app/collections.go` (or `collections_view.go`) — list + detail UI state machine.
- **Extend:** `internal/store/` — new query file(s) for collection detail; **new migration** for camera fields.
- **Extend:** `internal/exifmeta/` and/or `internal/ingest/` — populate camera fields on write.
- **Extend:** `internal/domain/` — grouping mode type.

### Testing requirements

- **Mandatory:** Store integration tests with temp SQLite DB (patterns in `review_query_test.go`, `store_test.go`).
- **Regression:** `ReviewBrowseBaseWhere` meaning unchanged; collection detail cannot show rejected or soft-deleted assets.

### Previous story intelligence (2.7 / 2.3)

- **2.7:** `deleted_at_unix` exclusion is non-negotiable in browse queries.
- **2.3:** Grid uses **paging** and bounded pixmap loading — mirror for collection detail grid.
- **2.2:** `ReviewFilters` + `ReviewFilterWhereSuffix` pattern — for **collection detail**, predicate is **fixed** collection membership + base where; do not silently fork filter semantics.

### Latest technical information

- Fyne v2.7: prefer **`widget.List`** or **`widget.Table`** for collection list; reuse **`container.Border`** / **`container.Scroll`** for detail layout consistency with Review section.

### Project structure notes

- No `project-context.md` in repo — rely on architecture + this story.
- **Epic 2.10** will add quick-assign from grid — design collection detail grid so future actions can attach without rewriting grouping (avoid monolithic anonymous closures if a small struct helps).

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Story 2.8, FR-21–FR-24]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-21–FR-24, FR-26]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.3 entities, §3.1 Collections row, §5.2 boundaries, §5.3 FR-15–FR-25 → SQL + app]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — Component7 Collections list/detail shell, Navigation patterns (Collection detail full page), Modals vs main browsing]
- [Source: internal/app/shell.go — Collections placeholder to replace]
- [Source: internal/store/review_query.go — `ReviewBrowseBaseWhere`, `ReviewGridRow`, ordering]
- [Source: internal/store/collections.go — `ListCollections`, membership helpers]
- [Source: internal/store/migrations/002_collections.sql — schema]
- [Source: internal/exifmeta/capture.go — EXIF patterns for extension]

## Party mode (create session 1/2, 2026-04-13)

**Roster:** `_bmad/_config/agent-manifest.csv` not present in workspace — simulated round (solo party mode per skill `--solo`).

**John (PM):** AC6 alone is ambiguous: "no linked assets" vs "all linked assets filtered out by reject/delete" should feel the same to the user — both are an empty album. We were missing **zero collections in the library** (now AC7). Pin **invalid collection id** so a deleted row never takes down the shell.

**Winston (Architect):** I would not ship v1 with heavy SQL `GROUP BY` plus offset pagination — brittle across SQLite and tests. Prefer one **ordered SELECT** with `ReviewBrowseBaseWhere` + membership, then **group in Go** until profiling says otherwise; document paging (AC10). Centralize camera strings in a **`CameraLabel`-style helper**; expose **`CollectionExists`** (or equivalent) so the UI does not scrape driver errors.

**Sally (UX):** Detail stays **full-page in the content region**, not a dialog stack. Grouping is a frequent control — keep it as discoverable as the filter strip, not buried in a menu. AC7 copy must not **over-promise** Story 2.9 CRUD; stay neutral.

**Murat (TEA):** Cover **5→1** star sections, empty buckets skipped, **Unrated last**; **newest day first**; **camera** unknown bucket omitted when empty; **invalid / deleted collection id**; two assets on the **same local calendar day**. Regression: collection browse uses the **same** `ReviewBrowseBaseWhere` contract as `ListAssetsForReview`.

**Orchestrator synthesis — applied in this file:** Former single regression criterion is **AC11**; added **AC7–AC10** (list empty, visibility-only, invalid id, paging); camera label construction rules; risks/tasks/DoD/tests aligned; `sprint-status.yaml` audit line added.

---

## Party mode (create session 2/2, 2026-04-13) — challenge pass

**Roster:** manifest still absent — simulated round; voices rotated toward **contradiction** with session 1 defaults.

**Winston (Architect):** I partially disagree with my own prior take: “ordered SELECT + group in Go” is **not** sufficient unless you say **how** paging walks that stream. **Offset pagination on a flat sort** will **tear groups in half** unless you use **keyset pagination**, **per-group cursors**, or **per-section queries**. Session 1’s AC10 waved at “document strategy”; that’s too soft — we need an **explicit disallowed pattern** (naive global `OFFSET` without a grouping-safe strategy) or we’ll ship broken headers.

**John (PM):** Scope check: **Collections list** should stay **cheap** for 2.8 — **no** requirement to show **visible photo counts** per row (that’s analytics-ish and duplicates filter semantics). If dev wants a subtitle later, fine, but **not** an AC here. **Invalid id** should be **`errors.Is(ErrCollectionNotFound)`**, not ad-hoc strings — PM cares that support logs are clean.

**Sally (UX):** **Disagreement with “Back only”**: users will **re-click Collections** in the left nav expecting to “start over” at the list. Today’s `RadioGroup` often **does nothing** when you click the **already selected** item — that’s a **trap**. AC12 forces **nav reset**; implementation detail belongs in dev notes, not user-facing copy.

**Murat (TEA):** Add a **hard test hook** for **NULL vs5-star ordering** — SQLite `ORDER BY rating DESC` can surprise people on `NULL`. **Camera label**: **whitespace-only** make with real model must not produce `" Canon"` buckets. **Paging test** doesn’t need 10k rows — **12 assets** and a **page size of 5** can prove **no split section** if the strategy is wrong.

**Amelia (Dev):** `CollectionExists` + `ErrCollectionNotFound` should be **the same public contract** as review store patterns (wrapped errors, no `panic`). For AC10 option **(B) per-section**, define **cap** behavior: infinite scroll per section vs global — pick one in code comment **now** to avoid Epic 2.10 rework.

**Orchestrator synthesis — applied in this file:** Strengthened **AC10** with options (A/B/C) and **anti-pattern** callout; added **AC12** nav reset; pinned **`NULL`** star ordering under AC3; camera **whitespace + UTF-8** clarifications; **ErrCollectionNotFound** requirement; new **risks**; DoD/tasks/tests updated for AC12 + paging guardrail test; Fyne **RadioGroup** caveat in dev notes.


---

## Party mode (dev session 1/2, 2026-04-13)

**Roster:** `_bmad/core/config.yaml` and `_bmad/_config/agent-manifest.csv` absent — **solo simulation** (`--solo`), dev-hook round (spec already landed; focus **implementation hardening**).

**Amelia (Dev):** I'm not signing off until we fix init order: `reloadCollectionRows` ran **before** `widget.NewList`, and the AC7 patch called `Hide()` on a **nil** list — that's a footgun for anyone who adds more preload logic. Move the first reload to **after** the list exists, or guard every visibility call.

**Murat (TEA):** AC12 is still "trust me, the button fires" in prose. Extract the `prev==collections && next==collections` predicate to a **pure function** and unit-test it. AC10's star paging test is great; **day buckets** use the same per-section OFFSET idea — mirror the regression so we don't only protect one grouping mode.

**Sally (UX):** An empty library with a **visible, zero-row** `widget.List` reads like a broken control. Hide the list when there are no albums (and on load failure); let the empty-state copy carry the screen.

**Winston (Architect):** Predicate belongs beside `clearReviewUndoIfLeftReview` with a one-line **why** (AC12). Keeps shell readable and avoids scattering stringly `"collections"` checks later.

**Orchestrator synthesis — applied in repo:** Reordered `NewCollectionsView` so the first `reloadCollectionRows` runs after list construction; `reloadCollectionRows` nil-guards list visibility for safety; **hide list** on zero albums / list error; added `collectionsNavShouldResetToList` + `TestCollectionsNavShouldResetToList_AC12`; `TestCollectionsView_zeroAlbums_emptyStateHidesList` (Fyne `test` driver); `TestCollectionDetail_daySectionPagingDoesNotMixDays`; DoD checkboxes updated; this section recorded.

---

## Party mode (dev session 2/2, 2026-04-13)

**Roster:** `_bmad/_config/agent-manifest.csv` still absent — **solo simulation** (`--solo`), dev-hook round **challenging dev 1/2** (no repeat of list init / day test narrative).

**Murat (TEA):** Session 1 added **day** paging next to **stars**; **camera** grouping used the same store primitive but had **no** `ListCollectionCameraSectionPage` regression. That is an easy future footgun—mirror the star/day proof for **two labels + OFFSET**.

**Winston (Architect):** Option **(B)** was coded, but the **cap model** lived only in heads: who owns cursors, per-section vs global. **Document beside the strategy** how scrolling maps to OFFSET so Epic 2.10 does not reinterpret “paging” as a redesign.

**Amelia (Dev):** `replaceDetailBody` and `CountCollectionVisibleAssets` handle **ErrCollectionNotFound**, but **lazy** `ListCollection*SectionPage` during scroll could still return that sentinel after a concurrent delete—users would see a **generic page error** and stay on a dead detail shell. **Plumb** not-found from the section grid fetch to the same **ResetToList + short message** path as the other AC9 handlers; **sync.Once** so a storm of failing cells does not loop.

**Sally (UX):** If the album vanishes mid-session, **one** calm dismissal beats a grid full of “can’t load” tiles. Align with the copy already used when `CollectionExists` fails on open.

**Orchestrator synthesis — applied in repo:** `TestCollectionDetail_cameraSectionPagingDoesNotMixLabels`; extended **AC10** store comment with **scroll/cap** contract; `collectionSectionGrid` **staleOnce** + `onStaleCollection` → `ResetToList` when page fetch is **`errors.Is(…, ErrCollectionNotFound)`**; DoD + story changelog updated; `sprint-status.yaml` audit line.

---

## Dev Agent Record

### Agent Model Used

Cursor AI agent

### Debug Log References

_(none)_

### Completion Notes List

- Collections: full-page list + detail in `internal/app/collections.go`; shell uses primary **buttons** (not RadioGroup) so re-tapping **Collections** always fires and calls `ResetToList` (AC12).
- Store: migration `005_camera_meta.sql` adds `camera_make`, `camera_model`, `camera_label`; `InsertAssetWithCamera` + `exifmeta.ReadCamera` on ingest/import; `collection_detail.go` implements **per-section LIMIT/OFFSET** paging (AC10 option B) with `ReviewBrowseBaseWhere` + membership for all queries; `ErrCollectionNotFound` on missing collection id.
- Grouping: stars5→1 + Unrated last (empty buckets omitted), days via `strftime(..., 'localtime')`, cameras via persisted `camera_label` with unknown (NULL) bucket last; UI label `Unknown camera` for NULL rows.
- Tests: `collection_detail_test.go` covers ordering, invisible members, invalid id, same calendar day, camera sections, star + **day** + **camera** paging guardrails; `domain/collection_grouping_test.go` for parse; `store_test.go` schema version 5; `collections_test.go` + `shell_test.go` for AC7/AC12 automation (party dev1/2).
- **Manual (AC7 list empty / AC12):** Smoke-test in app: zero albums copy; open album → Back; open album → tap **Collections** again → returns to list. Optional full manual per story DoD checklist.

### File List

- internal/store/migrations/005_camera_meta.sql
- internal/store/migrate.go
- internal/store/camera_label.go
- internal/store/collection_detail.go
- internal/store/assets.go
- internal/store/store_test.go
- internal/store/collection_detail_test.go
- internal/exifmeta/camera.go
- internal/domain/collection_grouping.go
- internal/domain/collection_grouping_test.go
- internal/ingest/ingest.go
- internal/ingest/register_import.go
- internal/app/collections.go
- internal/app/shell.go
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/2-8-collections-list-detail.md

### Change Log

- 2026-04-13: Story 2.8 implemented — collections UI, camera metadata + grouping queries, tests, sprint/story status → review.
- 2026-04-13: Party mode **dev** 1/2 — empty-library list hidden, nav-reset predicate + tests, day-section paging regression, `NewCollectionsView` init order fix.
- 2026-04-13: Party mode **dev** 2/2 — camera-section paging regression test; AC10 scroll/cap comment; collection grid returns to list on `ErrCollectionNotFound` during lazy fetch (stale album mid-scroll).

