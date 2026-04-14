# Story 2.6: Reject, session undo, Rejected/Hidden, and restore

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **to hide bad shots without losing them**,  
So that **I can recover quickly and trust default views**.

**Implements:** FR-29, FR-30; UX-DR5, UX-DR8.

## Acceptance Criteria

1. **Reject hides from default surfaces (FR-29):** **Given** an asset visible in **Review** (thumbnail grid **or** loupe), **when** the user **rejects** it, **then** the row is updated so **`rejected = 1`** (and **`rejected_at_unix`** set per architecture), **and** the asset **disappears** from the current **default** Review query (`ReviewBrowseBaseWhere`: active, non-rejected). **And** the same asset **must not** appear in any future **share / package selection** path unless product explicitly adds an override (architecture §3.4: default **cannot share rejected**). *Current codebase:* no `internal/share` yet—**document and enforce** this rule at the store/API boundary when share mint lands (Epic 3); for this story, add a **single** reusable read helper if useful (e.g. `AssetRejectedState` / `SELECT rejected FROM assets WHERE id = ?`) so mint cannot drift.
2. **Session undo (FR-30 + architecture §3.4):** **Given** a reject just applied, **when** the user is still in the **same review context**, **then** they can **undo** the last reject(s) via **transient UI** (UX-DR8: capped stack—**no** unbounded toast list). **MVP rule (locked):** in-memory **LIFO** of `asset_id` scoped to **Review**; stack **clears** when the user **navigates to another primary shell destination** (Upload, Collections, Rejected, etc.) **or** **closes the app**. Undo performs a **DB restore** (clear reject) for the popped id(s). Optional **~30s** toast is polish only—not required for AC parity.
3. **Rejected / Hidden destination (FR-30):** **Given** primary nav **Rejected** (UX-DR13 label **“Rejected”**; subtitle copy may say **Rejected/Hidden** per shell placeholder), **when** the user opens it, **then** they see a **real list** of **`rejected = 1`**, **non-deleted** assets (not the Story 2.1 placeholder). **Given** **restore** on an asset there, **when** confirmed in UI (lightweight control—no need for modal unless you choose), **then** **`rejected`** clears (**`rejected_at_unix`** cleared or NULL), **and** the asset **reappears** in default Review / counts / grid subject to normal filters.
4. **Styling (UX-DR5):** **Reject** affordances use **caution** semantic (**`widget.WarningImportance`** / theme **reject–caution** role), **visually distinct** from **destructive** (Delete is Story **2.7**—**do not** ship delete here). **Keyboard:** Reject shortcut **must not** sit **adjacent** to **1–5** rating keys (extend Story **2.4** precedent in `review_loupe.go` package comment with the **chosen** binding).
5. **Transient undo feedback (UX-DR8):** Undo affordance is **transient** (banner, toast, or inline snack region)—**distinct** from **batch operation receipts** (UX-DR6). Cap simultaneous transient undo prompts (e.g. max **1–3** visible or single composite “Undo (N)”).

6. **Nav / undo scope (clarify MVP A):** Undo stack is **Review-only**. **Clearing** the stack happens when the user selects **any other primary nav destination** than **Review** (including **Rejected**, **Upload**, **Collections**) or closes the app. **Do not** clear the stack on **filter-only** changes inside Review (collection / rating / tags). *Rationale:* architecture §3.4 MVP (A); avoids unbounded cross-session undo.

7. **Store ↔ undo coupling (session 2):** Push onto the session undo stack **only** when **`RejectAsset` returns `changed == true`**. A duplicate tap / stale UI must not grow the stack when the row was already rejected.

8. **Undo stack memory bound (session 2):** Even inside a long Review session, cap retained undo ids (e.g. **max 128** — implementation choice, document constant). When full, **drop the oldest** ids (FIFO trim) while **Undo** still pops LIFO from the surviving tail. Transient UI may show at most **1–3** prompts (UX-DR8); the cap prevents pathological RAM use if the user never navigates away.

9. **Accessibility (session 2):** Reject and Restore controls expose **clear accessible names** (not icon-only without description). Respect **UX-DR15** focus-order expectations; for this story, do not regress keyboard grid/loupe navigation.

## Party mode — dev round **1/2** (automated, headless, hook dev)

_Simulated round — `_bmad/_config/agent-manifest.csv` not present._

**Amelia (Dev):** Bulk **Reject selected** was calling `refreshAll()` before the SQL loop. That dismisses the loupe fine but also **resets the grid and runs a full refresh** before any reject — double work and a pointless empty intermediate state. **Fix:** dismiss loupe only, then reject in a loop, then **one** `refreshReviewData`. Undo already tolerates `RestoreAsset` idempotency; **consume** the stack slot even when `changed` is false.

**Sally (UX):** I still resist **loud** onboarding copy, but people open **Rejected** and wonder where undo went. **Compromise:** a **single italic hint** under the undo button, **only while undo is visible** — not in the count line, not a toast.

**Winston (Arch):** Centralize the nav rule in a **named helper** so the “leave Review” predicate is not buried in the `RadioGroup` closure. **Push back** to Sally only on focus order: keep the hint **below** the undo control in the same cluster so the filter strip stays first in the mental model.

**Murat (TEA):** No unit covered “clear undo when leaving Review.” **Fix:** `clearReviewUndoIfLeftReview` + assertions in `shell_test.go`. A headless Fyne UI test is unnecessary — the pure helper contract is enough.

### Orchestrator synthesis — concrete edits (applied)

- `internal/app/review.go`: bulk reject — loupe dismiss only, then rejects + single refresh; **undo session hint** label (italic) with undo cluster.
- `internal/app/shell.go` + `shell_test.go`: **`clearReviewUndoIfLeftReview`** extracted and tested.
- `internal/app/review_loupe.go` / `review_grid.go`: button copy **“Reject photo”** / **“Restore photo”** for clearer accessible names (AC9).
- Story + sprint log updated; status remains **review** until manual QA row closes.

## Party mode — dev round **2/2** (automated, headless, hook dev; deepen)

_Simulated round — `_bmad/_config/agent-manifest.csv` not present._

**Sally (UX):** Session 1’s hint nailed **nav scope**, but surfacing **“128”** in end-user copy is engineer-brained noise. **Insist:** keep the numeric cap in **code only**; photographers get **plain language** (“oldest steps may drop off”) without the magic number.

**Amelia (Dev):** **Against** a single **transaction** around bulk reject for MVP — one failure would **roll back** earlier hides and blur what actually landed in the library. **Lock in** **per-row** applies with **stop-on-first-error**; earlier rows stay rejected and **undoable**. That matches current behavior; the gap was **documentation**, not a rewrite.

**Murat (TEA):** Session 1’s **`shell_test`** reused one counter across cases; **`rejected → upload`** was coupled to earlier invocations. **Refactor** to **parallel subtests** with fresh counters and add **`review → upload`**. Stack **`Push` for `id <= 0`** should stay a no-op — add a **tiny regression** so AC7 hygiene can’t slip.

**Winston (Arch):** **`RestoreAsset`**’s **`changed`** bit was **ignored** in the UI after session 1 — fine for UX, **weak** for supportability. **Use `changed`** in Review undo + Rejected restore and emit **`slog.Debug`** on no-op so “slot consumed, row already clean” is visible in logs without alarming the user.

### Orchestrator synthesis — concrete edits (session 2 applied)

- `internal/app/review.go`: undo hint adds **cap behavior** in **user language** (no literal 128); **best-effort bulk-reject** comment; undo uses **`RestoreAsset` `changed`** + debug on no-op.
- `internal/app/rejected.go`: restore uses **`changed`** + debug on no-op.
- `internal/app/reject_undo_stack_test.go`: **`TestReviewRejectUndoStack_pushNonPositiveIgnored`**.
- `internal/app/shell_test.go`: **subtests** + explicit **`review → upload`** clear case.

## Risks & mitigations (create review)

| Risk | Mitigation |
|------|------------|
| User rejects, switches to **Rejected** to verify, loses undo — feels like losing a safety net | MVP locked per architecture; optional micro-copy on undo surface (“Session undo — stays in Review”); backlog usability note if feedback is loud |
| **Idempotent** reject/restore (double-activate, stale id after refresh) | **Locked (session 2):** mutating APIs return **`(changed bool, err)`** with **`changed == false`** for no matching row / wrong state—**no** `sql.ErrNoRows` on `UPDATE`; UI avoids error storms; integration tests in `reject_test.go` |
| Loupe **stale pixmap** after reject if index/list refresh races | After successful reject: refresh list/count **before** stepping index or closing loupe; drop cached image for hidden id |
| Restore in **Rejected** while same `asset_id` still on Review undo stack | Stack pops **clear reject** — must be **idempotent** if flag already cleared |
| **Empty Rejected** list | UX-DR9: one primary CTA (e.g. back to Review), consistent with shell empty states |
| **Double-activate** reject builds a **phantom** undo stack (session 2) | Push stack **only** when `RejectAsset` reports `changed`; cap depth + trim oldest |
| **Rejected vs browse counts diverge** under filters (NFR-04) | **Same** `ReviewFilterWhereSuffix` after **`ReviewRejectedBaseWhere`**; parity tests (added) |

## Definition of Done (handoff)

- [x] Default Review queries still use **`ReviewBrowseBaseWhere`** only; rejected surfaces use **`ReviewRejectedBaseWhere`** + **`ReviewFilterWhereSuffix`** (no hand-rolled predicates in `internal/app`).
- [x] Store integration tests: reject → default count drops; restore → reappears; soft-deleted never in rejected list; filter suffix parity on rejected base. *(reject/restore/idempotency + rejected list/filter tests in `reject_test.go` / `review_query_test.go`)*
- [x] Reject/restore **idempotency** behavior documented and tested at store boundary. *(`RejectAsset` / `RestoreAsset` return `(changed bool, err)`; tests in `reject_test.go`.)*
- [x] **Soft-deleted** rows: reject/restore are **silent no-ops** (`changed == false`)—no user-visible error storm from stale loupe/grid ids. *(store tests)*
- [x] Loupe/grid stay aligned after reject (`refreshAll` / same hooks as Story 2.5); no stale large image.
- [x] Keyboard reject binding documented in **`review_loupe.go`** package comment (**not** adjacent to **1–5**).
- [x] Rejected nav shows **real list**, not placeholder; restore is **not** `DangerImportance`.

## Tasks / Subtasks

- [x] **Store layer: reject / restore / query** (AC: 1, 3)
  - [x] Add `RejectAsset` / `RestoreAsset` in `internal/store/reject.go` for **active** rows; **idempotent** `(changed bool, err)` — `changed == false` for missing / deleted / wrong state; **no** `sql.ErrNoRows` on mutates.
  - [x] **`ReviewRejectedBaseWhere`** (existing) + **`CountRejectedForReview` / `ListRejectedForReview`** in `review_query.go` with **`ReviewFilterWhereSuffix`** parity to default Review.
  - [x] Integration tests: round-trip, idempotency, soft-deleted no-op, rejected list count/list + min-rating + collection filters.
- [x] **Review UI: reject action** (AC: 1, 4, 5)
  - [x] **Grid:** **Reject** control on chrome or cell overflow (caution styling); minimum **single-asset** reject for visible selection (if multi-select active, define **one** behavior: reject **primary** selection vs **all** selected—document in code comment; epic AC is single-asset—**all selected** is acceptable if consistent with tags bulk patterns).
  - [x] **Loupe:** **Reject** button with same semantics; after reject, **refresh** list and either **step** to next image or **close** loupe if the result set no longer contains the current index—**no** stale pixmap on a hidden id.
  - [x] Wire **`refreshAll`** (or equivalent) so count/grid/loupe stay aligned with **`CountAssetsForReview` / `ListAssetsForReview`** after reject.
- [x] **Session undo stack** (AC: 2, 5, 6, 7, 8)
  - [x] Small **LIFO** (`[]int64` or linked stack) owned by **Review** view state; **push** only when **`RejectAsset` → `changed`**; **pop** on Undo; **cap** depth (~128) and **drop oldest** when over cap.
  - [x] **Clear stack** when shell navigates **away from Review** to **any** other primary destination (including **Rejected**), **not** when filters change inside Review (hook from `shell.go` **or** callback when `RadioGroup` changes panel—avoid scattering if one listener suffices).
  - [x] Transient UI: **Undo** control appears after reject; **capped** stack / single surface (UX-DR8).
- [x] **Rejected panel: replace placeholder** (AC: 3, 4)
  - [x] New **`NewRejectedView(...)`** (or similar) with paged grid/list reusing **`review_grid`** patterns **or** a slim list—**same thumbnail decode path** where practical for consistency.
  - [x] **Restore** action per row (caution vs primary is OK; **restore** is **not** destructive—avoid **Danger** styling).
  - [x] **Empty state** (UX-DR9): one primary CTA when `CountRejectedForReview` is 0.
  - [x] Wire **`shell.go`** to use real view instead of **`NewSectionPlaceholder`** for **`rejected`**.
- [x] **Share guardrail (future-proof FR-29)** (AC: 1)
  - [x] `AssetEligibleForDefaultShare` in `reject.go` + **`// TODO(Epic 3)`** to call from mint; rejects ineligible when `rejected=1` or soft-deleted.
- [x] **Tests & QA** (AC: 1–6)
  - [x] Store tests for reject/restore/rejected-list + filter suffix parity.
  - [x] Optional: pure **unit** test for stack **clear-on-nav** contract if extracted from Fyne. *(LIFO, cap/trim, Clear covered in `reject_undo_stack_test.go`; nav clear wired in `shell.go`.)*
  - [ ] Manual: keyboard reject binding **not** adjacent to **1–5**; light/dark caution visibility.

## Dev Notes

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **Default queries** | **`ReviewBrowseBaseWhere`** = `rejected = 0 AND deleted_at_unix IS NULL`—already central in `internal/store/review_query.go`; **do not** fork this predicate in `internal/app`. |
| **Rejected bucket queries** | **`rejected = 1 AND deleted_at_unix IS NULL`** + same filter suffix as default Review (architecture §3.4). |
| **Undo scope** | Architecture §3.4 **MVP (A):** session stack in **Review** until **shell nav away** or app exit; **(B):** **Rejected** panel **always** supports **restore**. |
| **Share** | Rejected assets **excluded** from selection and mint validation by default (architecture §3.4, PRD FR-32 / FR-29). |
| **`rejected_at_unix`** | Column exists in `001_initial.sql`; set on reject, clear on restore for sensible sorting/auditing. |

### Technical requirements

- **Single predicate source:** Extend **`review_query.go`** with rejected-list helpers; **no** hand-rolled `WHERE` in widgets.
- **Transactions:** Single-asset reject/restore can use one `Exec` each; bulk reject (if implemented) should follow chunked transaction patterns from `tags.go` only if needed.
- **Logging:** `log/slog` on unexpected DB errors; user-visible strings stay free of raw SQL.
- **Loupe / grid parity:** Reuse **`ListAssetsForReview`** ordering (`capture_time_unix DESC, id DESC`) for stepping; after reject, total/index handling must match **`loupeStepIndex`** expectations.

### Architecture compliance

- **§3.4:** Reject, undo, restore, default vs rejected queries, share exclusion.
- **§3.8 / §4.5:** Fyne in `internal/app`, SQL in `internal/store`; default browse excludes rejected/deleted.
- **§5.2:** No SQL in UI—call store APIs.

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3** (`go.mod`): **`WarningImportance`** for reject; follow **`review_loupe.go`** / **`review_grid.go`** patterns.

### File structure requirements

- **Extend:** `internal/store/review_query.go`, `internal/store/review_query_test.go`, `internal/store/reject.go`, `internal/store/reject_test.go`.
- **Extend:** `internal/app/review.go`, `internal/app/review_grid.go`, `internal/app/review_loupe.go`, `internal/app/shell.go`.
- **Add:** `internal/app/rejected.go` (optional) if the Rejected surface is large enough to isolate.

### Testing requirements

- **Mandatory:** SQLite temp-DB tests for reject/restore, rejected list **count/list parity**, and filter suffix behavior under **`rejected = 1`** base.
- **Regression:** Stories **2.2–2.5** review filter tests remain green; **`ReviewBrowseBaseWhere`** unchanged for default paths.

### Previous story intelligence (Story 2.5)

- **`refreshAll`**, **`buildFilters`**, and loupe **`onReviewDataChanged`** are the **correct** hooks to keep grid/count/strip aligned after state mutations—reject **must** trigger the same refresh discipline as tag writes.
- **Multi-select** uses **`tapLayer`** / **`desktop.Mouseable`**—if bulk reject is implemented, reuse selection semantics and document edge cases (empty selection, stale ids).
- **Count line** already surfaces some errors—reject failures should be **user-visible** (dialog or count hint), not **log-only**.

### Git intelligence summary

- Recent review work lives under `internal/app/review*.go` and `internal/store/review_query.go`; **`shell.go`** still hosts the Rejected **placeholder**—this story **replaces** it.

### Latest technical information

- No third-party dependency changes anticipated; SQLite `INTEGER` 0/1 for `rejected` matches existing CHECK.

### Project structure notes

- No `project-context.md` in repo; planning artifacts + this file define the contract.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Story 2.6, FR-29/FR-30, UX-DR5/UX-DR8 inventory]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.4 Reject, undo, and delete; §3.5 share validation note]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-29, FR-30, FR-32, Journey E]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — Reject vs Delete, caution/destructive, transient undo, Journey E]
- [Source: _bmad-output/implementation-artifacts/2-4-review-loupe-keyboard-rating.md — UX-DR5 non-adjacent reject]
- [Source: _bmad-output/implementation-artifacts/2-5-tags-bulk-review.md — refreshAll / multi-select patterns]
- [Source: internal/store/review_query.go — `ReviewBrowseBaseWhere`, `ReviewRejectedBaseWhere`, `ReviewFilterWhereSuffix`, list/count]
- [Source: internal/store/migrations/001_initial.sql — `rejected`, `rejected_at_unix`]
- [Source: internal/app/shell.go — Rejected placeholder to replace]
- [Source: internal/app/theme.go — reject/caution semantic role]

## Dev Agent Record

### Agent Model Used

Party mode dev **1/2** + **2/2** (simulated); implementation refined in-repo after initial UI land.

### Debug Log References

### Completion Notes List

- **2026-04-13:** Store API + tests landed; UI/shell tasks remain.
- **2026-04-13:** Completed Fyne UI: grid **Reject selected photos** (`WarningImportance`) with bulk = all selected (sorted ids); loupe **Reject** + **R** shortcut; `refreshReviewData` / `invalidatePages` after reject; session `reviewRejectUndoStack` (cap 128, FIFO trim) with push only on `changed`; undo banner; shell clears stack on primary nav away from Review; **`NewRejectedView`** with same filter strip + thumbnail grid + per-row **Restore** (`MediumImportance`) + empty CTA **Back to Review**; **`review_grid`** `rejectedMode` uses `ListRejectedForReview` / `CountRejectedForReview`.
- **2026-04-13 (party dev 1/2):** Bulk reject avoids pre-loop `refreshAll`; undo scope hint under undo control; `clearReviewUndoIfLeftReview` + test; loupe/grid button labels **Reject photo** / **Restore photo**.
- **2026-04-13 (party dev 2/2):** Undo hint wording (cap without exposing 128); `RestoreAsset` `changed` + `slog.Debug` in Review/Rejected; bulk-reject best-effort comment; stack non-positive push test; shell clear-on-nav subtests + `review→upload`.

### Change Log

- **2026-04-13:** Story 2.6 UI + undo + Rejected panel implemented; stack unit tests added; sprint status → review.
- **2026-04-13:** Party mode dev 1/2 — UX hint for undo scope, bulk-reject refresh discipline, nav-clear helper test, a11y button copy.
- **2026-04-13:** Party mode dev 2/2 — observability on restore no-ops, clearer undo-cap copy, test hardening.

### File List

- `internal/store/reject.go`
- `internal/store/reject_test.go`
- `internal/store/review_query.go`
- `internal/store/review_query_test.go`
- `internal/app/reject_undo_stack.go`
- `internal/app/reject_undo_stack_test.go`
- `internal/app/rejected.go`
- `internal/app/review.go`
- `internal/app/review_grid.go`
- `internal/app/review_loupe.go`
- `internal/app/review_test.go`
- `internal/app/shell.go`
- `internal/app/shell_test.go`
- `_bmad-output/implementation-artifacts/2-6-reject-undo-hidden-restore.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
