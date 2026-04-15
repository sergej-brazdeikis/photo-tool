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

*Aligned with `_bmad-output/planning-artifacts/epics.md` — Story 2.2 (FR-15, FR-16, UX-DR2).*

1. **Given** the **Review** surface, **when** filters render, **then** order is **Collection → minimum rating → tags** (FR-15).  
   - **Elaboration:** Fixed **left-to-right** order only; each segment has a **visible text label** (not icon-only): **“Collection”**, **“Minimum rating”**, **“Tags”** (capitalization may follow shell copy, but not icon-only). Default option strings in controls: **“No assigned collection”**, **“Any rating”**, **“Any tag”** (tags slot may stay “Any tag” only until Story 2.5).
2. **Given** a **fresh session**, **when** Review opens, **then** defaults are **no assigned collection** and **any rating** (FR-16).  
   - **Elaboration:** *Fresh session* = process start with no prior in-memory Review state and filter persistence **unset**; “no assigned collection” = **no** membership constraint (sentinel, not a DB collection row); “any rating” = no minimum-star threshold (includes NULL `rating`). Third control defaults to **“Any tag”** per strip contract / Story 2.5 handoff.
3. **Given** the user changes filters, **when** applied, **then** the result set updates **without silent mismatch** (UX-DR2).  
   - **Elaboration:** Visible **count** (and any result summary) must match the **same** filter predicate as **`CountAssetsForReview` / `ReviewFilterWhereSuffix` / `ReviewBrowseBaseWhere`** and the grid / **`ListAssetsForReview` / `ListAssetIDsForReview`** surfaces—no second hand-written filter.
4. **And** keyboard users can traverse the strip with **visible focus** (UX-DR2).  
   - **Elaboration:** **Tab** order follows layout order within the strip; **focus** is visible in **dark** and **light** `PhotoToolTheme` (Story 2.1 baseline). **UX-DR15** end-to-end order (**strip → grid → loupe**) is satisfied once Stories **2.3–2.4** ship; this story only **locks** strip-internal traversal + theme contrast.
5. **And** the strip occupies **at most one** default **row** of controls; additional filters or sort/scope controls use **overflow** (sheet, drawer, or equivalent)—**not** a **second** horizontal **nav** (UX-DR2).  
   - **Elaboration:** MVP may not add extra controls yet; still **document** the overflow pattern for the next control and avoid introducing a second full-width filter/nav row when extending Review. At **NFR-01** minimum width (**1024×768**), if combined labels + `Select` chrome risk clipping, **prefer** shorter copy or a **single** overflow affordance—**not** a second horizontal control row.

### Architecture guardrail (default browse predicate)

- **Given** default browse rules, **when** filters run against the library, **then** queries **exclude** `rejected = 1` and **exclude** soft-deleted assets (`deleted_at_unix IS NOT NULL`) unless a story explicitly defines a different surface ([Source: `_bmad-output/planning-artifacts/architecture.md` — §4.5 **Agent MUST**; implemented via `ReviewBrowseBaseWhere` + shared review queries]).

## Tasks / Subtasks

- [x] **Replace Review placeholder with Review layout + filter strip** (AC: 1, 4, 5)
  - [x] Introduce a dedicated builder (e.g. `NewReviewView(...)` in `internal/app/`) and swap `shell.go` Review panel from `NewSectionPlaceholder` to that view—**keep** Upload / Collections / Rejected placeholders unchanged unless trivial glue is needed.
  - [x] Compose strip as **horizontal** `container` (e.g. `HBox` with padding): **Collection** control, **minimum rating** control, **tags** control—**no reordering** at runtime.
  - [x] **UX-DR2 layout:** default strip stays **one row**; defer extra sort/scope/advanced filters to **overflow** (sheet / drawer / “More filters”)—do **not** add a second horizontal nav row (document pattern for next controls).
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
- [x] **Tests** (AC: 2–5 + default-browse guardrail)
  - [x] **Store/integration tests:** `review_query_test.go` — default exclusions, min-rating vs unrated, collection scope; `ListCollections` ordering. *(Party session 1)*
  - [x] **Light UI test** optional: if headless Fyne is painful, document manual QA steps in Dev Agent Record; **must** keep `go test ./...` green.

## Dev Notes

### Spec authority (epic vs story)

- **`epics.md` §2.2** states the **headline** acceptance bar for stakeholders; **this story file** carries elaborations (SQL parity, sentinel strings, degraded DB copy, Story 2.5 handoffs). If they diverge, **update both**—the epic must not imply only two default controls when the strip has three slots (FR-15 order).

### UX-DR16 scope fence (vs UX-DR2 in this story)

- **UX-DR2 (this story):** fixed strip order, defaults, keyboard focus, **one row** + **overflow** for extra controls—no second horizontal nav row.
- **UX-DR16 (Story 2.11 / NFR-01 evidence):** measured **combined top-chrome height budget**, **minimum thumbnail edge**, **minimum loupe image region** at reference layouts. **Not** a Story 2.2 acceptance gate; do not treat filter-strip implementation as proof of those numeric thresholds.

### AC3 — strip selections → `domain.ReviewFilters` (no silent mismatch)

| Strip control | UI sentinel / value | `ReviewFilters` field |
|---------------|---------------------|------------------------|
| Collection | “No assigned collection” | `CollectionID == nil` (no membership constraint) |
| Collection | specific album label | `CollectionID` non-nil |
| Minimum rating | “Any rating” | `MinRating == nil` (includes NULL `rating` in store) |
| Minimum rating | `1`…`5` | `MinRating` non-nil |
| Tags | “Any tag” | `TagID == nil` (no tag predicate until Story 2.5 lists real tags) |
| Tags | specific tag (2.5+) | `TagID` non-nil |

Fresh-session defaults = **all three pointers unset** (`ReviewFilters` zero value); see **`TestReviewFilters_FR16DefaultMeansUnconstrained`**.

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
- **Domain:** `TestReviewFilters_FR16DefaultMeansUnconstrained` pins the **all-nil** `ReviewFilters` default (three “any” dimensions) alongside UI sentinel string tests in `internal/app/review_test.go`.

### Previous story intelligence (Story 2.1)

- **Shell pattern:** `NewMainShell` builds a `map[string]fyne.CanvasObject` and swaps center stack—mirror that **lifetime** for Review (no duplicate DB open).
- **Placeholders:** Story 2.1 **AC4** required non-deceptive placeholders; continue that honesty: show **counts**, not fake thumbnails.
- **Theme / focus:** `internal/app/theme.go` already encodes focus color—verify strip sits on **background** or **surface** color with sufficient contrast for focus ring.
- **Regression style:** Story 2.1 exported `PrimaryNavLabels()` for order tests; Story 2.2 exports **`ReviewFilterStripSegmentLabels()`** (`internal/app/review.go`) + **`TestReviewFilterStripSegmentLabels_order`**; default option sentinels are pinned by **`TestReviewFilterStrip_defaultSentinels_matchStory22`**.

### Latest technical information

- Fyne **v2.7** selection widgets fire callbacks on user change—use those to trigger recount; avoid polling.
- **Goroutine rule:** Fyne UI updates must run on main/UI thread (`fyne.CurrentApp().Driver()` patterns as used elsewhere in the codebase).

### Project structure notes

- Architecture lists `internal/domain` for “filters”—keep package import DAG clean: **domain must not import Fyne or store** if you want strict layering; otherwise keep filter DTOs next to store queries in `app` until a later refactor (prefer consistency with §5.1 layout).

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 2, Story 2.2, FR-15, FR-16, UX-DR2]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-15, FR-16]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.3 schema; §3.8 / §3.8.1 Fyne state, layout ↔ async; §3.12 step 5 (Review queries FR-15–FR-16); §4.5 default queries; §5.1–§5.2 boundaries]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — § “Filter strip” (~664+); filter vs sort vs scope + overflow (~820); micro-form / one row / sheet (~854); order target strip → grid → loupe (~896)]
- [Source: _bmad-output/planning-artifacts/initiative-fyne-image-first-phase1-party-2026-04-15.md — UX-DR16–DR19 measurement / async coherence (cross-check if layout or strip height claims change)]
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
| **Duplicate tag display names** | Same `Select` label collision as collections once Story 2.5 lists real tags. Mitigation: first matching option wins (documented on `buildFilters`); later enforce unique tag labels or disambiguate in UI. |
| **Literal English copy vs future i18n** | AC default strings and segment labels are **MVP English literals** (testable in CI). Full localization is **out of scope** until a dedicated epic; changing user-visible strings requires updating **`internal/app/review.go`** constants **and** this story / regression tests together. |

### Definition of Done (Story 2.2, create lens)

- Epics **FR-15**, **FR-16**, **UX-DR2** behaviors are **traceable** to AC + tasks; no hand-written second filter for count vs list/ID queries (`ReviewBrowseBaseWhere` + `ReviewFilterWhereSuffix` + shared list APIs).
- **`epics.md` §2.2** stays **non-contradictory** with the three-slot strip defaults (including **`any tag`** sentinel semantics).
- **Regression:** `go test ./...` green, including strip **label order**, **Tab/focus** order (dark + light), **theme focus contrast**, **suffix/base SQL contracts**, **default sentinel strings** pinned to AC copy, and **`TestReviewFilters_FR16DefaultMeansUnconstrained`** (DTO vocabulary for FR-16 “all any”).

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

### Party mode — session 1/2 (automated, `dev` hook; 2026-04-14 revisit)

Roundtable (simulated): **Amelia (Dev)** argued pairwise `ReviewFilterWhereSuffix` tests are insufficient now that **tag** is a first-class dimension—**collection+rating** and **collection+tag** do not prove a three-knob query keeps **`?` order** aligned with fragment concatenation. **Winston (Architect)** called that paranoid until someone hand-rewrites the suffix builder; **Amelia** shot back that one swapped `append(args, …)` ships silent wrong counts. **Sally (UX)** raised the same **duplicate-label** trap for **tags** as for albums—users cannot tell which row they picked if two tags share a label. **Mary (Analyst)** confirmed this stays a **data-quality** issue for MVP but must not pretend uniqueness exists in code.

**Resolution:** Add **`TestReviewFilterWhereSuffix_collectionMinRatingTagArgOrder`** (collection id → min rating → tag id). Document **first matching option wins** for duplicate **collection or tag** labels beside **`buildFilters`** in `review.go`.

### Party mode — session 2/2 (automated, `dev` hook; ListIDs vs grid parity deepen)

Roundtable (simulated): **Winston (Architect)** challenged the prior round’s narrow focus on `ReviewFilterWhereSuffix` fragments: **`ListAssetIDsForReview`** is the spine for **“Share filtered set”** and other multi-asset flows, and it can drift from the grid **without** any typo in the suffix—e.g. a future edit changes **`ORDER BY`** in one query but not the other, or a LIMIT-less ID enumerator accidentally reorders ties. **Amelia (Dev)** countered that duplicating the `ORDER BY` string is still fine for MVP if CI proves the three surfaces agree on real data. **Murat (Test Architect)** insisted on an **integration** test with **collection + min rating + tag** all active so cardinality and **sort order** stay locked across `CountAssetsForReview`, `ListAssetsForReview`, and `ListAssetIDsForReview`. **Sally (UX)** framed the failure mode as user-visible **distrust**: the strip shows **N** matches while a filtered package preview silently iterates a different ordering or length.

**Orchestrator synthesis:** Land **`TestListAssetIDsForReview_matchesCountAndListAssetsForReviewOrder`** in `internal/store/review_query_test.go` (two-row match, one row excluded by min-rating under the same album+tag scope; asserts `len(ids)==count` and row/`id` order parity including `capture_time_unix DESC`).

### Party mode — session 1/2 (automated, `dev` hook; 2026-04-15)

Roundtable (simulated): **Amelia (Dev):** Story 2.5 already surfaces tag-strip sync failures on the **count line** (non-modal), but the string was still generic while **collections** used **`libraryErrText`**—support and QA cannot distinguish read failures from a silent refresh bug. **Sally (UX):** Do not paste raw `sqlite` into the strip; **mapped** library-read copy is enough if the prefix says **tags** failed. **Winston (Architect):** Mirror the **collections** shape: `(tags unavailable — …)` after the numeric count; no new architecture. **Mary (Analyst):** Hand-edited or corrupted libraries can lose the tags tables; the count must stay **honest** and the user must not trust a stale tag dropdown silently.

**Orchestrator synthesis:** Retain the **`syncTagStrip` error value**; append **`libraryErrText`** in **`tags unavailable`** suffixes on Review; apply the same pattern on **Rejected** for parity; add **`TestReviewView_tagStripSyncFailure_showsActionableSuffix`**. Sprint audit comment only; story stays **done**.

### Party mode — session 2/2 (automated, `dev` hook; 2026-04-15 deepen)

Roundtable (simulated): **🎨 Sally (UX):** Session 1/2 fixed **tags** copy, but the **count-failure** line still appended **`listErr`** without the **collections unavailable** prefix—QA hears "two different products" depending on whether `Count*` succeeded. **🏗️ Winston (Architect):** Worse: on that path, **tags** were named **before** **collections**, which fights **FR-15** left-to-right mental order. **💻 Amelia (Dev):** **`Rejected`** only read **`ListCollections` once** at construct time; a DB that dies **after** open leaves **`listErr` stale** forever while tag sync and counts go red—**Review** already re-lists each refresh. **📋 John (PM):** Ship **one** degradation story: same prefixes, same ordering, Rejected = Review.

**Orchestrator synthesis:** Reorder **`qerr`** degradation fragments to **collections → tags**; use **`; collections unavailable — `** + **`libraryErrText`** for **`listErr`** (Review + Rejected). **`refreshRejectedData`:** reload **`ListCollections`** every pass with the same **sentinel / stale-id** behavior as Review’s collection strip. Tests: **`TestReviewView_closedDB_degradedSuffix_ordersCollectionsBeforeTags`**, **`TestRejectedView_closedDB_degradedSuffix_ordersCollectionsBeforeTags`**. Sprint audit comment; status **done**.

### Party mode — session 1/2 (automated, `create` hook; 2026-04-15)

Roundtable (simulated): **📋 John (PM):** The sprint file still said **ready-for-dev** while the story header says **done**—that will confuse stand-up and scope reports; pick one source of truth. **🎨 Sally (UX):** UX-DR15 is bigger than the strip; without saying so in AC, QA might file false bugs that grid/loupe tab order is "wrong" before 2.3/2.4. **🏗️ Winston (Architect):** NFR-01 minimum width is where filter UIs die—either spell "shorter copy or overflow, never a second nav row" in AC5 or we'll relitigate layout every story. **📊 Mary (Analyst):** Acceptance tests assume **fixed English strings**; if someone "helpfully" localizes the `Select` options without updating specs, we'll get green tests that lie about product copy.

**Orchestrator synthesis:** (1) Set **`2-2-filter-strip: done`** in `sprint-status.yaml` with an audit comment. (2) Tighten **AC4**/**AC5** elaborations for **UX-DR15 deferral** and **NFR-01** clipping strategy. (3) Add **i18n risk** + a short **DoD** block for traceability/regression expectations. (4) Add **`TestReviewFilterStrip_defaultSentinels_matchStory22`** beside existing label-order coverage; refresh Dev Notes to reflect **`ReviewFilterStripSegmentLabels`** as **implemented**, not optional.

### Party mode — session 2/2 (automated, `create` hook; 2026-04-15 deepen)

Roundtable (simulated): **📚 Paige (Tech Writer):** Session 1/2 hardened *this* file, but **`epics.md` §2.2** still read as if FR-16 stopped at **two** defaults—stakeholders skimming only the epic would treat the third-slot **"Any tag"** contract as scope creep. **📋 John (PM):** The epic should stay a **headline**, not a clone of the story, yet it must **not contradict** the PRD strip; pick the smallest epic edit that restores trust. **Resolution:** Add **`any tag`** to the epic's fresh-session bullet; keep SQL/predicate detail **here** only.

**🎨 Sally (UX):** If we do not **fence** **UX-DR16**, teams will argue filter work already "proved" combined chrome height—**Story 2.11** loses leverage. **🏗️ Winston (Architect):** Numeric budgets and **minimum thumb edge** belong to **2.11 / NFR-01** evidence, not 2.2 acceptance. **Resolution:** Record that split under **Dev Notes** in this pass.

**📚 Paige (Tech Writer):** AC3 "no silent mismatch" is still prose-heavy for implementers extending **FR-17**; spell the **DTO fields** once. **📋 John (PM):** Put it in **Dev Notes**, not new AC—AC count is already high. **Resolution:** DTO table + **`TestReviewFilters_FR16DefaultMeansUnconstrained`** so "FR-16 defaults" names the same triple-nil state in spec and Go.

**Orchestrator synthesis:** Land **epic §2.2** clause, **story Dev Notes** (authority / UX-DR16 fence / AC3 map), **`internal/domain/review_filter_test.go`** FR-16 default test, **`sprint-status.yaml`** audit comment—**`2-2-filter-strip` remains `done`**.

## Dev Agent Record

### Agent Model Used

Composer (Cursor AI coding agent)

### Debug Log References

### Completion Notes List

- **2026-04-15 (party `dev` 2/2):** **`Count*` error** lines use **`collections unavailable —`** for **`listErr`** and **FR-15** fragment order (collections before tags); **`refreshRejectedData`** reloads **`ListCollections`** each refresh (parity with Review); **`TestReviewView_closedDB_degradedSuffix_ordersCollectionsBeforeTags`**, **`TestRejectedView_closedDB_degradedSuffix_ordersCollectionsBeforeTags`**.
- **2026-04-15 (party `dev` 1/2):** Tag strip sync errors use **`libraryErrText`** on the Review count line (parallel to collections-unavailable); same pattern on **Rejected**; regression **`TestReviewView_tagStripSyncFailure_showsActionableSuffix`** (dropped `tags` / `asset_tags`, refresh via rating `Select`).
- **2026-04-15 (BMAD dev-story):** Re-ran full workflow on provided story path; all tasks already `[x]`; `go test ./...` and `go build .` green (no code changes). AC1–5 satisfied by existing `NewReviewView` strip (`ReviewFilterStripSegmentLabels`, FR-16 sentinels), shared `ReviewBrowseBaseWhere` + `ReviewFilterWhereSuffix` + count/list/ID parity tests. Status remains **done** per sprint `2-2-filter-strip: done`.
- **2026-04-15 (party `create` 2/2):** Epic §2.2 **`any tag`** default parity; Dev Notes **UX-DR16 fence** + **AC3 DTO table**; `TestReviewFilters_FR16DefaultMeansUnconstrained`; sprint audit comment only (status **done**).
- **2026-04-15 (party `create` 1/2):** Sprint **`2-2-filter-strip`** aligned to **done**; story AC4/AC5 + risks/DoD tightened; `TestReviewFilterStrip_defaultSentinels_matchStory22` pins FR-16 / strip default copy.
- **2026-04-14 (dev-story re-run):** `go test ./...` and `go build .` green; all acceptance criteria and tasks remain satisfied. **UX backlog (one row + overflow):** additional sort/scope/advanced filters after MVP should use a sheet, drawer, or “More filters” control—not a second full-width nav row (UX-DR2 / epics alignment).
- Review panel uses **`NewReviewView`** (`internal/app/review.go`): labeled strip **Collection → Minimum rating → Tags**, defaults **No assigned collection / Any rating / Any tag**, live **`CountAssetsForReview`** label, honest **Story 2.3** grid placeholder.
- **`ReviewFilterWhereSuffix`** centralizes filter SQL for **count vs future list** parity (session 2 architect constraint). **`dev` party session 1** adds **suffix unit tests** plus **degraded-DB** label behavior (count still runs; list failure surfaces as a suffix warning when count works).
- **`dev` party session 2/2:** **`ReviewBrowseBaseWhere`** pins the shared **rejected / soft-delete** predicate for Story 2.3; **`newReviewViewWithoutDB`** + **`TestNewReviewView_nilDB_honestLabel`** cover nil `*sql.DB` without dropping strip chrome.
- **AC4 keyboard/focus:** `internal/app/review_test.go` runs Fyne `test.FocusNext` over `NewReviewView` in **dark and light** `PhotoToolTheme`; asserts canvas focus visits the three `Select` widgets in layout order. `internal/app/theme_test.go` asserts `ColorNameFocus` ≠ `ColorNameInputBackground` per variant (focus ring vs strip control chrome).
- **2026-04-14 (party dev1/2 revisit):** `TestReviewFilterWhereSuffix_collectionMinRatingTagArgOrder` pins **three-knob** bind order; `review.go` comment documents **duplicate album/tag labels** → first `Select` match.
- **2026-04-14 (party dev2/2 deepen):** `TestListAssetIDsForReview_matchesCountAndListAssetsForReviewOrder` locks **filtered ID lists** to the same **count + sort** contract as the Review grid query (package / filtered-share path cannot silently diverge).

### File List

- `internal/app/theme_test.go` — focus vs input-background contrast (Story 2.2 AC4)
- `internal/app/review_test.go` — filter strip Tab/focus order tests (Story 2.2 AC4)
- `internal/store/migrations/003_review_filters.sql` — `rating` column + index
- `internal/store/migrate.go` — schema v3
- `internal/domain/review_filter.go`, `internal/domain/review_filter_test.go`
- `internal/store/review_query.go`, `internal/store/review_query_test.go` — `ReviewBrowseBaseWhere`, `ReviewFilterWhereSuffix`, `CountAssetsForReview`, `ListAssetsForReview`, `ListAssetIDsForReview` (+ triple-filter arg-order contract test + ListIDs/count/order parity test)
- `internal/store/collections.go` — `ListCollections`, `CollectionRow`
- `internal/store/store_test.go` — expect schema version 3
- `internal/app/review.go`, `internal/app/review_test.go` — Review UI + strip label order + default-sentinel string regression tests + tag-sync degraded count line
- `internal/app/rejected.go` — Rejected filter strip uses the same **tags unavailable** / `libraryErrText` pattern when `ListTags` fails; **`refreshRejectedData`** reloads **`ListCollections`** each pass (party dev 2/2 parity with Review)
- `internal/app/shell.go` — Review panel wired to `NewReviewView`

## Change Log

- **2026-04-15:** Party mode **`dev` session 2/2** — **`qerr`** degradation copy/order parity (collections prefix + strip order); Rejected **per-refresh `ListCollections`**; closed-DB ordering tests (Review + Rejected); sprint audit comment.
- **2026-04-15:** Party mode **`dev` session 1/2** — actionable **tags unavailable** copy on Review/Rejected count lines; `TestReviewView_tagStripSyncFailure_showsActionableSuffix`; sprint audit comment.
- **2026-04-15:** BMAD **dev-story** workflow — verification pass: `go test ./...`, `go build .` green; no implementation gaps vs AC/tasks; story/sprint status unchanged (**done**).
- **2026-04-15:** BMAD **create-story** workflow — merged epics.md Story 2.2 AC (including UX-DR2 **one row + overflow**); preserved **Status: done**; refreshed References; sprint tracking later aligned to **done** (party create 2026-04-15).
- **2026-04-15:** Party mode **`create` session 2/2** — epic AC **`any tag`** default; story Dev Notes (spec authority, UX-DR16 fence, AC3 mapping); `TestReviewFilters_FR16DefaultMeansUnconstrained`.
- **2026-04-15:** Party mode **`create` session 1/2** — AC4/AC5 UX-DR15 + NFR-01 elaborations; DoD + i18n risk; sprint **`2-2-filter-strip` → done**; `TestReviewFilterStrip_defaultSentinels_matchStory22`.
- **2026-04-14:** Party mode **`dev` session 2/2** (deepen) — `TestListAssetIDsForReview_matchesCountAndListAssetsForReviewOrder` guards filtered ID lists vs grid **count + sort** (`internal/store/review_query_test.go`); sprint note updated; story **done** unchanged.
- **2026-04-14:** BMAD dev-story workflow re-run for Story 2.2 — verified implementation vs AC/tasks; `go test ./...` and `go build .` green; sprint `2-2-filter-strip` remains **done**; UX-DR2 overflow pattern for future controls noted in Completion Notes.
- **2026-04-13:** Story 2.2 — completed remaining Keyboard/focus task: automated Tab-order tests for Review filter strip (dark/light theme) and theme test for focus ring vs input background; `go test ./...` green.
- **2026-04-13:** Party mode **`dev` session 1/2** — `ReviewFilterWhereSuffix` contract tests, closed-DB Review label test, and `refreshCount` behavior when `ListCollections` fails but counting remains valid.
- **2026-04-13:** Party mode **`dev` session 2/2** — `ReviewBrowseBaseWhere` + contract test; nil-DB Review path (`newReviewViewWithoutDB`) and `TestNewReviewView_nilDB_honestLabel`; story marked **done**.
- **2026-04-14:** Party mode **`dev` session 1/2** (revisit) — suffix **collection+rating+tag** arg-order test; duplicate-label note on `buildFilters`.

