# Story 2.5: Tags editing in bulk review

Status: done

<!-- Ultimate context engine analysis completed — comprehensive developer guide created. -->

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

<!-- Party mode (automated create, 2026-04-13): session 1 — decisions/risks/DoD; session 2 — stress-test + FK/NOCASE/UX/accessibility guardrails (no verbatim repeat). -->
<!-- Party mode (automated dev, 2026-04-13): session 1/2 — dev hook; headless roundtable → user-visible tag error feedback + stale-id comment in syncTagStrip. -->
<!-- Party mode (automated dev, 2026-04-13): session 2/2 — challenge session 1; chunk vs all-or-nothing, silent tag-strip failures, FK test gap. -->

## Story

As a **photographer**,  
I want **to add/edit tags on photos from bulk review**,  
So that **I can organize beyond stars**.

**Implements:** FR-07 (tags portion; ratings already covered in 2.3–2.4); FR-15 (tags filter becomes functional); extends UX-DR2 third segment.

## Acceptance Criteria

1. **Schema & persistence (epics / FR-07):** **Given** a tag model exists (**new migration** under `internal/store/migrations/`, version bump in `migrate.go`), **when** the user assigns, removes, or changes which tags are linked to an asset (including replacing one link with another via remove+add), **then** associations persist in SQLite and survive app restart. **`PRAGMA foreign_keys = ON` is already required** (`internal/store/open.go`); migration **must** declare `FOREIGN KEY (asset_id) REFERENCES assets(id)` and `FOREIGN KEY (tag_id) REFERENCES tags(id)` on `asset_tags` with **documented** `ON DELETE` behavior (see Session 2 guardrails). **Out of MVP:** global rename of a tag label that rewrites all rows—users can add a new label and remove the old one. Prefer **normalized** `tags` + `asset_tags` (architecture §3.3) over ad-hoc JSON unless a documented exception is needed for filtering correctness.
2. **Filter strip “Tags” (FR-15, Story 2.2 extension):** **Given** at least one tag exists in the library, **when** the user opens the **Tags** control on Review, **then** options include **“Any tag”** plus **one entry per distinct tag** (sensible sort, e.g. name ascending). **Given** **“Any tag”** is selected, **when** counts and grid refresh run, **then** the tag dimension imposes **no extra constraint**—behavior matches Story **2.2** placeholder semantics (“no-op” on the predicate). **Given** a **specific tag** is selected, **when** filters apply, **then** only assets **linked** to that tag appear (consistent `CountAssetsForReview` / `ListAssetsForReview` / `ReviewFilterWhereSuffix`).
3. **Editing in bulk review:** **Given** one or more assets in Review, **when** the user edits tags, **then** changes apply to **each selected asset** as specified by the UI (add/remove or replace—pick one **documented** bulk semantics in code comments and keep UX copy honest). **Minimum bar:** tag editing works from **loupe** for the current asset **and** from the **grid** with **multi-selection** (modifier-click or equivalent Fyne pattern) so a **batch** of assets can receive the same tag operation without opening loupe per file. **Session 2:** if modifier-only selection is awkward on a target OS, provide a **documented** alternate (e.g. explicit “select mode” toggle or secondary tap target) rather than silently dropping batch tagging—**keyboard-driven multi-select is stretch**, but single-key focus/activation for tag controls must remain coherent with UX-DR2/DR15.
4. **No silent mismatch (inherits Story 2.2 / 2.4):** **Given** tag filter or tag data changes, **when** the review query refreshes, **then** grid count, rows, and tag filter options stay aligned (recount + list refresh on UI thread; new tags appear in strip without restart).
5. **Loupe / filter interaction:** **Given** Story **2.4** behavior, **when** the user changes **Tags** (or collection / min rating) in the strip while loupe is open, **then** loupe **dismisses** or **resyncs** per existing `refreshAll` rule—no stale index against a new result set.
6. **Tag identity in filters:** **Given** the filter strip, **when** a specific tag is chosen, **then** the predicate uses the tag’s **stable row id** (`tag_id`), not the display string—so rename/refactor of label copy cannot desync count vs grid without a DB update.

## Tasks / Subtasks

- [x] **Migration + schema** (AC: 1)
  - [x] Add `004_*.sql`: `tags` (stable id, unique tag label per product rules), `asset_tags` junction (`asset_id`, `tag_id`, PK/unique pair); indexes supporting `EXISTS` filter and `ListTags`.
  - [x] Encode **FK + ON DELETE** in SQL (see “Referential integrity” under Session 2); verify with `PRAGMA foreign_key_check` in at least one integration test after migrate.
  - [x] Bump `targetSchemaVersion` and wire `migrate.go` switch for v4.
- [x] **Domain + store** (AC: 1, 2, 4)
  - [x] Extend `domain.ReviewFilters` with optional tag constraint **`TagID *int64`** (**nil** = “Any tag” per strip default). Update `Validate()` and `internal/domain/review_filter_test.go` (**reject** non-nil `TagID <= 0`; **do not** require row existence in domain—store/query treats unknown id as **empty result set**, not a hard error, unless product later adds a strict mode).
  - [x] Extend `ReviewFilterWhereSuffix` + `CountAssetsForReview` / `ListAssetsForReview` to apply tag `EXISTS` when set; add `review_query_test.go` cases: any-tag vs specific tag, empty library, tag with zero assets, **stale TagID** (id not in `tags`) yields **0 count / empty list** matching grid.
  - [x] Store APIs: `ListTags`, upsert/find tag by name, `ReplaceAssetTags(assetID, []tagID)` or add/remove helpers; batch variant for multi-asset updates inside **one transaction** where appropriate.
- [x] **Filter strip wiring** (AC: 2, 4)
  - [x] Replace static `tagOpts := []string{reviewTagAny}` in `internal/app/review.go` with options from `ListTags`; preserve **“Any tag”** as default matching FR-16 / Story 2.2.
  - [x] Map `tagsSel` selection → `ReviewFilters`; include in `buildFilters()` and `refreshAll()`.
- [x] **Bulk review UI** (AC: 3, 4, 5)
  - [x] Grid: multi-select state + visual affordance (selected border/overlay); ensure tap-to-open-loupe still works without breaking selection semantics. **MVP:** modifier (Cmd/Ctrl)+click toggle; plain click clears other selection unless extending per platform norms—document in code comment. **Stretch (session 2 / fast-follow):** Shift+click range selection if Fyne gesture support is clean.
  - [x] Tag editor: e.g. `SelectEntry` for typeahead new/existing tags + list/chip of tags on selection; commands **Add** / **Remove** applying to **all** selected IDs per **Bulk tag semantics** below (no bulk “replace entire tag set” in MVP).
  - [x] Loupe: show/edit tags for current asset; persist on change; reuse same store helpers as grid.
  - [x] On tag write: invalidate affected thumbs if badges show tags later; at minimum refresh grid page / count.
- [x] **Tests** (AC: 1–4)
  - [x] Store integration tests for migrations, tag CRUD, filter queries, bulk update.
  - [x] Domain tests for filter validation with tag set/unset.
  - [x] Optional: lightweight app tests if headless Fyne allows; otherwise document manual QA (multi-select + filter + loupe).

## Dev Notes

### Product decisions (Party mode — create, session 1)

Baseline; Session 2 **adds guardrails** below—only override session 1 if a conflict is explicitly called out.

| Topic | Decision |
|--------|-----------|
| **Bulk tag semantics (AC3)** | **Add:** for each selected `asset_id`, link tag if not already linked (idempotent). **Remove:** unlink tag from each selected asset. **Not in MVP:** bulk operation that replaces the full tag set on each asset in one gesture (users can Remove then Add). |
| **Tag label normalization** | **SQLite:** `tags.label` is human text; enforce uniqueness with **`UNIQUE(label COLLATE NOCASE)`** (or store a normalized key column—pick one approach and use it in upsert). **Always** `strings.TrimSpace`; collapse internal ASCII whitespace runs to a single space before insert/lookup. **Empty** label rejected in UI + `Validate`. |
| **Filter list contents (AC2)** | `ListTags` returns **all** tag rows sorted by `label` ascending (case-insensitive), plus sentinel **Any tag** in UI only. Tags with **zero** assets remain visible in the strip (predictable library-wide vocabulary). Grid may be empty when such a tag is selected. |
| **Predicate shape (AC2, AC6)** | `ReviewFilters` carries optional **`TagID *int64`**; `nil` = Any tag. SQL uses **`EXISTS (SELECT 1 FROM asset_tags …)`** only in `ReviewFilterWhereSuffix`—never match on free-text tag name in the query layer. |

### Session 2 stress test (Party mode — create)

**Challenges to session 1 (resolved for dev):**

| Topic | Tension | Resolution |
|--------|---------|------------|
| **NOCASE uniqueness** | Architect: “`NOCASE` only folds ASCII; locale-sensitive collisions (e.g. Turkish dotted/dotless I) can slip through.” | Keep `NOCASE` for MVP **but** document in migration or store package doc: labels are **SQLite NOCASE-equivalent**, not full Unicode case-folding; duplicates that differ only by non-ASCII case are **out of scope** for MVP. |
| **Orphan / empty tags in strip** | UX: “Empty tags clutter the strip.” PM: “Vocabulary stability beats hiding.” | **Keep session 1** (show zero-asset tags). Add **UX copy** hint when a selected tag matches nothing (“No photos with this tag”) to reduce confusion. **Fast-follow:** optional “hide unused tags” setting—**not** in this story. |
| **FK `ON DELETE`** | Dev: “Soft-deleted assets still have `id`; junction rows linger.” Architect: “Hard delete or tag delete must not strand rows.” | **`asset_tags.asset_id` → `assets(id)` ON DELETE CASCADE:** removing an asset row removes links (aligns with future hard-delete/quarantine finalization). **`asset_tags.tag_id` → `tags(id)` ON DELETE RESTRICT** while MVP has **no** tag-delete API (prevents accidental wipe); junction-only deletes via **unlink** remain the supported path. Revisit when tag deletion ships. |
| **Stale filter TagID** | Dev: “UI could hold a tag id removed by another process.” | Single-writer desktop MVP: still handle **missing tag id** in SQL as **no rows** (EXISTS never matches). After `ListTags` refresh, if current selection id missing, **reset strip to “Any tag”** and run `refreshAll` (document in `review.go`). |
| **Grid loupe vs selection** | UX: “Modifier-click is invisible to many users.” | **Session 1** pattern stands; add **visible** selection chrome + one-line **in-app help** or status hint (e.g. “Cmd/Ctrl+click to select multiple”) in review chrome **if** space allows—otherwise document in manual QA until a help surface exists. |

**Additional risks (session 2):**

| Risk | Mitigation |
|------|------------|
| Bulk transaction size (1000+ selection) | Cap UI batch or chunk writes with **one transaction per chunk** and a **single** user-visible error if any chunk fails; log chunk index. |
| Tag upsert race (double create same label) | Use a **single** `INSERT … ON CONFLICT` / transaction-wrapped upsert path; surface **one** user-visible error on constraint failure after retry. |
| Filter strip selection resets during refresh | Preserve selection by **tag id** when rebuilding `Select` options; only fall back to “Any tag” when id absent (see above). |

### Technical requirements

- **Single predicate source:** Reuse `ReviewBrowseBaseWhere` + `ReviewFilterWhereSuffix` only—**no** second copy of tag SQL in `internal/app` (Stories 2.2–2.4 precedent).
- **Transactions:** Bulk link/unlink use **one SQLite transaction per chunk** (`tagBulkChunk`, default 500) so a failure after earlier chunks can leave **partial** updates; errors after chunk 0 include how many leading assets were already updated. True single-transaction all-or-nothing for 1000+ rows is deferred (SQLite practical limits / lock duration).
- **Tag identity:** Normalization rules in table above; single upsert path (`FindOrCreateTagByLabel`) used by GUI and future CLI.
- **Performance:** Tag filter uses indexed junction lookup; avoid loading all assets into memory.
- **Logging:** `log/slog` on unexpected DB errors; user-visible strings stay free of raw SQL.

### Risks (create review)

| Risk | Mitigation |
|------|------------|
| Fyne multi-select + tap-to-open loupe fight for the same pointer events | Keep a dedicated tap layer or explicit “open” affordance; test macOS/Windows; document chosen pattern beside grid code. |
| `Select` options refreshed while user has dropdown open | Debounce or close dropdown on `refreshAll`; ensure UI-thread refresh only. |
| Count/list drift after tag write | Reuse same `refreshAll` path as ratings; add store test for count=list page under tag filter after bulk update. |
| Unicode / combining characters in labels | Trimming is ASCII-space-centric; document that labels are user-supplied UTF-8; rely on SQLite UTF-8 storage; no extra normalization MVP. |

### Definition of done (story)

- [x] Migration `004` applied; `targetSchemaVersion` = 4; fresh library + upgrade from v3 both tested.
- [x] Domain `ReviewFilters` + tests include tag unset/set; **`TagID` > 0 when set**; stale id behavior covered at **store/query** layer.
- [x] Store: `ListTags`, junction writes, `ReviewFilterWhereSuffix` EXISTS clause, integration tests for any vs specific tag, **foreign_key_check** after migrate, **stale TagID** count/list parity.
- [x] Review UI: strip populated from DB; bulk add/remove from grid selection + loupe; `refreshAll` on tag filter change preserves AC5; **strip survives tag list refresh** without losing a valid selected id.
- [x] `go test ./...` green; manual note only if headless cannot cover a Fyne gesture.

### Architecture compliance

- **§3.3:** Normalized `tags` / `asset_tags` aligned with FR-15 filtering.
- **§3.8 / §5.2:** Fyne in `internal/app`, SQL only in `internal/store`; migrations embedded like existing `001`–`003`.

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3** (`go.mod`): compose standard widgets; multi-select may need careful `List`/`Tap` handling—follow patterns from `review_grid.go` / `tapLayer`.

### File structure requirements

- **Add:** `internal/store/migrations/004_*.sql`
- **Extend:** `internal/store/migrate.go`, `internal/store/review_query.go`, possibly `internal/store/assets.go` or new `tags.go`
- **Extend:** `internal/domain/review_filter.go` (+ `review_filter_test.go`)
- **Extend:** `internal/app/review.go` (strip + `buildFilters`), `internal/app/review_grid.go` (selection + tag UI hooks), `internal/app/review_loupe.go` (tag editor)

### Testing requirements

- **Mandatory:** SQLite temp-DB tests for migration application, tag persistence, and **Count**/**List** parity under tag filters.
- **Regression:** Existing review filter tests (collection, rating, rejected/deleted exclusions) remain green.

### Previous story intelligence (Story 2.4)

- **Tags placeholder:** `ReviewFilters` currently has **no** tag field; `tagsSel` is **“Any tag”** only—this story **must** extend the DTO and SQL, not bolt on parallel predicates.
- **Loupe dismiss:** `refreshAll` already clears loupe when filters change; extending `buildFilters()` automatically inherits AC5 if tag changes call `refreshAll`.
- **Grid/list parity:** Loupe stepping uses the same `ListAssetsForReview` + `ReviewFilters` as the grid—tag filter must apply to **both**.

### Git intelligence summary

- Treat `internal/app/review*.go`, `internal/store/review_query.go`, and Story **2.2** / **2.4** artifacts as the authoritative interaction patterns.

### Latest technical information

- SQLite **partial indexes** optional if tag cardinality huge; MVP can start with straightforward indexes on `asset_tags.tag_id` and `asset_tags.asset_id`.

### Project structure notes

- No `project-context.md` in repo; planning artifacts + this file define the contract.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 2, Story 2.5, FR coverage FR-07, FR-15]
- [Source: _bmad-output/planning-artifacts/epics.md — Requirements inventory FR-07, FR-15, UX-DR2]
- [Source: _bmad-output/planning-artifacts/PRD.md — Bulk review tags, filter order]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.3 tags/asset_tags, §3.8 UI, §4.5 default browse, §5.2 boundaries]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — Filter strip order and mental model]
- [Source: _bmad-output/planning-artifacts/implementation-readiness-report.md — Tags schema timing / 2.5]
- [Source: _bmad-output/implementation-artifacts/2-2-filter-strip.md — “Any tag” placeholder semantics]
- [Source: _bmad-output/implementation-artifacts/2-4-review-loupe-keyboard-rating.md — loupe dismiss on filter refresh, shared query contract]
- [Source: internal/app/review.go — `buildFilters`, `tagsSel`, `reviewTagAny`]
- [Source: internal/domain/review_filter.go — `ReviewFilters` baseline]
- [Source: internal/store/review_query.go — `ReviewFilterWhereSuffix`, list/count]
- [Source: internal/store/migrate.go — migration versioning]

## Dev Agent Record

### Agent Model Used

Cursor agent (Claude) — BMAD dev-story workflow + party mode (automated dev, sessions 1–2/2), 2026-04-13.

### Party mode — dev round (headless, simulated voices)

#### Session 1/2

- **Amelia (dev):** Bulk/loupe tag failures only hit `slog`; users get silent no-ops on DB errors and on “remove unknown label.” Ship `dialog.ShowError` / `ShowInformation` on those paths; guard loupe tag actions when `currentID` is invalid.
- **Sally (UX):** Agree — silent failures read as “broken.” Prefer short factual dialogs over expanding the count label; keep the existing “No photos with this tag” on zero-result filter.
- **Winston (architect):** Push back lightly — don’t modal-spam on hot paths; these are explicit button taps, so one dialog per failed action is fine. Document stale `TagID` reset beside `syncTagStrip` so the next maintainer doesn’t “fix” it away.
- **Murat (test):** Dialogs are hard to assert headless; keep store/query coverage as the quality gate; note manual check for dialog copy on add/remove errors.

**Orchestrator synthesis:** Implement Fyne `dialog` feedback for bulk Review and loupe tag add/remove; add `syncTagStrip` stale-id comment; re-run `go test ./...`; mark story done in sprint file.

#### Session 2/2 (stress-test; do not repeat session 1 verbatim)

- **Amelia (dev):** Session 1 fixed button-path dialogs, but **`syncTagStrip` errors are still log-only** while `CountAssetsForReview` runs — users see a plausible count with a **stale tag dropdown**. Surface strip sync failure in the **count line** (non-modal), not a dialog on every refresh.
- **Winston (architect):** The story text said **all-or-nothing** per bulk action; **`LinkTagToAssets` already chunks**. That is honest **partial durability** if chunk *k+1* fails. Either document the boundary or wrap errors so the dialog is not lying about what happened.
- **Sally (UX):** If partial apply happens, the error string must say **“first N photos updated”** in plain language — not “chunk index.” Users will blame the app if half the selection changed with a generic “database error.”
- **Murat (test):** Story AC demanded **`foreign_key_check` after migrate** — we had helpers but weak coverage on **fresh open**. Add **`ForeignKeyCheck` after `Open`** test; add a **forced small chunk** test that proves the partial-apply error shape.

**Orchestrator synthesis:** Extend `LinkTagToAssets` / `UnlinkTagFromAssets` errors when `chunkStart > 0`; document chunk semantics in `tags.go` and story **Technical requirements**; append **tag strip sync** failure to the count label; add `TestForeignKeyCheck_afterMigrate` + `TestLinkTagToAssets_partialChunkErrorMessage` (`tagBulkChunk` as tunable var for tests only).

### Debug Log References

- Fyne `PointEvent` has no modifiers in v2.7.3; `tapLayer` implements `desktop.Mouseable` (`MouseUp`) for Cmd/Ctrl+click bulk select without double-firing `Tapped`.
- `Select.OnChanged` fired during programmatic `SetSelected` in `syncTagStrip`; added `suspendTagSelectRefresh` guard to stop `refreshAll` recursion.

### Completion Notes List

- Implemented migration `004_tags.sql` (tags + asset_tags, FK CASCADE/RESTRICT, NOCASE uniqueness), schema v4, `ReviewFilters.TagID`, `ReviewFilterWhereSuffix` EXISTS, store tag APIs with chunked transactions, Review strip + bulk tag bar + loupe tag row; grid multi-select with selection fill and plain-click opens loupe.
- Remove operations use `FindTagByLabel` only (no accidental tag row creation). `go test ./...` and `go build .` pass.
- Party dev session 1: user-visible dialogs for tag create/link/unlink errors and for remove-with-unknown-label; loupe guards empty `currentID`.
- Party dev session 2: partial-chunk error messages; count-line hint when tag strip sync fails; `ForeignKeyCheck` + chunk-failure store tests; `tagBulkChunk` variable for test control.

### File List

- `internal/store/migrations/004_tags.sql`
- `internal/store/migrate.go`
- `internal/store/tags.go` (incl. session 2: chunk partial errors, `tagBulkChunk` var)
- `internal/store/tags_test.go` (incl. session 2: `ForeignKeyCheck`, partial chunk test)
- `internal/store/review_query.go`
- `internal/store/review_query_test.go`
- `internal/store/store_test.go`
- `internal/domain/review_filter.go`
- `internal/domain/review_filter_test.go`
- `internal/app/review.go` (incl. session 2: tag strip sync hint on count label)
- `internal/app/review_grid.go`
- `internal/app/review_loupe.go`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

### Change Log

- **2026-04-13:** Story 2.5 — tags schema, filter + bulk review UI, tests; sprint status `2-5-tags-bulk-review` → `review`.
- **2026-04-13:** Party mode (dev) session 1/2 — tag error dialogs (Review + loupe), `syncTagStrip` stale-id comment; sprint `2-5-tags-bulk-review` → `done`.
- **2026-04-13:** Party mode (dev) session 2/2 — bulk tag chunk partial-apply errors; tag strip sync failure in count label; FK check + chunk tests; story technical note aligned with chunking.
