# Story 2.12: Empty states and error tone

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- create-story workflow (2026-04-13): Epic 2 Story 2.12 — UX-DR9 primary CTAs on empty surfaces + proportionate error copy; brownfield `review.go`, `rejected.go`, `collections.go`, `shell.go`, `collection_store_err_text.go`. -->

## Story

As a **photographer**,  
I want **helpful empty states and honest errors**,  
So that **I never feel lost after import or filtering**.

**Implements:** UX-DR9; cross-cutting UX quality (epics).

## Acceptance Criteria

1. **UX-DR9 — Review, empty library:** **Given** the **Review** surface with **default** filter strip values (**Collection** = “No assigned collection”, **Minimum rating** = “Any rating”, **Tags** = “Any tag”) per FR-15/FR-16, **when** **zero** assets match the default browse predicate (`CountAssetsForReview` with those defaults), **then** the user sees a **clear empty state** (short explanation + **one primary CTA**, e.g. **HighImportance** button) whose **obvious next step** is to **add photos** (e.g. switch primary nav to **Upload** via existing shell wiring — reuse the same navigation pattern as `NewRejectedView` → `onGotoReview` in `shell.go`). **And** the layout does not rely on only the count line (`Matching assets: 0`) as the empty experience.
2. **UX-DR9 — Review, no filter results (non-empty library):** **Given** at least **one** asset matches **default** Review filters, **when** the **current** filter combination yields **zero** matches, **then** the user sees an empty state with **one primary CTA** that **recovers** the situation (e.g. **“Reset filters”** restoring the strip to defaults, or a single action that clearly undoes the narrowing predicate — pick one primary, no competing equal-weight buttons). **And** copy explains that **filters** hid everything (not that the library is empty).
3. **UX-DR9 — Rejected / Hidden:** **Given** the **Rejected** surface, **when** there are **no** hidden rows in the **current** filter scope, **then** the user sees **one primary CTA** consistent with UX-DR9. **Differentiate** (a) **no rejected assets at all** under default filters vs (b) **filters exclude** all hidden items — message + CTA should match the case (today `rejected.go` uses one line for all `n == 0`; split or clarify so users are not misled).
4. **Collections surfaces (cross-cutting):** **Given** **Collections** list with **zero** albums (`collections.go` already shows copy + **New album**), **when** viewed, **then** **one** control reads as the **primary** action (importance / placement) and secondary actions do not visually compete at the same weight. **Given** an album **detail** with **zero** visible assets (`replaceDetailBody` early return), **when** shown, **then** add **one primary CTA** (e.g. **Back to albums** and/or **Go to Review**) so the user is not left with label-only dead-ends.
5. **Proportionate honesty — IO/DB errors:** **Given** user-visible failures from **store** or **filesystem** operations (count/list/load paths in Review, Rejected, Collections, upload ingest), **when** an error is shown (**`dialog.ShowError` / `ShowInformation` / inline label**), **then** copy is **factual** (what failed, at a high level) and includes a **concrete next step** (retry, check library path, reset filters, go to Upload, etc.). **Avoid** cheerful or dismissive empty/error tone (UX spec anti-pattern: “cheerful empty errors”). **Do not** surface raw SQLite/driver strings to users where a mapped message exists — extend **`collectionStoreErrText`** or a **small shared helper** only if needed; keep SQL in **`internal/store`**.
6. **Themes & layout:** **Given** new empty-state blocks, **when** rendered in **dark** and **light** themes (**UX-DR1**), **then** text remains readable (semantic theme colors / labels, no ad-hoc grays). **And** empty-state chrome stays consistent with **NFR-01** safe regions (no mandatory horizontal scroll of the **shell** to reach the primary CTA at **1024×768** — align with Story **2.11** evidence mindset).
7. **Regression / tests:** **Given** headless tests in **`internal/app`**, **when** they run in CI, **then** add or extend tests that pin **presence** of empty-state CTAs or **key strings** for the three UX-DR9 Review/Rejected cases (and Collections detail empty if practical), without requiring interactive window focus. **And** existing story tests (**e.g.** `TestCollectionsView_zeroAlbums_emptyStateHidesList`) remain green or are updated if copy/structure changes.

## Party mode — session 1/2 (automated headless, hook **create**, 2026-04-13)

**Note:** `_bmad/_config/agent-manifest.csv` was not present; round was **simulated** with separated voices per `bmad-party-mode` skill.

**Winston (Architect):** Insisted on one shell-level navigation primitive (`gotoUpload` mirroring `gotoReview`) so Review does not reach into nav buttons, and on reusing **`reviewFiltersAtFR16Defaults`** (nil `ReviewFilters` semantics) so "library empty" cannot drift from FR-16.

**Sally (UX Designer):** Pushed for **one High-importance control** per empty surface; challenged putting **Reset filters** on Rejected, and accepted it once copy clearly separates "filters hid everything" from "nothing is hidden yet."

**Mary (Analyst):** Required measurable differentiation: default-filter zero vs narrowed-filter zero; album detail must not end on label-only dead-ends.

**Murat (Test Architect):** Demanded **callback firing** tests (atomic counters), **Scroll-descent** in widget walks for buttons inside `container.Scroll`, and pinned strings for the three Review/Rejected cases.

**Orchestrator synthesis (applied):** Implemented `onGotoUpload` + `gotoUpload` in `shell.go`; Review empty block + **Reset filters** with `suspendColSelectRefresh` / `suspendMinRatingSelectRefresh` / `suspendTagSelectRefresh`; Rejected split **Back to Review** vs **Reset filters**; Collections **New album** emphasis when `len(collRows)==0` with live `newAlbumBtn` refresh; detail empty **Back to albums** + **Go to Review**; **`libraryErrText`** for count/read lines; headless tests `TestStory212_*`; `collectButtons` walks **Scroll** content.

**Session 2/2 (planned challenge):** Audit remaining **`dialog.ShowError` / ingest** paths for raw driver text; consider a single `onNavigate(key)` if callback arity grows; manual NFR-01 spot-check for new empty blocks at 1024×768.

## Party mode — session 2/2 (automated headless, hook **create**, 2026-04-14)

**Note:** `_bmad/_config/agent-manifest.csv` still absent; round **simulated** (skill `--solo` equivalent). Session **2/2** challenges session 1’s “AC5 mostly done” claim—focus **error-dialog sweep**, not re-litigating empty-state layout.

**John (PM):** Session 1 closed UX-DR9 surfaces but left **`dialog.ShowError(err, …)`** in Review grid/loupe, Rejected, Collections delete, and Upload post-import—that violates AC5’s “no raw driver text” for common DB/FS failures. That’s a **spec execution gap**, not a nice-to-have polish pass.

**Winston (Architect):** Centralizing copy in **`userFacingDialogErrText`** is the right brownfield move, but **string sniffing** on `err.Error()` is brittle long term. Ship the helper **now** with tests that freeze behavior; introduce typed **`store.Err…`** values only after we see a real substring misfire.

**Sally (UX Designer):** The generic write-fallback is acceptable **if** we never surface `fmt.Errorf("create collection: %w", err)` to users—that reads like a log line. Upload file-picker failures need their **own** line (“open the selected file”), not “update the library.”

**Murat (Test Architect):** A single mapper function needs a **table-driven unit test**—locked DB, FK, quarantine rename, default fallback. GUI tests already pin empty states; do not duplicate that coverage here.

**Orchestrator synthesis (applied):** Added **`userFacingDialogErrText`** and **`userFacingFileOpenErrText`** in `collection_store_err_text.go`; replaced every **`dialog.ShowError` that passed a raw `err`** across `review.go`, `review_loupe.go`, `rejected.go`, `collections.go`, and `upload.go` with mapped copy. Added **`collection_store_err_text_test.go`**. **Deferred:** `onNavigate(key)` refactor (callback count still manageable); NFR-01 1024×768 spot-check remains manual per Story 2.11.

## Party mode — dev session 1/2 (automated headless, hook **dev**, 2026-04-14)

**Note:** `_bmad/_config/agent-manifest.csv` absent; round **simulated** (skill `--solo` equivalent). Session **1/2** challenges create-session closure: **album write surfaces** still routed some failures through **`collectionStoreErrText`**, whose default branch is **`return err.Error()`** — sufficient for FK/not-found mapping but not for **UNIQUE / generic SQLITE** strings on assign and inline forms.

**Amelia (Dev):** Insists this is an AC5 gap distinct from the earlier dialog sweep: loupe album toggles, quick assign links, and Collections rename/create popups could still surface **`UNIQUE constraint failed`** verbatim. Fix should be **narrow** (write UI only) so **`libraryErrText`** read semantics stay intact.

**Murat (Test Architect):** Pushes back on folding SQLite sniffing into **`collectionStoreErrText`** — **`libraryErrText`** wraps it; misclassified **`no such table`** would read as an “update” failure. Demands a **dedicated helper** with **table-driven tests** (validation pass-through vs opaque SQL).

**Sally (UX Designer):** Fine with a generic **“Could not update the library…”** fallback for opaque DB errors on album flows; **“create collection: name is required”** must remain literal so the user knows what to correct.

**Winston (Architect):** Agrees: keep **`collectionStoreErrText`** as the FK/not-found/validation primitive; add **`userFacingCollectionWriteErrText`** that composes it, then scrubs SQL-shaped tails before delegating to **`userFacingDialogErrText`**.

**Orchestrator synthesis (applied):** Implemented **`userFacingCollectionWriteErrText`** + **`TestUserFacingCollectionWriteErrText`**; replaced **`collectionStoreErrText`** on Review assign dialogs, loupe collection toggle + new-album form, and Collections album form inline errors. Added **`dropRejectReason`** for drag/drop URI edge cases + **`TestDropRejectReason`**.

## Party mode — dev session 2/2 (automated headless, hook **dev**, 2026-04-14)

**Note:** `_bmad/_config/agent-manifest.csv` still absent; round **simulated** (skill `--solo` equivalent). Session **2/2** challenges dev 1/2’s focus on **`ShowError`** — AC5 also covers **`ShowInformation`** and “concrete next step” on **receipt-style** copy, not only DB mapping.

**Amelia (Dev):** Dev 1/2 left **`dialog.ShowInformation("Some items were skipped", strings.Join(...))`** unbounded — huge drops bury the primary message and feel like a log dump, not UX-DR9 tone. **`not a local file (https)`** in **`dropRejectReason`’s** default branch is still a semi-technical fragment without a recovery path.

**Murat (Test Architect):** Cap behavior needs a **deterministic unit test** (exact threshold and tail string). **`summarizeDoneMessage`** must assert the **failed > 0** next-step sentence so a future refactor cannot drop AC5 on the import-complete dialog.

**Sally (UX Designer):** Information dialogs should stay **scannable**; an “and N more” tail with **Add images…** is better than truncating silently. Non-file drops should sound like **user action** (“save or export”), not protocol names.

**Winston (Architect):** Keep **`droppedSkipSummaryForDialog`** next to **`classifyDroppedURIs`** in **`drop_paths.go`** — single place for drop UX strings; **`upload.go`** only composes receipts.

**Orchestrator synthesis (applied):** **`droppedSkipSummaryForDialog`** (cap 8 + tail); **`dropRejectReason`** maps non-`file` scheme errors to guided copy; **`summarizeDoneMessage`** appends a **failed-count recovery** line; tests **`TestDroppedSkipSummaryForDialog`**, **`TestSummarizeDoneMessage_failedIncludesNextStep`**, updated **`TestDropRejectReason`**. **Still deferred:** typed **`store.Err…`** until substring misfire; **`onNavigate(key)`** refactor.

## Tasks / Subtasks

- [x] **Review empty states** (AC: 1, 2, 5–7)
  - [x] Add a dedicated empty-state region in **`internal/app/review.go`** (or extracted helper) that switches between **library-empty** vs **filters-empty** using `CountAssetsForReview(db, domain.ReviewFilters{})` vs current `buildFilters()` result (same predicates as today).
  - [x] Wire **primary CTA**: library-empty → navigate to **Upload** (mirror `gotoReview` pattern from `shell.go` — may require a new callback from `NewReviewView` like `onGotoUpload func()` injected from `newMainShell`).
  - [x] Filters-empty → implement **Reset filters** (programmatically set strip to defaults; trigger existing refresh path; ensure tag/collection sync paths stay consistent with `suspendTagSelectRefresh` / `suspendColSelectRefresh` patterns).
  - [x] Audit Review error strings (`refreshReviewData` failure branches, assign flows) for **next-step** clarity (AC5) — *dialogs use `userFacingDialogErrText` / `collectionStoreErrText`; inline count/query uses `libraryErrText`.*
- [x] **Rejected empty states** (AC: 3, 5–7)
  - [x] In **`internal/app/rejected.go`**, detect **global** no-hidden vs **filter** no-match (reuse same default filter semantics as Review where applicable).
  - [x] Adjust copy + single primary CTA per case; keep **Back to Review** as the primary where appropriate.
  - [x] Audit rejected error paths (`restore`, bulk delete, tag strip sync) for proportionate copy — *restore/delete dialogs mapped via `userFacingDialogErrText`.*
- [x] **Collections empty / error polish** (AC: 4, 5–7)
  - [x] **`internal/app/collections.go`:** primary vs secondary button emphasis for zero-album list; album detail `n == 0` branch — add CTA row (navigate to Review and/or pop stack to list).
  - [x] Align list load failure copy (`Can't load albums — library read failed.`) with AC5 **next step** if needed.
- [x] **Shared patterns** (AC: 5–6)
  - [x] Optionally factor a tiny **`userFacingErrText(err error) string`** or expand **`collection_store_err_text.go`** only where the same DB errors appear in multiple surfaces; avoid duplicating paragraph-long strings.
  - [x] Verify new widgets use **theme**-aware labels and **Importance** on buttons per Fyne patterns.
- [x] **Tests** (AC: 7)
  - [x] Extend **`review_test.go`** / **`rejected`** / **`collections_test.go`** with headless assertions on empty-state visibility and/or CTA callbacks (inject test doubles for nav callbacks if needed).
  - [x] Unit tests for **`userFacingDialogErrText`** / **`userFacingFileOpenErrText`** in **`collection_store_err_text_test.go`** (session 2/2).
  - [x] Dev session 2/2: capped **`ShowInformation`** skip summaries + import receipt next step when **`Failed > 0`**; non-file drop copy (**`drop_paths`**).

## Dev Notes

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **UX-DR9** | **One primary CTA** per empty surface: library empty, no results, empty Rejected ([Source: `_bmad-output/planning-artifacts/epics.md` — UX-DR9, Story 2.12]). |
| **Error tone** | **Proportionate honesty**: factual + **next step**; not “cheerful” dismissals ([Source: `_bmad-output/planning-artifacts/ux-design-specification.md` — §Anti-Patterns, §2.3 Success Criteria “Honesty”, §2.5 Feedback]). |
| **Layering** | Fyne UI in **`internal/app`**; SQL and predicates in **`internal/store`**; no new business rules in widgets beyond orchestration ([Source: `_bmad-output/planning-artifacts/architecture.md` — §5.x layering]). |
| **Navigation** | Primary nav order **Upload → Review → Collections → Rejected** (**UX-DR13**); CTAs should use the same shell selection model as existing buttons in **`shell.go`**. |

### Existing implementation to extend

- **Review** count + filters + grid: **`internal/app/review.go`**, **`internal/app/review_grid.go`**. Default filter constants: `reviewCollectionSentinel`, `reviewRatingAny`, `reviewTagAny`.
- **Rejected** mirror filters + empty row: **`internal/app/rejected.go`** (`emptyHint`, `backToReview`, `refreshRejectedData`).
- **Collections** list empty + detail empty: **`internal/app/collections.go`** (`reloadCollectionRows`, `replaceDetailBody` when `n == 0`).
- **Shell / nav callbacks:** **`internal/app/shell.go`** — today **`gotoReview`** is passed into **`NewRejectedView`**; add **`gotoUpload`** (or generic `selectNavKey`) for Review empty state.
- **Mapped store errors:** **`internal/app/collection_store_err_text.go`** (**Story 2.9** pattern).

### File structure requirements

| Area | Files |
|------|--------|
| Empty / error UX | **`internal/app/review.go`**, **`internal/app/rejected.go`**, **`internal/app/collections.go`**, **`internal/app/shell.go`** |
| Optional shared copy | **`internal/app/collection_store_err_text.go`** or small new **`internal/app/user_error_text.go`** (only if justified) |
| Tests | **`internal/app/review_test.go`**, **`internal/app/collections_test.go`**, new **`rejected_test.go`** if no suitable home |

### Testing requirements

- Prefer **headless** `fyne.io/fyne/v2/test` patterns already used in **`collections_test.go`** and **`review_test.go`**.
- When injecting **nav callbacks**, use **`func()`** hooks that set **atomic** / **channel** flags in tests to prove CTAs fire.
- Manual smoke: **dark/light** swap, **1024×768** window, empty library after fresh DB, filter combination that yields zero rows, Rejected with zero hidden, album with zero photos.

### Previous story intelligence

- **Story 2.11** formalizes **NFR-01 / NFR-07** layout evidence; empty-state blocks must not push **primary shell nav** off-screen at minimum width ([Source: `_bmad-output/implementation-artifacts/2-11-layout-display-scaling-gate.md` — NFR-01 surfaces, `shell.go` / Review / loupe file map]).
- **Story 2.10** established **`collectionStoreErrText`** for user-facing DB errors and **assign** empty hints — keep tone consistent ([Source: `_bmad-output/implementation-artifacts/2-10-quick-collection-assign.md` — AC4, AC7]).

### Risks and mitigations

| Risk | Mitigation |
|------|------------|
| **Callback wiring explosion** | Prefer one **`onNavigate(key string)`** from shell over many one-off hooks. |
| **Filter reset races** | Reuse existing **`suspend*Refresh`** flags so `SetSelected` does not recurse `refreshReviewData` mid-reset. |
| **False “empty library”** | Base “library empty” strictly on **default** `domain.ReviewFilters{}` count, not on loupe state. |
| **Competing CTAs** | UX-DR9 allows **one primary**; secondary actions = **Medium** importance or **link-style** text. |
| **Err-string mapping drift** | `userFacingDialogErrText` uses substrings on `err.Error()`; regressions caught by **`collection_store_err_text_test.go`**; prefer **`store.Err…`** if a misfire appears in the field. |

### Definition of Done

- All AC1–AC7 satisfied; UX-DR9 surfaces show **one** clear primary CTA; error copy reviewed for **next steps** and **no raw driver text** in common paths.
- CI green including new/updated **`internal/app`** tests.

## Dev Agent Record

### Agent Model Used

Party mode **create** sessions 1/2 + 2/2 (simulated roundtable) + implementation pass (Cursor agent). Party mode **dev** session 1/2 (2026-04-14): simulated Amelia / Murat / Sally / Winston + `userFacingCollectionWriteErrText` + DnD copy tests. Party mode **dev** session 2/2: `droppedSkipSummaryForDialog`, non-file drop guidance, import **failed** next-step on receipt dialog.

### Debug Log References

### Completion Notes List

- UX-DR9 empty surfaces: Review, Rejected, Collections list/detail; `libraryErrText` for inline read errors; shell `gotoUpload` matches nav-button side effects (undo clear, Collections reset).
- Party session **2/2:** All **`dialog.ShowError`** call sites in Epic 2 surfaces + Upload now route through **`userFacingDialogErrText`** / **`userFacingFileOpenErrText`** (or existing `collectionStoreErrText` for assign links); unit tests lock mapping behavior.
- **2026-04-14 (dev-story):** Re-ran `go test ./...`, `go test -count=1 ./...`, and `go build .` — all passed; acceptance criteria and tasks already satisfied in tree; story remains **review**.
- **2026-04-14 (party dev 1/2):** Album write paths + DnD reject lines hardened for AC5; `go test ./...` green.
- **2026-04-14 (party dev 2/2):** `ShowInformation` skip lists capped; non-file drop copy; import complete message adds next step when `Failed > 0`; `go test ./...` green.

### File List

- `internal/app/review.go` — `reviewFiltersAtFR16Defaults`, empty state block, `onGotoUpload`, reset filters, `libraryErrText` in count errors, `suspendMinRatingSelectRefresh`, dialog errors via `userFacingDialogErrText`
- `internal/app/rejected.go` — default vs filter empty copy, reset filters, `libraryErrText`, strip suspend flags, dialog errors via `userFacingDialogErrText`
- `internal/app/collections.go` — `onGotoReview`, detail empty CTAs, list error copy, `newAlbumBtn` importance refresh, album form/delete errors mapped
- `internal/app/shell.go` — `gotoUpload`, constructor order for collections + review
- `internal/app/collection_store_err_text.go` — `libraryErrText`, `userFacingCollectionWriteErrText`, `userFacingDialogErrText`, `userFacingFileOpenErrText`
- `internal/app/collection_store_err_text_test.go` — mapping regression tests + `TestUserFacingCollectionWriteErrText` (dev 1/2)
- `internal/app/drop_paths.go` — `dropRejectReason`, `droppedSkipSummaryForDialog` (dev 1/2 + 2/2)
- `internal/app/review_loupe.go` — reject/delete/tag dialogs use `userFacingDialogErrText`
- `internal/app/upload.go` — file picker + post-import collection/link errors mapped; drop `ShowInformation` bodies use `droppedSkipSummaryForDialog`; `summarizeDoneMessage` failed next step (dev 2/2)
- `internal/app/drop_paths_test.go` — `TestDropRejectReason`, `TestDroppedSkipSummaryForDialog` (dev 1/2 + 2/2)
- `internal/app/review_test.go` — Story 2.12 tests, `collectButtons` + Scroll, `findLabelContaining`, closed-DB assertion relaxed for mapped errors
- `internal/app/rejected_test.go` — Story 2.12 Rejected tests
- `internal/app/collections_test.go` — Story 2.12 detail empty test

## Change Log

- **2026-04-14:** Dev-story workflow — verification only (`go test ./...`, `go build .`); story and sprint status unchanged at **review**.
- **2026-04-14:** Party mode dev session 1/2 — `userFacingCollectionWriteErrText`, DnD `dropRejectReason`, tests; story remains **review**.
- **2026-04-14:** Party mode dev session 2/2 — AC5 `ShowInformation` / receipt polish (`droppedSkipSummaryForDialog`, import failed next step); story remains **review**.

## References

- [Source: `_bmad-output/planning-artifacts/epics.md` — Story 2.12, UX-DR9 inventory, Epic 2 goals]  
- [Source: `_bmad-output/planning-artifacts/ux-design-specification.md` — UX-DR9 (epics inventory), §Anti-Patterns “Cheerful empty errors”, §2.5 Feedback (errors + next step)]  
- [Source: `_bmad-output/planning-artifacts/PRD.md` — FR-15, FR-16 (filter defaults)]  
- [Source: `_bmad-output/planning-artifacts/architecture.md` — Fyne desktop, store layering, slog + wrapped errors]  
- [Source: `internal/app/shell.go` — primary nav, `gotoReview` pattern]  
- [Source: `internal/app/review.go` — `refreshReviewData`, filter strip]  
- [Source: `internal/app/rejected.go` — Rejected empty hint + `Back to Review`]  
- [Source: `internal/app/collections.go` — list empty, detail `n == 0` label]  
- [Source: `_bmad-output/implementation-artifacts/2-11-layout-display-scaling-gate.md` — NFR-01 / layout constraints for new chrome]  
- [Source: `_bmad-output/implementation-artifacts/2-10-quick-collection-assign.md` — error tone / `collectionStoreErrText` precedent]  
