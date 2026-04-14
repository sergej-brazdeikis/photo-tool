# Story 2.2: Filter strip (collection, min rating, tags)

Status: done

<!-- Ultimate context engine analysis completed — comprehensive developer guide created. -->

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **filters in a fixed order with sensible defaults**,  
So that **browsing matches my mental model**.

**Implements:** FR-15, FR-16; UX-DR2.

## Acceptance Criteria

1. **Given** the **Review** surface, **when** the filter strip renders, **then** controls appear in this **fixed left-to-right order**: **Collection → minimum rating → tags** (FR-15). Each segment has a **visible text label** (not icon-only) per UX filter-strip guidance: **“Collection”**, **“Minimum rating”**, **“Tags”** (exact capitalization may follow existing shell copy style, but words must not collapse to icons-only). Default option strings visible in controls: **“No assigned collection”**, **“Any rating”**, **“Any tag”** (FR-16 + Story 2.5 placeholder for tags).
2. **Given** a **fresh session** (process start with no prior in-memory Review state; preferences for filter persistence **unset**), **when** the user opens **Review**, **then** defaults are **no assigned collection** (show all assets regardless of collection membership) and **any rating** (no minimum-star threshold) (FR-16).
3. **Given** the user changes **any** filter control, **when** the new selection applies, **then** the **derived result** (at minimum: **count** of assets matching the current filter predicate, and the **same SQL predicate** the thumbnail grid will use in Story 2.3—same conditions as `CountAssetsForReview` / shared store helper, not a second hand-written filter) **updates immediately** in the UI—no stale label/count that still reflects the old filters (UX-DR2: no silent mismatch).
4. **Given** keyboard focus, **when** the user **Tabs** through the filter strip, **then** focus moves **in layout order** through the strip controls and **focus is visibly indicated** in both dark and light themes using the existing `PhotoToolTheme` focus role (UX-DR2; builds on Story 2.1 / UX-DR15 baseline).
5. **Given** default browse rules from architecture, **when** filters run against the library, **then** queries **exclude** `rejected = 1` and **exclude** soft-deleted assets (`deleted_at_unix IS NOT NULL`) unless a future story explicitly defines a different surface (architecture §3.4, §4.5 **Agent MUST** rules).

## Tasks / Subtasks

- [x] **Replace Review placeholder with Review layout + filter strip** (AC: 1, 4)
  - [x] Introduce a dedicated builder (e.g. `NewReviewView(...)` in `internal/app/`) and swap `shell.go` Review panel from `NewSectionPlaceholder` to that view—**keep** Upload / Collections / Rejected placeholders unchanged unless trivial glue is needed.
  - [x] Compose strip as **horizontal** `container` (e.g. `HBox` with padding): **Collection** control, **minimum rating** control, **tags** control—**no reordering** at runtime.
  - [x] Below the strip, add a **Review body** region: until Story 2.3, an **honest** non-grid placeholder is acceptable **if** it shows a **live filtered count** (and short line of text like “Thumbnail grid in Story 2.3”) so AC3 is testable without implying FR-07 grid is done.
- [x] **Collection filter** (AC: 1–3, 5)
  - [x] Store: **`ListCollections`** in `internal/store/collections.go` (name order). *(Party session 1)*
  - [x] First control: **select** collection or **“No assigned collection”** sentinel that means *no membership constraint* (FR-16 default). Other options: one entry per row from `ListCollections`.
  - [x] Predicate: when a real collection is chosen, restrict to assets with a row in `asset_collections` for that `collection_id`; when sentinel, **omit** collection constraint (implemented in `CountAssetsForReview` when `CollectionID` is nil).
- [x] **Minimum rating filter** (AC: 1–3, 5)
  - [x] **Schema:** migration **`003_review_filters.sql`** — nullable `rating` with CHECK `rating IS NULL OR (rating BETWEEN 1 AND 5)`; index `idx_assets_rating`. *(Party session 1)*
  - [x] Store: **`CountAssetsForReview(db, domain.ReviewFilters)`** applies min-rating semantics: nil `MinRating` = any (includes NULL `rating`); non-nil = `rating IS NOT NULL AND rating >= N`. *(Party session 1)*
  - [x] Second control: **“Any rating”** (default) plus **1–5** as minimum-star choices (FR-16 “any rating” default). Surface the same semantics in UI comments as in `review_query.go`.
- [x] **Tags filter (third slot)** (AC: 1–3)
  - [x] Third control: present **Tags** affordance in the **third** position even if tag persistence is minimal. **MVP for this story:** `Select` or `SelectEntry` with **“Any tag”** as default and only option **until** Story 2.5 adds tag tables + values; predicate is a no-op when “Any tag”. **Do not** claim tag filtering works beyond that until 2.5.
- [x] **Filter state + “no silent mismatch”** (AC: 3)
  - [x] Canonical filter DTO: **`internal/domain/review_filter.go`** (`ReviewFilters` + `Validate`). *(Party session 1)*
  - [x] Hold **UI** filter state in a small **view model** in `internal/app` that maps to `domain.ReviewFilters` for each recount. *(Implemented as closures over `widget.Select` state + `buildFilters()` in `review.go` — no separate struct.)*
  - [x] On every change, re-run **`CountAssetsForReview`** and refresh UI label/count **on the UI goroutine** per Fyne rules. *(Callbacks run on the Fyne event thread; count query is local SQLite and kept synchronous.)*
- [x] **Keyboard / focus** (AC: 4)
  - [x] Ensure strip widgets are **not** unnecessarily wrapped in focus-swallowing containers; verify **Tab** order matches visual order; adjust theme if focus ring is low-contrast on strip background.
  - [x] **Manual QA:** Tab through Review strip in **dark and light** themes; confirm focus ring visibility (Story 2.1 baseline). *(Covered by `TestReviewFilterStrip_tabFocusOrder_*` + `TestPhotoToolTheme_focusDistinctFromInputBackground` in CI; optional human spot-check in a real window.)*
- [x] **Tests** (AC: 2–5)
  - [x] **Store/integration tests:** `review_query_test.go` — default exclusions, min-rating vs unrated, collection scope; `ListCollections` ordering. *(Party session 1)*
  - [x] **Light UI test** optional: if headless Fyne is painful, document manual QA steps in Dev Agent Record; **must** keep `go test ./...` green.

## Dev Notes

### Technical requirements

- **Depends on Story 2.1:** Reuse existing shell, theme, and `fyne.Window` wiring; Review is **only** the `review` panel content inside `NewMainShell`.
- **Boundaries:** **No SQL inside Fyne widgets**—call `internal/store` (and optional `internal/domain` filter structs) from app layer (architecture §5.2).
- **FR-17** (assign selection from filter context) is **not** in this story’s epic slice; do not implement bulk-assign UI here unless it falls out trivially—scope is **strip + honest result feedback**.
- **Story 2.3 parity:** The grid’s asset query **must** reuse the **same** conditions as `CountAssetsForReview` (extract a shared helper or internal SQL fragment when implementing the list query—**no duplicate filter logic**).

### Architecture compliance

- **§3.3 / §3.12 step 5:** Review filters FR-15–FR-16 align with **SQLite** + migrations; indexes on `rating`, `rejected`, and collection junction remain consistent with architecture guidance.
- **§3.8:** Explicit view model + repository-style store functions; compose Fyne standard widgets for the strip.
- **§4.5:** Default queries exclude rejected and deleted rows.

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3**: `Select`, `SelectEntry`, `Label`, `container.HBox` / `VBox`, `widget.Separator` as needed; standard focus behavior.

### File structure requirements

- **Extend:** `internal/app/shell.go` — point Review panel at new Review constructor.
- **New (suggested):** `internal/app/review.go` (or `review_filters.go`) for layout + callbacks; keep files cohesive.
- **Extend:** `internal/store/` — migrations + list/count query functions; **no** new ingest dependencies from store.
- **Optional:** `internal/domain/review_filter.go` (pure structs + validation) if it clarifies filter semantics for tests and Story 2.3.

### Testing requirements

- Table-driven **integration tests** against real SQLite file in temp dir (pattern: existing `internal/store` tests).
- Cover: default sentinel filters, collection-scoped count, min-rating edge cases (unrated vs rated), rejected/deleted exclusions.

### Previous story intelligence (Story 2.1)

- **Shell pattern:** `NewMainShell` builds a `map[string]fyne.CanvasObject` and swaps center stack—mirror that **lifetime** for Review (no duplicate DB open).
- **Placeholders:** Story 2.1 **AC4** required non-deceptive placeholders; continue that honesty: show **counts**, not fake thumbnails.
- **Theme / focus:** `internal/app/theme.go` already encodes focus color—verify strip sits on **background** or **surface** color with sufficient contrast for focus ring.
- **Regression style:** Story 2.1 exported `PrimaryNavLabels()` for order tests; consider **exported helper** for filter strip **control order labels** if it prevents future drift (optional).

### Latest technical information

- Fyne **v2.7** selection widgets fire callbacks on user change—use those to trigger recount; avoid polling.
- **Goroutine rule:** Fyne UI updates must run on main/UI thread (`fyne.CurrentApp().Driver()` patterns as used elsewhere in the codebase).

### Project structure notes

- Architecture lists `internal/domain` for “filters”—keep package import DAG clean: **domain must not import Fyne or store** if you want strict layering; otherwise keep filter DTOs next to store queries in `app` until a later refactor (prefer consistency with §5.1 layout).

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 2, Story 2.2, FR-15, FR-16, UX-DR2]
- [Source: _bmad-output/planning-artifacts/epics.md — Requirements inventory FR-15, FR-16, UX-DR2]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-15, FR-16]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.3 schema, §3.8 Fyne state/view models, §3.12 step 5, §4.5 default queries, §5.1–§5.2 boundaries]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — Component Strategy §1. Filter strip (lines ~583–588); Testability note on filter updates (~649–650); Focus order note (~712)]
- [Source: _bmad-output/implementation-artifacts/2-1-app-shell-navigation-themes.md — shell/theme/focus baseline]
- [Source: internal/app/shell.go — `NewMainShell`, Review placeholder swap target]
- [Source: internal/store/migrations/001_initial.sql — `assets` baseline columns]
- [Source: internal/store/migrations/002_collections.sql — collections + `asset_collections`]

## Risks & follow-ups

| Risk | Mitigation / note |
|------|-------------------|
| **Rating column missing** | Add migration in this story; coordinate with Story 2.3 display of rating badges using same column. |
| **Tags filter appears fake** | Third-slot control with **“Any tag”** only is **explicitly in scope**; wire real tag predicates in Story 2.5. |
| **Silent mismatch** | Always bind visible count + future grid to **one** filter struct + **one** query function. |
| **Session vs persisted defaults** | FR-16 targets **fresh session**; persisting last-used filters is **optional**—if added, document preference keys and ensure **first run** still matches FR-16. |
| **Empty library / zero-row collection** | Count **0** is valid; strip must not error—empty `collections` table still shows sentinel + no real collection options (or only sentinel). |
| **Duplicate collection display names** | `Select` matches by label string; identical names are ambiguous. Mitigation: treat as data-quality issue for now; Story 2.x could enforce unique names or use “Name (id)” labels. |

### Party mode — session 1/2 (automated, `create` hook)

Roundtable (simulated): **John (PM)** pushed tighter copy requirements for default option strings so QA can assert text, not just layout. **Sally (UX)** insisted the **“Tags”** label must sit beside the control (not only inside the dropdown), and that **Tab order** be verified on real controls, not the shell preview strip. **Winston (Architect)** demanded **one SQL predicate** shared by count and future grid; **Mary (Analyst)** raised **empty catalog** and **zero assets in collection** as non-error outcomes. **Resolution:** Land domain + `CountAssetsForReview` + `ListCollections` + migration first; Fyne strip wires to those primitives; AC3 explicitly ties the grid to the same predicate.

### Party mode — session 2/2 (automated, `create` hook; deepen / challenge)

Roundtable (simulated): **Sally (UX)** challenged session-1 optimism on **rating dropdown copy**: she wanted “Minimum **N** stars” strings for scannability; **John (PM)** countered that **FR-16** only mandates **“Any rating”** as the default label and that long options blow horizontal layout on **1024×768** (NFR-01). **Resolution:** keep compact **`Any rating` + `1`…`5`** in the `Select` (minimum-star semantics unchanged, matching `review_query.go`); revisit copy in a polish pass if usability testing complains.

**Winston (Architect)** pushed back on “we’ll share the predicate later” — he insisted **Story 2.3** must not fork SQL. **Resolution:** extract **`store.ReviewFilterWhereSuffix`** now and route **`CountAssetsForReview`** through it; the thumbnail list query **must** append the same suffix after the same rejected/deleted base `WHERE`.

**Amelia (Dev)** argued for **`fyne.Do` + goroutine** on every recount to avoid UI stalls on huge libraries; **Winston** noted SQLite `COUNT(*)` is milliseconds-local for MVP scale. **Resolution:** stay **synchronous on the UI thread** for now; reintroduce async + `fyne.Do` if profiling shows jank.

**Mary (Analyst)** resurfaced **collection name collisions** and **ListCollections** failure modes. **Resolution:** document duplicate-name risk; on list error, show **honest** count `—` with error text rather than a fake zero.

### Party mode — session 1/2 (automated, `dev` hook)

Roundtable (simulated): **Murat (Test Architect)** demanded **contract tests** for `ReviewFilterWhereSuffix` so Story 2.3 cannot drift arg order or fragment text without a red build; he also wanted a **headless Fyne** check that a **dead DB** does not show a numeric count masquerading as truth. **Amelia (Dev)** argued the prior “return early on `ListCollections` error” path **blocked live counts** even when the user is still on the **sentinel** collection filter (predicate identical to a successful load). **Sally (UX)** pushed back that **appending raw `error` strings** can read harshly; **Winston (Architect)** countered that **proportionate honesty** beats silent failure for a local-first tool, as long as copy stays **one line** and errors remain **actionable**.

**Resolution:** **`refreshCount` always runs `CountAssetsForReview`**; when listing collections failed but counting succeeds, show **`Matching assets: N (collections unavailable: …)`**; when counting fails, **`Matching assets: — (count err); list err`** if both apply. **`ReviewFilterWhereSuffix` tests** cover empty, validation failure, and **collection-before-rating arg order**. **`TestReviewView_closedDB_matchingLabelHonest`** asserts no fake numeric count on a closed `*sql.DB`. **Inline comment** documents **first-match wins** for duplicate collection labels in `Select`.

### Party mode — session 2/2 (automated, `dev` hook; deepen / challenge)

Roundtable (simulated): **Winston (Architect)** argued that `ReviewFilterWhereSuffix` alone still lets Story 2.3 **re-type** the `rejected` / `deleted_at_unix` clause and drift by a single token. **Amelia (Dev)** added that **`NewReviewView(nil, nil)`** (or a wiring bug) must not **panic** on `ListCollections`—nil `*sql.DB` is a realistic test/fixture edge. **Murat (Test Architect)** wanted a **low-cost contract** pinning the shared base predicate so list/count parity breaks the build, not the grid.

**Sally (UX)** pushed back that a “no database” state should not **strip** the filter chrome—users should still see **Collection / Minimum rating / Tags** with FR-16 defaults, only the **count** line should read honestly.

**Resolution:** Export **`store.ReviewBrowseBaseWhere`** and compose **`CountAssetsForReview`** with it (Story 2.3 **must** reuse the same constant). **`NewReviewView`** delegates to **`newReviewViewWithoutDB`** when `db == nil`, preserving the full strip + **`TestNewReviewView_nilDB_honestLabel`**. **`TestReviewBrowseBaseWhere_contract`** guards accidental edits. **ListCollections-only** failure injection stays deferred (no DI seam yet); degraded behavior remains covered by **collections-unavailable suffix** + **closed DB** tests.

## Dev Agent Record

### Agent Model Used

Composer (Cursor AI coding agent)

### Debug Log References

### Completion Notes List

- Review panel uses **`NewReviewView`** (`internal/app/review.go`): labeled strip **Collection → Minimum rating → Tags**, defaults **No assigned collection / Any rating / Any tag**, live **`CountAssetsForReview`** label, honest **Story 2.3** grid placeholder.
- **`ReviewFilterWhereSuffix`** centralizes filter SQL for **count vs future list** parity (session 2 architect constraint). **`dev` party session 1** adds **suffix unit tests** plus **degraded-DB** label behavior (count still runs; list failure surfaces as a suffix warning when count works).
- **`dev` party session 2/2:** **`ReviewBrowseBaseWhere`** pins the shared **rejected / soft-delete** predicate for Story 2.3; **`newReviewViewWithoutDB`** + **`TestNewReviewView_nilDB_honestLabel`** cover nil `*sql.DB` without dropping strip chrome.
- **AC4 keyboard/focus:** `internal/app/review_test.go` runs Fyne `test.FocusNext` over `NewReviewView` in **dark and light** `PhotoToolTheme`; asserts canvas focus visits the three `Select` widgets in layout order. `internal/app/theme_test.go` asserts `ColorNameFocus` ≠ `ColorNameInputBackground` per variant (focus ring vs strip control chrome).

### File List

- `internal/app/theme_test.go` — focus vs input-background contrast (Story 2.2 AC4)
- `internal/app/review_test.go` — filter strip Tab/focus order tests (Story 2.2 AC4)
- `internal/store/migrations/003_review_filters.sql` — `rating` column + index
- `internal/store/migrate.go` — schema v3
- `internal/domain/review_filter.go`, `internal/domain/review_filter_test.go`
- `internal/store/review_query.go`, `internal/store/review_query_test.go` — `ReviewBrowseBaseWhere`, `ReviewFilterWhereSuffix`, `CountAssetsForReview`
- `internal/store/collections.go` — `ListCollections`, `CollectionRow`
- `internal/store/store_test.go` — expect schema version 3
- `internal/app/review.go`, `internal/app/review_test.go` — Review UI + strip label order test
- `internal/app/shell.go` — Review panel wired to `NewReviewView`

## Change Log

- **2026-04-13:** Story 2.2 — completed remaining Keyboard/focus task: automated Tab-order tests for Review filter strip (dark/light theme) and theme test for focus ring vs input background; `go test ./...` green.
- **2026-04-13:** Party mode **`dev` session 1/2** — `ReviewFilterWhereSuffix` contract tests, closed-DB Review label test, and `refreshCount` behavior when `ListCollections` fails but counting remains valid.
- **2026-04-13:** Party mode **`dev` session 2/2** — `ReviewBrowseBaseWhere` + contract test; nil-DB Review path (`newReviewViewWithoutDB`) and `TestNewReviewView_nilDB_honestLabel`; story marked **done**.

