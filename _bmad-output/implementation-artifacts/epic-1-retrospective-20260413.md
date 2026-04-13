# Epic 1 retrospective — photo-tool

**Epic:** Ingest photos into a trustworthy local library  
**Mode:** Headless synthesis from `sprint-status.yaml`, `epics.md` (Epic 1), story artifacts1.2–1.5, and `deferred-work.md`. No live facilitation.  
**Date:** 2026-04-13  
**Stance:** Psychological safety; systems, processes, and learning—not individuals.

---

## Epic status (from tracking)

Per `_bmad-output/implementation-artifacts/sprint-status.yaml`:

| Key | Status |
|-----|--------|
| `epic-1` | in-progress |
| `1-1-library-foundation` | done |
| `1-2-capture-time-hash` | in-progress |
| `1-3-core-ingest` | in-progress |
| `1-4-collections-schema` | in-progress |
| `1-5-upload-confirm-receipt` | in-progress |
| `epic-1-retrospective` | done |
| `epic-2` | backlog |

**Interpretation:** This is a **checkpoint retrospective**. Substantial implementation exists across ingest, store, and upload (per story dev records), but sprint keys remain **in-progress** until review patches close and “done” is applied consistently. The `epic-1-retrospective: done` entry reflects documentation/ritual completion, not closure of every Epic 1 story.

**Planning vs tracking:** `epics.md` defines **eight** Epic 1 stories (1.1–1.8: scan CLI, import CLI, drag-and-drop). `sprint-status.yaml` only lists **five** development keys. Unmapped 1.6–1.8 are a **visibility risk**: work can ship “informally” or stall without appearing on the board.

---

## What went well

1. **Foundation delivered** — Story 1.1 is **done**; library layout, SQLite, migrations, and `assets` uniqueness align with architecture.

2. **Clear vertical slices in code** — `internal/exifmeta`, `internal/filehash`, `internal/ingest`, `internal/domain` (`OperationSummary`), and `internal/store` stay separated from Fyne, supporting the intended single pipeline for GUI and future CLI (NFR-04).

3. **Ingest orchestration is explicit** — Hash-before-copy, dedup, cleanup on failure, and streaming hash usage are documented in package comments and story1.3 notes; integration tests were added for duplicate batches and summary fields (addressing an earlier “tests missing” concern).

4. **Migrations became forward-capable** — Story 1.4 introduced a versioned migration chain and `002_collections.sql`, unblocking collections and junction tables needed for FR-18 / FR-20 and downstream FR-04–FR-06.

5. **Upload flow matches Journey A intent** — Story 1.5 records the required ordering: ingest and assets first, **collections and links only after confirm**, with receipts tied to `OperationSummary` and `IngestWithAssetIDs` / `AssetIDByContentHash` for deterministic ID resolution (including duplicates in a batch).

6. **Deferrals are written down** — EXIF limitations, `go-exif` memory behavior, filename collision theory, and `copyToFile` sync are captured in story reviews and `deferred-work.md`, which supports honest backlog grooming instead of silent “MVP means never track it.”

---

## What to improve

1. **One definition of “done” per story** — Align `sprint-status.yaml`, story file `Status`, task checklists, and open **Review** items. Leaving `[Review][Patch]` boxes checked off only when fixed—or explicitly re-filed—prevents “in-progress forever” and false confidence.

2. **Fragile SQLite error handling** — Multiple reviews flag **English substring** matching on driver errors for uniqueness and late-duplicate paths. Prefer **typed** constraint detection (`errors.As` / result codes) so behavior survives driver or message changes (stories 1.3, 1.5, `deferred-work.md`).

3. **Collections link semantics vs idempotency** — Story 1.4 review notes `INSERT OR IGNORE` can swallow **invalid FK** cases without surfacing errors. That conflicts with AC2 and with upload flows that need trustworthy failures. Fix the SQL pattern **before** treating 1.4 as closed.

4. **Upload state machine hardening** — Story 1.5 review identifies **orphan collections** when assign succeeds but nothing is linkable, **re-entrant Import** before confirm (receipt/default date drift), and **non-atomic** create+link. These are **product-trust** issues for FR-06 / UX-DR6, not polish.

5. **Test coverage vs AC wording** — Ingest tests should include at least one **EXIF-backed** path (not only mtime fallback) so `CanonicalDayDir` / `rel_path` behavior matches Story 1.2 +1.3 AC literally.

6. **Board completeness for Epic 1** — Add `1-6` … `1-8` (or equivalent) to sprint tracking, **or** record a deliberate scope decision that 1.6–1.8 are deferred and reflect that in `epics.md` / sprint notes. Hidden stories distort Epic 2 readiness.

---

## Risks for the next epic (Epic 2: Review, filter, organize, and curate)

1. **`OperationSummary` and logging contract drift** — Epic 2 will encode receipts and filters against the same categories. Any late rename or semantic shift in Epic 1 CLI/UI work propagates into wrong assumptions in grid, loupe, and QA.

2. **Collections integrity leaks into UX** — Epic 2’s collection list, detail, and CRUD assume **clean** create/link/delete semantics. Orphan collections, silent link failures, or ambiguous error paths from Epic 1 will surface as “the app lost my album” bugs.

3. **Capture-time semantics meet the grid** — Deferred EXIF items (offset tags, sub-second, swallowed parse errors) may be tolerable in ingest logs but **visible** once Review sorts and groups by capture time (FR-22–FR-24). Expect follow-up hardening or explicit “known limitations” in UX copy.

4. **NFR-01 / NFR-07 concentrate in Epic 2** — Layout and OS scaling validation land there. Unfinished upload or shell ergonomics from Epic 1 should not consume Epic 2’s QA budget by default.

5. **False “Epic 1 complete”** — If 1.6–1.8 stay off the board, the team might start Epic 2 while scan/import/DnD parity (FR-27, FR-28, UX-DR14) is still ambiguous—hurting NFR-02 / NFR-04 claims for power users.

---

## Action items (concrete)

1. **Reconcile sprint keys with `epics.md` Epic 1** — **Owner:** Project lead (Sergej Brazdeikis). **Success:** Every story 1.1–1.8 either has a `development_status` key or an explicit deferral note in sprint artifacts; no “phantom” scope.

2. **Replace substring-based SQLite uniqueness detection with typed constraint handling** — **Owner:** Developer. **Success:** `isUniqueContentHash` (and similar) uses driver-aware checks; unit/integration tests still pass; story review items closed or re-filed.

3. **Fix `LinkAssetsToCollection` idempotency without hiding bad FKs** — **Owner:** Developer. **Success:** Duplicate pairs remain safe; invalid `asset_id` / `collection_id` return errors; `go test ./internal/store/...` updated; story 1.4 review patches cleared.

4. **Close Story 1.5 review patches (orphan collection, Import re-entry, create+link atomicity)** — **Owner:** Developer + short UX pass. **Success:** Confirm/cancel flows cannot create empty collections mistakenly; Import/disabled state matches Journey A; transactional or compensating behavior documented.

5. **Publish Epic 2 technical readiness gate** — **Owner:** Architect + PO. **Success:** Short checklist (ingest API stable, `OperationSummary` frozen for MVP, schema v2 applied, minimal upload receipt path verified) is agreed **before** `2-1` or equivalent is marked in-progress.

---

## Continuity

**Epic 1** is the first epic in this artifact set; there is **no prior epic retrospective** to score follow-through against.

---

## Significant discoveries (systems level)

- **Process:** Sprint file story count ≠ `epics.md` Epic 1 story count. Treat as a **planning system** fix so readiness for Epic 2 is not judged on a partial board.

- **Technical:** Several high-value fixes cluster around **SQLite error semantics** and **transaction boundaries** at the collections boundary—addressing them now reduces expensive rework when Epic 2 multiplies call sites.

---

## Note on re-running

When all Epic 1 stories in `epics.md` that the team commits to are **done** in `sprint-status.yaml`, consider a **short follow-up retro** to validate whether checkpoint findings held and whether 1.6–1.8 changed the narrative for CLI/DnD users.

---

_End of retrospective._
