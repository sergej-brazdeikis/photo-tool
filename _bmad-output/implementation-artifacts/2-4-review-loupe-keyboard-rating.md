# Story 2.4: Review loupe with safe chrome and keyboard rating

Status: review

<!-- Ultimate context engine analysis completed — comprehensive developer guide created. -->

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **a large letterboxed view with keyboard 1–5 and prev/next**,  
So that **I can review on any aspect ratio monitor**.

**Implements:** FR-09–FR-12, FR-25 (desktop portion); NFR-01; UX-DR4, UX-DR5.

## Acceptance Criteria

1. **Given** an asset opened from the grid, **when** the loupe opens, **then** the image uses up to **90%** of the **available review viewport** (the main content area for Review—account for shell nav/chrome) and the **entire** image is visible **without cropping** (letterboxed / `ImageFillContain` semantics) (**FR-09**, **FR-12**, **FR-25** image rules; **UX-DR4**).
2. **Given** keyboard **1–5** **or** an on-screen star control, **when** the user sets a rating, **then** the value **persists to SQLite** (`assets.rating`) **without** an extra confirmation dialog (**FR-10**). *Scope note:* “stars” may be **five discrete controls** (not a blocking modal); behavior must match **instant persist** semantics.
3. **Given** window aspects from **1:1** through **21:9**, **when** the window is resized (manual QA aligned to **NFR-01** matrix intent), **then** **primary loupe chrome** (rating, prev/next, close/back—whatever ships in this story) remains **visible and usable** without requiring scroll to find them (**FR-11**, **NFR-01**). *MVP:* document chosen layout strategy (e.g. **Border** + max image region ~90%, chrome in **non-image** bands or pinned overlay with safe margins).
4. **Given** the user is in the loupe, **when** they use **prev/next** (on-screen **and** keyboard shortcuts—e.g. arrow keys—**desktop FR-25**), **then** navigation moves **backward/forward** through the **same ordered result set** as the grid (**stable sort**: `capture_time_unix DESC`, `id DESC` tie-break, same **`domain.ReviewFilters`** and **`store.ListAssetsForReview` / `CountAssetsForReview`** as the grid). **MVP behavior:** **clamp at ends**—at the **first** asset **Prev** is disabled and Left-arrow is a no-op; at the **last** asset **Next** is disabled and Right-arrow is a no-op (**no wrap**). **No** silent jumps to a different filter or sort.
5. **Keyboard safety (UX-DR5):** **Do not** bind **Reject** (or any Story **2.6** hide action) to keys **immediately adjacent** to **1–5** on a typical QWERTY layout. For this story, **either** omit Reject shortcuts entirely **or** document a deliberate non-adjacent binding (e.g. not `4`/`5`/`6` neighbors). **No** “accidental reject” regression from rating keys.
6. **Given** the loupe is open, **when** the user presses **Escape**, **then** the loupe closes and focus returns to the grid without leaving Review (**UX-DR15** focus order: document strip → grid → loupe in Dev Agent Record / code comment). **Tab** order inside the loupe must reach rating + prev/next + close **before** any purely decorative controls.
7. **Filter consistency (session 2 challenge):** **Given** the loupe is open, **when** the user changes **Collection** or **Minimum rating** (or any control that triggers a **new** `CountAssetsForReview` / `ListAssetsForReview` result set), **then** the loupe **closes** (or is explicitly resynced to the new set with a defined index rule). **MVP:** **auto-dismiss** the loupe on refresh so the grid and loupe cannot disagree silently.

## Tasks / Subtasks

- [x] **Pass `fyne.Window` into grid + loupe** (AC: 1, 3, 5, 6, 7)
  - [x] Replace ignored `win` / `nil` window in `newReviewAssetGrid` with the real `win` from `NewReviewView` so popups, shortcuts, and focus behave correctly.
  - [x] Reconcile `shell.go`: Review is wrapped in `container.NewScroll(...)` — loupe uses **`widget.NewModalPopUp` on `win.Canvas()`** (full-window overlay), so nested scroll does not constrain loupe layout (documented in Dev Notes / `review_loupe.go`).
- [x] **Store: persist rating** (AC: 2)
  - [x] Add `UpdateAssetRating` in `internal/store` — `UPDATE assets SET rating = ? WHERE id = ? AND deleted_at_unix IS NULL`, honoring `003_review_filters.sql` **CHECK** (`NULL` or **1–5**); Go validates **1–5** before query.
  - [x] Tests in `internal/store/assets_rating_test.go` (temp SQLite): valid write, invalid rating, missing id, soft-deleted row.
- [x] **Open loupe from grid** (AC: 1, 4, 7)
  - [x] Add tap/click affordance on each grid cell (`tapLayer` + `Handler` rebound per `List` bind) that opens loupe for that asset’s **flat index** in the filtered list (`row*reviewGridColumns + col`).
  - [x] Loupe reads **`domain.ReviewFilters` + index** from `reviewAssetGrid` and uses **`store.ListAssetsForReview`** / grid `total` (parity with grid).
- [x] **Loupe UI: image + chrome** (AC: 1–3)
  - [x] Full-resolution image load: **file path** from `filepath.Join(libraryRoot, filepath.FromSlash(rel_path))`; **async** `os.Stat` + `fyne.Do` + `canvas.Image` file bind (same spirit as Story 2.3 async thumbs).
  - [x] **Letterbox** with `canvas.Image` + `ImageFillContain`; **`loupeImageLayout`** sizes the image stack to **~90%** of the loupe body (architecture **§3.8** custom layout).
  - [x] Five rating buttons + **Prev** / **Next** / **Close** (importance highlights current star).
- [x] **Keyboard: 1–5 and navigation** (AC: 2, 4, 5, 6)
  - [x] While loupe is open: **`Canvas().SetOnTypedRune` / `SetOnTypedKey`**; **`nil` on close** (idempotent `cleanup`).
  - [x] **Escape** closes loupe; shortcut table in **`review_loupe.go`** package comment (**UX-DR5**: no reject/delete keys).
- [x] **Prev/next data path** (AC: 4)
  - [x] `ListAssetsForReview(..., limit=1, offset=idx)` with shared filters; **end clamp** via `loupeStepIndex`; prev/next **disabled** at ends.
  - [x] Index change bumps **async generation** token; decode/missing file uses **`reviewGridMsgDecodeFail`** / page errors use **`reviewGridMsgPageLoadFail`**.
- [x] **Grid ↔ loupe consistency** (AC: 2, 7, **SC-3** / **FR-10** display)
  - [x] After rating write: **`reviewAssetGrid.invalidatePages()`** + list refresh; **changing filters dismisses** an open loupe (`refreshAll` in `review.go`).
- [x] **Tests** (AC: 2, 4, 5, 6)
  - [x] Store tests for rating update (`assets_rating_test.go`).
  - [x] Pure-Go **`loupeStepIndex`** (incl. OOB/negative start index) + **`loupeRatingKeyAllowed`** tests (`review_loupe_test.go`). **Manual:** NFR-01 resize matrix remains Story 2.11 / QA.

## Dev Notes

### Risks & mitigations (BMAD party — create round, session 1/2)

| Risk | Mitigation |
|------|------------|
| Nested **`container.NewScroll(review)`** in `shell.go` breaks modal/loupe sizing or eats scroll gestures | Prefer **window-level overlay** (`canvas.Overlay` / `widget.PopUp`) or temporary **stack** swap so loupe owns layout; document chosen approach in `review_loupe.go`. |
| **Window-level** rune handlers leak after close → duplicate key handling on grid | Centralize register/unregister in loupe `OnClose`; pair with deferred cleanup or explicit `Canvas().SetOnTypedRune(nil)` pattern Fyne supports. |
| **Prev/next** races async full-res decode → stale image or rating | Reuse Story 2.3 **binding token** / `sync.Map` pattern; cancel or ignore completions when index changes. |
| **OFFSET** stepping on huge libraries | Acceptable for MVP (single-step `LIMIT 1`); flag for later “cursor by id” optimization if profiling demands. |
| **Tags** filter UI is a placeholder (`ReviewFilters` has **no** tag field yet) | Loupe/grid parity = **collection + minimum rating** only until a tags story extends `domain.ReviewFilters` and `ReviewFilterWhereSuffix`; do not invent ad-hoc tag SQL in the app layer. |

### Risks & mitigations (BMAD party — **create** round, session **2/2** — deepen / challenge)

| Risk | Mitigation |
|------|------------|
| **FR-09 “viewport”** vs **modal** covering the *whole* window (including shell nav): strict reading says “review viewport,” not global canvas | **MVP stance:** modal uses **full `win.Canvas()`**; the **image** is ~**90% of the loupe content** region (not the OS window chrome). Ultra-strict interpretation (“only Review body”) deferred — would need a **non-modal overlay clipped to the center stack** (higher complexity). |
| **Filter / count drift** while loupe is open (user changes strip; grid resets; loupe still steps old `OFFSET`) | **AC7:** `refreshAll` **dismisses** the loupe before applying a new filter snapshot — no silent index mismatch. |
| **`widget.List` recycling** + tap handler: closure captures wrong `idx` or stale `reviewGridCell` header | **Session 2 fix:** `tapLayer.Handler` is **rebound every bind**; no long-lived closure over a synthetic `parseReviewGridCell` struct for routing. |
| **Rating `OnTapped` closures** referencing `loadRow` before assignment | Declare `var loadRow func(); loadRow = func() { … }` for safe self-reference (compile-time footgun called out in dev review). |
| **Key repeat / double fire** on hold (OS sends repeated runes) | Acceptable MVP: each repeat is an idempotent **UPDATE**; optional debounce later if UX feels noisy. |

### Risks & mitigations (BMAD party — **dev** hook, session **1/2**, automated)

| Risk | Mitigation |
|------|------------|
| **`ListAssetsForReview` error / empty row** at current `idx` leaves **stale `currentID`** from a prior successful load | Set **`currentID = 0`**, **disable** star buttons and clear **`OnTapped`**; **keyboard 1–5** gated by **`loupeRatingKeyAllowed`** (`internal/app/review_loupe.go`). |
| **Guard logic drifts** between keyboard and future star shortcuts | Single helper **`loupeRatingKeyAllowed`** + **`TestLoupeRatingKeyAllowed`**. |

### Risks & mitigations (BMAD party — **dev** hook, session **2/2** — deepen / challenge)

| Risk | Mitigation |
|------|------------|
| **`idx` / `grid.total` desync** if a future code path calls `reset` without pairing **`dismissLoupe`** (or equivalent) | **`loadRow` + `nav`** clamp `idx` into **`[0, total-1]`** when `total > 0`; **`loupeStepIndex`** normalizes negative or past-end `idx` before applying delta so prev/next stay deterministic. |
| **Chrome state lags** after `idx` clamp or row-load failure | **`defer applyChrome()`** inside **`loadRow`** so **Prev/Next** enablement tracks the post-clamp index on every exit path. |
| **“Modal won’t track resize”** (NFR-01 worry) | Fyne **`modalPopUpRenderer.Refresh`** resizes the modal widget when **`Canvas.Size()`** changes; no ad-hoc **`SetOnResized`** unless manual QA proves a driver gap (defer to Story **2.11**). |

### Technical requirements

- **Single source of truth:** Grid and loupe must use the **same** `domain.ReviewFilters` and **`store.ListAssetsForReview` / `CountAssetsForReview`** contracts (**Story 2.2–2.3**). Do **not** duplicate `WHERE` clauses.
- **No SQL in widgets:** Keep queries in `internal/store`; UI binds rows only.
- **Logging:** On unexpected failures, use **`log/slog`**; user-visible strings stay free of driver/SQL fragments (**Story 2.3** precedent).
- **Reject / Delete / Share:** **Out of scope** for this story unless a **non-adjacent** keyboard binding is explicitly specified; **UX-DR5** is about **not** endangering rating keys.

### Architecture compliance

- **§3.8:** Custom layout allowed for **loupe**; **paged** list navigation should not load the whole library into memory.
- **§3.12 / §4.5:** Default browse rules remain via shared review query helpers; rating updates must respect **`deleted_at_unix IS NULL`** active rows.
- **§5.2:** Fyne in `internal/app`, persistence in `internal/store`.

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3** (per `go.mod`) — prefer composition (`container.Border`, `widget.Toolbar`/`HBox`) before heavy custom draw.

### File structure requirements

- **Extend:** `internal/app/review.go` (wire window, dismiss loupe on filter refresh), `internal/app/review_grid.go` (`tapLayer`, `invalidatePages`, `onLoupeOpen`).
- **Add:** `internal/app/review_loupe.go` + `internal/app/review_loupe_test.go` (index clamp tests).
- **Extend:** `internal/store/assets.go` or `review_query.go` for rating `UPDATE` (+ tests alongside `review_query_test.go` or new `assets_rating_test.go`).

### Testing requirements

- **Mandatory:** SQLite integration tests for rating persistence and constraint behavior.
- **Manual / Story 2.11:** Record NFR-01 resize notes when possible; this story should **not** block on full matrix documentation.

### Previous story intelligence (Story 2.3)

- **Paged grid + `ensurePageLocked`:** navigation should reuse **offset math** consistent with `reviewGridPageSize` pages but **per-asset** `LIMIT 1 OFFSET idx` is acceptable for loupe stepping.
- **Async thumbnails:** reuse **`fyne.Do`**, **`sync.Map`** / binding token pattern to avoid stale images when prev/next races.
- **`libraryRoot` + `filepath.FromSlash(rel_path)`** for all disk reads.
- **Grid currently ignores `win`:** fix by threading `fyne.Window` for loupe/modal UX.

### Git intelligence summary

- Repo history is thin; treat **implementation artifacts** and **`internal/app/review_grid.go`** as the live pattern source.

### Latest technical information

- **Fyne v2.7.3:** When attaching window-level rune handlers for the loupe, ensure **cleanup on close** so the grid does not receive duplicate key handling; test focus when returning to the grid.

### Project structure notes

- No `project-context.md` in repo; planning artifacts + this file define the contract.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 2, Story 2.4]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-09–FR-12, FR-25, FR-10, NFR-01, SC-3]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.8 Desktop UI (loupe custom layout, grid performance), §4.5 default queries, §5.2 boundaries]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — §Review loupe (90%, letterbox, prev/next), UX-DR4–UX-DR5, focus order UX-DR15, keyboard triage]
- [Source: _bmad-output/implementation-artifacts/2-3-thumbnail-grid-rating-badges.md — grid paging, async decode, user-facing error copy, `ListAssetsForReview`]
- [Source: internal/app/review.go — `NewReviewView`, filter strip]
- [Source: internal/app/review_grid.go — grid cells, `ListAssetsForReview` paging]
- [Source: internal/app/shell.go — Review embedded in shell + scroll]
- [Source: internal/store/review_query.go — `ReviewGridRow`, `ListAssetsForReview`, filter suffix]
- [Source: internal/store/migrations/003_review_filters.sql — `rating` column CHECK]

## Dev Agent Record

### Agent Model Used

BMAD party mode (automated headless). **`_bmad/_config/agent-manifest.csv`** was not present; rounds **simulated** (distinct voices + orchestrator synthesis) per party-mode guidance when subagents are unavailable. Latest automation: **dev** hook, session **2/2** (this run).

### Debug Log References

### Completion Notes List

- **2026-04-13 (party session 1):** `store.UpdateAssetRating` + `assets_rating_test.go` landed; story AC4 pinned to **end-clamp** (no wrap); AC6 (**Escape**, focus order) added; risks table + tags-filter scope note added; sprint status → **in-progress**.
- **2026-04-13 (party session 2):** Challenged **viewport vs full-canvas modal** semantics; added **AC7** (filter refresh dismisses loupe); fixed **List recycling** via **`tapLayer.Handler`** rebound each bind; implemented **`openReviewLoupe`** (`ModalPopUp`, `loupeImageLayout` ~90%, canvas keyboard hooks with cleanup), **`invalidatePages`**, **`loupeStepIndex` tests**; **`NewReviewView` uses `win`** for grid/loupe.
- **2026-04-13 (dev-story):** Re-validated all ACs against `review_loupe.go` / `review.go` / grid wiring; `go test ./...` and `go build .` green; **UX-DR15 focus:** `cleanup` calls `cnv.Focus(grid.list)` after close; filter strip → scroll/grid → loupe overlay documented in `review_loupe.go` comment and here. Story → **review**; sprint `2-4-review-loupe-keyboard-rating` → **review**.
- **2026-04-13 (party dev1/2):** Fixed **stale `currentID`** on loupe row-load failure; added **`loupeRatingKeyAllowed`** + **`TestLoupeRatingKeyAllowed`**; stars **disabled** when the current index cannot be resolved.
- **2026-04-13 (party dev2/2):** Hardened **`loupeStepIndex`** for **OOB/negative `idx`**; defensive **`idx` clamp** in **`nav`/`loadRow`**; **`defer applyChrome()`** in **`loadRow`**; added **`TestLoupeStepIndex_clampsOOBStartIndex`**; **`go test ./...`** green.

### Simulated party round (session 2/2)

**Mary (Analyst):** Session 1 under-specified **filter change while loupe is open** — `OFFSET` navigation can silently diverge from user intent. Need an explicit AC: **dismiss or resync** on refresh.

**Winston (Architect):** A strict read of FR-09 "viewport" is the **Review body**, not the full **`win.Canvas()`** — modal covers shell nav. MVP is acceptable **only** if we document that **~90% applies to loupe content**, not the dimmed chrome; a stricter future is a **clipped overlay** on the center stack only.

**Sally (UX):** Full-canvas immersive review is defensible; dimming nav is fine if **Escape** returns focus to the grid (**AC6**). Higher risk is **OS key-repeat** spamming rating writes than nav being temporarily obscured.

**Amelia (Dev):** **`parseReviewGridCell` + recycled rows** make tap closures treacherous — route taps through **`tapLayer.Handler`** re-bound on each list update. **`loadRow` self-calls** require `var loadRow func()` forward declaration.

**Orchestrator synthesis (applied):** Add **AC7** + **auto-dismiss on `refreshAll`**; ship **modal loupe** with documented **90%-of-loupe-body** image region; **rebind tap handlers** per row; add **`loupeStepIndex` tests**; keep sprint **in-progress** for NFR-01 manual passes / polish.

### Simulated party round (automated headless, **dev** hook, session **1/2**)

**Amelia (Dev):** If `ListAssetsForReview` fails or returns no row, we **return early** but **`currentID` still points at the previous asset** — keyboard **1–5** keeps writing to the wrong id. That is a real correctness bug, not just noisy logs.

**Murat (Test Architect):** `UpdateAssetRating` already rejects id **0** with “no row updated,” so we do not corrupt SQLite — but we should still **encode the invariant** in a tiny unit test so a future refactor does not drop the guard.

**Sally (UX):** Users in an error state should not think stars still apply. **Disable** the discrete star controls and clear handlers; keyboard silence matches **disabled** affordance.

**Winston (Architect):** Centralize “may apply keyboard rating” in one predicate the UI and tests share; avoid scattering `currentID <= 0` checks that diverge later.

**Orchestrator synthesis (applied):** On row-load failure, **`currentID = 0`**, **disable** rating buttons and **`OnTapped = nil`**; on success, **re-enable** stars before rebinding; **`loupeRatingKeyAllowed` + `TestLoupeRatingKeyAllowed`**; **`go test ./...`** green; sprint remains **review**.

### Simulated party round (automated headless, **dev** hook, session **2/2**)

**Winston (Architect):** Session 1 fixed **stale `currentID`**; session 2 should **stress invariants**, not re-litigate that bug. The real latent risk is **`idx` drifting past `grid.total-1`** if a future maintainer calls **`reset`** without **`dismissLoupe`**. Don’t pretend `refreshAll` is the only entry point forever.

**Amelia (Dev):** I’m skeptical of **window-level resize listeners** — Fyne’s **modal popup renderer** already **`Resize`s to `Canvas.Size()`** on refresh. Adding **`SetOnResized`** hooks without a failing QA artifact is speculative complexity.

**Murat (Test Architect):** Agree on not mocking the driver blindly — but we still owe **pure-Go proof** that navigation math is safe when **`idx` is garbage**. **`loupeStepIndex`** should define behavior for **OOB and negative** starts; today it’s implicit.

**Sally (UX):** If we clamp indices, **Prev/Next disabled state** must update in the same frame as the letterbox refresh — otherwise users at the last row see **Next** enabled for a beat after a shrink. That’s a **micro-glitch**, not a spec change.

**Orchestrator synthesis (applied):** Normalize **`idx`** inside **`loupeStepIndex`**; clamp in **`nav`** / **`loadRow`** defensively; run **`applyChrome` via `defer` in `loadRow`** so chrome tracks clamped index and error exits; add **`TestLoupeStepIndex_clampsOOBStartIndex`**; document **Fyne modal resize** vs **2.11** manual matrix in the risks table — **no sprint status change** (stay **review** until **2.11** NFR-01 evidence).

### File List

- `internal/store/assets.go` — `UpdateAssetRating`
- `internal/store/assets_rating_test.go` — rating persistence tests
- `internal/app/review.go` — `win` threading, loupe wiring, dismiss on filter refresh
- `internal/app/review_grid.go` — `tapLayer`, `onLoupeOpen`, `invalidatePages`
- `internal/app/review_loupe.go` — modal loupe, layout, keyboard shortcuts (see package comment)
- `internal/app/review_loupe_test.go` — navigation clamp tests
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — story **review**
- `_bmad-output/implementation-artifacts/2-4-review-loupe-keyboard-rating.md` — status / agent record (this file)

## Change Log

- **2026-04-13:** Dev-story validation complete; all tasks already implemented; `go test ./...` and `go build .` passed; story and sprint entry marked **review**.
- **2026-04-13:** Party **dev** session **2/2**: loupe index/chrome hardening + `review_loupe_test` coverage; story risks/agent record updated.
