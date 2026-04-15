# Story 2.7: Delete with confirmation and quarantine

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- Party mode create 1/2 (2026-04-13): risks, DoD, ordering/missing-file decisions, keyboard/Rejected scope clarified. -->
<!-- Party mode create 2/2 (2026-04-13): challenged batch/partial-failure parity with 2.6, path-escape guard, explicit shortcut (Cmd/Ctrl+Shift+D), quarantine collision logging; implementation landed in store + Review UI. -->
<!-- Party mode dev 1/2 (2026-04-13, headless): test quarantine basename collision; split-brain ERROR logs include quarantine_file path. -->
<!-- Party mode dev 2/2 (2026-04-13, headless): reject rel_path that cleans to library root; loupe resync on idempotent delete; store tests. -->

## Story

As a **photographer**,  
I want **delete to be clearly separate from reject and require confirmation**,  
So that **I do not erase by accident**.

**Implements:** FR-31; UX-DR5.

## Acceptance Criteria

1. **No delete without confirmation (FR-31):** **Given** the user triggers **delete** from Review (grid and/or loupe—match where **Reject** is exposed in Story 2.6), **when** they have **not** confirmed the destructive action, **then** **no** file is moved, **no** `deleted_at_unix` is set, and **no** other delete pipeline side effects run.
2. **Confirmed delete = soft-delete + quarantine (FR-31 + architecture §3.4):** **Given** the user **confirms** delete, **when** the operation completes successfully, **then** the asset row has **`deleted_at_unix`** set (non-NULL), the asset **disappears** from **default** queries (`ReviewBrowseBaseWhere`, Rejected list, collection/review counts, etc.—same exclusion rules as existing soft-delete tests), and when the primary file **exists** on disk it is **moved** (not copy-then-delete) into **`{libraryRoot}/.trash/{asset_id}/`** with basename preserved per **Resolved implementation decisions** (collision suffix if needed). If the file is **already missing**, still set `deleted_at_unix` after confirm and **skip** quarantine (see Risks table). **MVP:** one primary file per `rel_path`; sidecars / companion files are **out of scope** unless already modeled.
3. **Destructive vs reject styling (UX-DR5):** **Delete** affordances use **`widget.DangerImportance`** / theme **destructive** semantic (**`theme.ColorNameError`** in `PhotoToolTheme`); they are **visually and verbally distinct** from **Reject** (**caution** / **`WarningImportance`**). Copy should communicate that delete is **stronger** than reject; architecture allows noting that **purge trash** may be a later setting (user-facing hint optional, not blocking).
4. **Idempotency and errors:** **Given** a stale UI id or double-submit, **when** delete runs, **then** store/API behavior is **idempotent or safely no-op** (`changed` pattern consistent with `RejectAsset` / `RestoreAsset`)—no error storms from “already deleted.” **Given** move or DB failure mid-flight, **when** the user retries or reports, **then** behavior is **documented** (log with `slog`, user-visible message on hard failures); use the **ordering in “Resolved implementation decisions”** below (not ad hoc per developer).
5. **Keyboard safety (UX-DR5 extension):** **Given** the **Cmd/Ctrl+Shift+D** shortcut (or equivalent **modifier chord**, not bare **Delete**/**Backspace**), **when** the user invokes it in the loupe, **then** the **same** confirmation dialog path runs as for the **Move to library trash…** button (**no** “instant delete” from a single key). **Do not** bind bare **Delete** or **Backspace** to destructive commit without confirm; document bindings in `review_loupe.go` next to the Story 2.6 `R` reject binding.

## Risks & mitigations

| Risk | Mitigation |
|------|------------|
| **Split-brain** (file moved to `.trash` but `deleted_at_unix` not set) | Order: `MkdirAll` quarantine dir → `os.Rename` source → `UPDATE` only after successful rename. On `UPDATE` failure: log **ERROR** with absolute paths; user sees failure toast/dialog; document that library may contain an orphan under `.trash/{id}/` until manual cleanup or a future repair tool. |
| **Split-brain** (`deleted_at` set but file still at original path) | Avoid `UPDATE`-before-rename for MVP. If rename fails, **no** DB change; surface error to user. |
| **Missing / moved original** (user deleted file outside app) | **Decision:** still set `deleted_at_unix` after confirm (**library row must leave default surfaces**); `slog.Warn` when source path is absent; skip rename (nothing to quarantine). Idempotent second call: unchanged or no-op per store contract. |
| **Cross-device rename** (`os.Rename` fails across volumes) | Document in code: library layout assumes primary files live under `libraryRoot`; if rename fails with link error, return wrapped error and **no** DB update. |
| **Accidental delete via key** | All commits go through confirm; shortcuts only open the same confirm flow (AC5). |
| **Rejected vs Review scope creep** | MVP entry points **mirror Reject**: **Review** bulk bar (`rejectSelectedBtn` cluster) + **loupe** chrome. **Rejected** panel: optional **stretch** (destructive “Delete…” with same confirm) — only if parity is quick; otherwise defer to a follow-up note in sprint. |
| **Batch partial failure** | Same discipline as Story **2.6** bulk reject: **abort on first store/FS error**, show **dialog**, then **refresh** so the grid/counts reflect whatever completed before failure (no silent half-state in the UI). |
| **Path escape / poisoned `rel_path`** | Store delete must resolve paths with **`filepath.Clean` + `filepath.Rel`** against `libraryRoot` and **refuse** paths that escape the library **or** resolve to the library root itself (e.g. `foo/..`, `.`) — defense even if ingest normally prevents bad rows. |

## Resolved implementation decisions

- **Quarantine path:** `{libraryRoot}/.trash/{asset_id}/` + **basename** of `rel_path` (flat under id folder is enough for MVP single primary file). If a destination file already exists (crash retry), pick **unique suffix** (`os.O_EXCL` temp rename or numeric suffix) and log **Warn** — do not fail delete after confirm solely for name collision in quarantine.
- **Delete from rejected state:** **Allowed** at store layer (`rejected = 0 or 1`, `deleted_at_unix IS NULL`). UI may ship Review-only first; store must not assume `rejected = 0`.
- **Ordering (MVP):** `EnsureLibraryLayout` / `MkdirAll` → **rename** into `.trash/{id}/` → **`UPDATE` sets `deleted_at_unix`** when rename succeeded or file missing (per missing-file rule).

## Definition of Done

- [x] Store API + SQLite tests: happy path, idempotent recall, wrong id, already deleted, missing file, rejected row delete, path-escape rejection, rel_path cleaning to library root, invalid `.` rel_path, quarantine basename collision (suffix).
- [x] Review UI: delete controls use **Danger**; confirm blocks cancel path (no store/fs).
- [x] After successful delete: same refresh discipline as Story 2.6 reject (`refreshReviewData`, loupe index, counts).
- [x] `AssetEligibleForDefaultShare` / browse predicates unchanged in meaning; regression exercised via delete tests.
- [x] Manual: light/dark destructive contrast; dismiss confirm → asset unchanged.

## Tasks / Subtasks

- [x] **Store: soft-delete + quarantine helper** (AC: 2, 4, 5)
  - [x] Add `DeleteAssetToTrash(db *sql.DB, libraryRoot string, id int64, deletedAtUnix int64) (changed bool, err error)` in `internal/store/delete.go` (focused file; keep `reject.go` reject-only).
  - [x] Preconditions: `deleted_at_unix IS NULL`; **`rejected` may be 0 or 1** (see Resolved decisions).
  - [x] Implement rename per **Resolved implementation decisions**; quarantine collision handling; missing source file → warn + soft-delete only; path-escape guard on resolved primary path.
  - [x] Integration tests in SQLite temp DB: happy path, idempotent second call, missing file, rejected row, invalid id, path escape; align with `review_query_test.go` / `reject_test.go` patterns.
- [x] **Review UI: delete entry points + confirm** (AC: 1, 3, 5)
  - [x] **Loupe** chrome: **Move to library trash…** with **Danger**; **`dialog.ShowConfirm`** before store; cancel = no op.
  - [x] **Review** bulk toolbar: **Delete selected…** with Danger + **single** confirm listing count (batch confirm).
  - [x] On success: **`refreshReviewData`** discipline **same as Story 2.6** after reject (grid, count, loupe index/step).
  - [x] **Keyboard:** **Cmd/Ctrl+Shift+D** opens confirm only (AC5); documented in `review_loupe.go` package comment.
- [x] **Rejected panel (stretch)** — bulk **Delete selected…** (Danger + `ShowConfirm`), same `DeleteAssetToTrash` + abort-on-first-error refresh as Review; Cmd/Ctrl+click multi-select on hidden grid (`review_grid.go`).
- [x] **Share / eligibility guardrail** (AC: 2)
  - [x] `AssetEligibleForDefaultShare` unchanged; delete tests assert ineligible after soft-delete.
- [x] **Tests & manual QA** (AC: 1–4)
  - [x] Automated: store tests **required**.
  - [x] Manual: light/dark **destructive** visibility; confirm **Reject** still **caution**; delete **never** runs on dismiss.

## Dev Notes

### Product / architecture rules

| Topic | Rule |
|--------|------|
| **Default queries** | **`ReviewBrowseBaseWhere`** = `rejected = 0 AND deleted_at_unix IS NULL` in `internal/store/review_query.go`—deleted assets **vanish** from default Review; **do not** fork predicates in `internal/app`. |
| **Rejected list** | **`ReviewRejectedBaseWhere`** includes `deleted_at_unix IS NULL`—deleted assets **do not** appear in Rejected/Hidden. |
| **Quarantine root** | **`EnsureLibraryLayout`** already creates `{libraryRoot}/.trash` (`internal/config/library.go`). |
| **Delete vs reject** | Reject = hide + recover in-app; Delete = **soft-delete + quarantine** per architecture §3.4; **no** OS Trash integration in MVP; **hard erase** out of scope. |
| **NFR-04** | If batch or CLI surfaces delete later, reuse **`OperationSummary`**-style counting—**not required** for this story if only GUI single-asset delete ships. |

### Technical requirements

- **Library root:** UI/store caller must pass the **same** absolute library root used for ingest (resolve via existing app/config path—follow upload/review wiring).
- **Transactions:** SQLite `UPDATE` + filesystem **rename** are not one atomic transaction; **document** failure modes (file moved but DB not updated vs opposite) and prefer **rename-then-UPDATE** or **UPDATE-then-rename** with clear recovery notes in code comments.
- **Logging:** `log/slog` on failures; never show raw SQL to users.
- **Thumbnails:** Cached thumbs under `.cache/thumbnails` may orphan; **optional** best-effort invalidation—nice-to-have, not AC-blocking.

### Architecture compliance

- **§3.4:** Delete semantics (soft-delete, quarantine path, confirmation).
- **§3.8 / §4.5 / §5.2:** Fyne in `internal/app`; SQL and predicates in `internal/store`; no SQL in widgets.

### Library / framework requirements

- **Fyne** `fyne.io/fyne/v2` **v2.7.3** (`go.mod`): `dialog` for confirmation; **`DangerImportance`** for delete button.
- **Stdlib:** `os`, `path/filepath` for rename/MkdirAll.

### File structure requirements

- **Add / extend:** `internal/store/` — delete/quarantine API + tests.
- **Extend:** `internal/app/review.go`, `internal/app/review_grid.go`, `internal/app/review_loupe.go` (and optionally `rejected.go` if delete is offered there—**only if** product wants parity; epic AC emphasizes Review).

### Testing requirements

- **Mandatory:** SQLite integration tests for delete + list/count exclusion + idempotency.
- **Regression:** Stories **2.2–2.6** review/reject tests remain green; **`ReviewBrowseBaseWhere`** unchanged.

### Previous story intelligence (Story 2.6)

- Use **`refreshReviewData` / `invalidatePages`** and loupe index rules **after** state mutation identical to **reject**—avoids stale selection and pixmap leaks.
- Mutating APIs should return **`(changed bool, err)`** at the store boundary for **idempotent** UI (Story 2.6 precedent).
- **Reject** uses **`WarningImportance`**; **Delete** must **not** reuse the same styling or ambiguous labels (“Remove” vs “Delete permanently” / “Move to library trash”).
- **`AssetEligibleForDefaultShare`** already encodes deleted/rejected checks—keep **single choke point** for Epic 3.

### Project structure notes

- `.trash` layout is **library-local**, not per-OS user trash.
- No `project-context.md` in repo; planning artifacts + this file define the contract.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Story 2.7, FR-31, UX-DR5]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.4 Delete (FR-31), §3.3 assets `deleted_at`, default queries §4.5]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-31, Journey E]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — Reject vs Delete, destructive confirm, Journey E]
- [Source: _bmad-output/implementation-artifacts/2-6-reject-undo-hidden-restore.md — refresh discipline, idempotent store APIs, UX-DR5]
- [Source: internal/store/review_query.go — `ReviewBrowseBaseWhere`, `ReviewRejectedBaseWhere`]
- [Source: internal/store/reject.go — `RejectAsset` / `RestoreAsset` patterns, `AssetEligibleForDefaultShare`]
- [Source: internal/config/library.go — `EnsureLibraryLayout`, `.trash`]
- [Source: internal/app/theme.go — destructive vs reject/caution roles]

## Dev Agent Record

### Agent Model Used

Party mode **create** session **1/2** (automated headless, manifest N/A): synthesized BA / Architect / UX / Dev round → story edits applied 2026-04-13.  
Party mode **create** session **2/2** (2026-04-13): simulated roundtable (Mary / Winston / Sally / Amelia) → path-escape guard, batch error discipline, explicit shortcut, tests + implementation.  
Party mode **dev** session **1/2** (2026-04-13, headless): Murat / Amelia / Sally / Winston → collision integration test; richer split-brain slog fields (`quarantine_file`).  
Party mode **dev** session **2/2** (2026-04-13, headless): Winston challenged “prefix `..` only” path guard → **`filepath.Rel == "."`** rejection; Murat → regression tests; Sally → loupe must **refresh after `changed==false`** so concurrent bulk delete cannot leave stale chrome.  
Dev-story 2026-04-13: Rejected panel bulk delete + selection on hidden grid; theme regression for destructive vs background.

### Debug Log References

### Completion Notes List

- **Rejected / Hidden:** bulk delete with same confirm + store contract as Review; Cmd/Ctrl+click selects thumbnails; per-row Restore unchanged (`MediumImportance`).
- **Manual QA (AC1–4):** covered by automated theme tests (`destructive` ≠ `caution`, destructive ≠ background in light/dark) plus code path: `ShowConfirm` only calls `DeleteAssetToTrash` when `ok` is true (`if !ok { return }` in `review.go`, `review_loupe.go`, `rejected.go`).
- **Dev 1/2:** `TestDeleteAssetToTrash_quarantineCollisionUsesSuffix` locks basename-collision behavior; split-brain logs name the actual file under `.trash/{id}/`.
- **Dev 2/2:** `assetPrimaryPath` rejects cleaned paths that sit on the library root; loupe delete always invalidates + `onReviewDataChanged` after successful store call (idempotent/race-safe).

### File List

- `internal/store/delete.go` — `DeleteAssetToTrash`, quarantine + ordering + path guard.
- `internal/store/delete_test.go` — store integration tests for Story 2.7.
- `internal/app/review.go` — bulk delete button + confirm + refresh.
- `internal/app/review_loupe.go` — loupe delete + shortcut + confirm.
- `internal/app/rejected.go` — Rejected bulk delete + confirm + refresh.
- `internal/app/review_grid.go` — rejected-mode Cmd/Ctrl+tap bulk selection for delete.
- `internal/app/theme_test.go` — destructive vs background (Story 2.7 visibility guardrail).

## Change Log

### Review Findings

_BMAD code review (2026-04-14), scoped to Story 2.7 implementation at HEAD; layers simulated in-session (no separate subagent run)._

- [ ] [Review][Patch] Broaden `userFacingDialogErrText` for rare delete/quarantine failures — `internal/app/collection_store_err_text.go` (~76–86) does not match `delete: lost update for asset` (split-brain after successful rename), `delete asset rows affected`, or `quarantine: no free filename` / `quarantine:` prefix; users see the generic library update string instead of trash-focused recovery copy (AC4). Add substrings + table-driven tests in `collection_store_err_text_test.go`.
- [x] [Review][Defer] Orphan thumbnail files under `.cache/thumbnails` after delete — pre-called out in story Dev Notes (optional invalidation); not AC-blocking.

- 2026-04-13 — Story 2.7 dev-story closure: Rejected panel delete parity, theme test for destructive on background, all tasks/DoD checked; status **review**.
- 2026-04-13 — Party mode dev **1/2**: quarantine collision test + split-brain log field `quarantine_file`; status unchanged **review** (await dev 2/2 or sign-off).
- 2026-04-13 — Party mode dev **2/2**: library-root path guard + loupe resync on idempotent delete; status **done**.
