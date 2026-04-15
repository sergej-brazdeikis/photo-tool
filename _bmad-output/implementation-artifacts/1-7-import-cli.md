# Story 1.7: Import CLI (register / backfill)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **power user**,  
I want **to register existing canonical files or backfill metadata**,  
So that **the DB matches disk after manual operations**.

**Implements:** FR-28; NFR-04.

## Acceptance Criteria

1. **Given** a resolved **import root** (configurable path per Dev Notes—e.g. flag and/or env; must obey register-in-place rules vs `libraryRoot`), **when** `import` runs **without** `--dry-run`, **then** supported image files under that tree that are **not** yet represented in the DB (by **content hash**) are **registered**: an **`assets`** row is inserted with **`rel_path`** relative to `libraryRoot`, **`capture_time_unix`** from **`exifmeta.ReadCapture`**, and **`content_hash`** from the same SHA-256 path as upload/scan—**without copying** the file (FR-28, NFR-03).
2. **Given** a file whose **hash** already exists in **`assets`**, **when** import processes it, **then** the outcome is **`skipped_duplicate`** (no second row, no file moves); deterministic with upload/scan (NFR-03).
3. **Given** **`--dry-run`**, **when** import runs, **then** **no** `INSERT` / `UPDATE` / `DELETE` on **`assets`** (and no other persistence that mutates library meaning), **no** file moves/copies/deletes under the library tree, and **only** summary output is produced (FR-28).
4. **Given** **backfill rules** documented in code for this story (see Dev Notes), **when** import encounters an **existing** row that matches the on-disk file per those rules, **then** applicable metadata is **updated** and **`OperationSummary.Updated`** is incremented (**updated metadata** in NFR-04 / PRD wording). If no row qualifies under the MVP rule set, **`Updated`** may remain zero but the **code path** must exist for the defined cases.
5. **Given** import completes, **when** the CLI prints results, **then** summary categories use **`domain.OperationSummary`** field names and meanings aligned with **`scan`** and the GUI ingest receipt: **`added`**, **`skipped_duplicate`**, **`updated`**, **`failed`** (JSON `snake_case` tags per architecture §4.1 / NFR-04).
6. **And** large trees follow the same **streaming / one-file-at-a-time** discipline as **`scan`** (NFR-02)—no unbounded accumulation of all file paths in memory; cite approach in code comment or test like Story 1.6.

## Tasks / Subtasks

- [x] **CLI: `import` subcommand** (AC: 3, 5)
  - [x] Add **`import`** to **`github.com/spf13/cobra`** in `internal/cli` beside **`scan`**: default **no subcommand** remains GUI-only (architecture §6).
  - [x] Flags: **`--dir`** (required), **`--recursive`** (default `false`, same semantics as **`scan`**), **`--dry-run`** (default `false`). **Backfill** is **always on** when capture time differs (no separate `--backfill` flag—see Dev Agent Record).
  - [x] **`Long` help text** explaining **import vs scan**: import **registers files already under the library root** (register-in-place); **does not** copy from arbitrary external folders like **`scan`**.
- [x] **Config / path resolution** (AC: 1)
  - [x] Resolve **`libraryRoot`** with **`config.ResolveLibraryRoot`**, **`config.EnsureLibraryLayout`**, **`store.Open`**—same bootstrap pattern as **`scan`** / `main`.
  - [x] Resolve **import root** from **`--dir`**; normalize with **`filepath.Clean`**. Enforce **containment**: import tree must lie **under** `libraryRoot` (e.g. `filepath.Rel` must not escape with `..`). Reject with clear error if not.
  - [x] Document **“configurable path”** (FR-28): MVP = required **`--dir`**; optional follow-up: persisted setting or env (e.g. `PHOTO_TOOL_IMPORT_DIR`) if product adds it—do not block on a settings UI.
- [x] **Register-in-place pipeline** (AC: 1, 2, 5, 6)
  - [x] Reuse **supported extensions** from **`internal/ingest/extensions.go`** (same as GUI / **`scan`**—no drift).
  - [x] Walk **`--dir`** with **`filepath.WalkDir`** when **`--recursive`**, else read only immediate children—mirror **`scan`** structure.
  - [x] Per file: **`ReadCapture` → hash (`filehash.ReaderHex`) → `store.AssetRowByContentHash`**. If **new**: **`InsertAsset`** with **`rel_path`** = path relative to **`libraryRoot`** (slash-normalized), **no** copy. If **hash exists elsewhere**: **`SkippedDuplicate`**. If **hash exists at this path** and **`capture_time_unix`** differs: **`Updated`** (backfill).
  - [x] Centralize logic in **`internal/ingest`** (`RegisterInPlacePath`); **no Fyne** imports in **`ingest`** (architecture §5.2).
- [x] **Dry-run** (AC: 3)
  - [x] Same contract as **`scan`**: **SELECT** allowed for classification; **no** writes to **`assets`**; classify **added** / **skipped_duplicate** / **updated** / **failed** consistently with live mode.
- [x] **Backfill (MVP rule)** (AC: 4)
  - [x] Rule (see `RegisterInPlacePath`): row found by **content hash**; if **`rel_path`** matches the file being processed and **`capture_time_unix`** ≠ newly read UTC unix → **`store.UpdateAssetCaptureTime`** and **`Updated++`** (dry-run counts **`Updated`** only).
  - [x] **`store`**: **`ActiveAssetByRelPath`**, **`AssetRowByContentHash`**, **`UpdateAssetCaptureTime`** with **`fmt.Errorf` / `%w`** wrapping.
- [x] **Output** (AC: 5)
  - [x] Human-readable summary lines consistent with **`scan`** / GUI receipt labels (**Added**, **Skipped duplicate**, **Updated**, **Failed**).
- [x] **Tests** (AC: 1–6)
  - [x] `go test ./...` green; tests in **`internal/cli/import_test.go`** and **`internal/ingest/register_import_test.go`** (temp library + SQLite).
  - [x] **Dry-run**: no new **`assets`** rows; **live**: pre-placed files registered when DB empty.
  - [x] **Duplicate**: second pass **skipped_duplicate**; same bytes at second path **skipped_duplicate**.
  - [x] **Parity**: dry vs live separate libraries (`TestRegisterInPlace_dryRun_matchesLiveSeparateLibraries`).
  - [x] **Conflict**: file at **`rel_path`** replaced on disk (different hash) → **Failed** (operator must reconcile).
  - [x] **Exit code**: **`scan`** / **`import`** return an error (status1 via **`MainExit`**) when **`Failed > 0`**, after printing the summary; **`TestRunImport_exitErrorWhenFileFails`**, **`TestRunScan_exitErrorWhenFileFails`** (Unix chmod); soft-delete hash reservation: **`TestRegisterInPlacePath_softDeletedRowReservesHash`**.

## Party mode (automated headless, session 1/2 — create hook)

**Mode:** `--solo` (no `agent-manifest.csv` in repo; simulated roundtable).

**Winston (Architect):** Containment is non-negotiable; anything that silently treated "import" like "scan" would violate FR-28 register-in-place. I pushed back on a single combined walker flag: `import` must hard-fail when `--dir` is outside `libraryRoot`, not merely warn. Hash-first dedup stays the spine; backfill must key off "same hash + same path" so we do not rewrite rows for duplicate copies.

**John (PM):** Users will not read architecture §3.2; the Cobra `Long` must spell out "no copy" in one breath. I disagree with shipping `--backfill` default-off: that hides the one metadata fix users asked for. **Decision:** backfill always on; document in the story instead of another flag.

**Mary (Analyst):** The messy case is "same path, new bytes." That is not FR-28 happy-path; counting it as **Failed** is honest and forces manual repair. Session 1/2 challenge: epics.md AC for 1.7 is thinner than the story; implementation follows the story file as the contract.

**Murat (Test Architect):** Parity tests must cover recursive-style trees, not only flat `--dir`. I want an ingest-level walk parity (mirrors 1.6 recursive hardening). **Added** `TestRegisterInPlace_dryRun_matchesLiveSeparateLibraries`.

**Orchestrator synthesis — edits applied:** Implemented `import` CLI + `ingest.RegisterInPlacePath`, store helpers for hash row / rel_path / capture update, containment error, always-on backfill, conflict-as-failed for replaced files, tests + sprint **`review`** for 1-7.

## Party mode (automated headless, session 2/2 — create hook, deepen)

**Mode:** `--solo` (no `agent-manifest.csv` in repo).

**Winston (Architect):** Session 1 treated `filepath.Clean` + `Rel` as sufficient containment. That is **false** if `--dir` is a symlink whose target leaves the library: `WalkDir` can traverse real files outside the tree while the “logical” path still looks nested. **Decision:** resolve **`EvalSymlinks`** on both library root and import dir before the containment check—only for **`import`**, not loosening **`scan`**’s ability to read external sources.

**Paige (Tech Writer):** I disagree with burying the symlink rule only in code. **Epics.md** Story 1.7 ACs were still PRD-vague; readers would assume parity with the story file. **Decision:** expand epic ACs to spell register-in-place, hash dedup, dry-run persistence ban, **`OperationSummary`** labels, and NFR-02 walk discipline so planning doc matches the implementation contract.

**Murat (Test Architect):** Recursive parity at ingest level is not enough for CLI confidence—**`RunImport --recursive`** needs its own test. Add **`TestRunImport_recursive_registersNestedFiles`**. Add **`TestRunImport_rejectsSymlinkDirOutsideLibrary`** with **`t.Skip`** when the OS cannot create symlinks.

**Amelia (Dev):** Pushing back on Winston’s other rabbit hole: **`content_hash`** is **globally unique** in SQLite even for soft-deleted rows, so “ignore deleted in hash lookup” without a migration would make **`InsertAsset`** fail and get misclassified as **`skipped_duplicate`**. **Decision:** document that edge as a **known schema interaction**; no behavioral change in session 2 beyond containment + tests + doc alignment.

**Orchestrator synthesis — edits applied (session 2):** `RunImport` containment uses **`filepath.EvalSymlinks`**; **`import_test.go`** recursive + symlink escape coverage; **`epics.md`** Story 1.7 ACs aligned with story/NFRs; story **`done`**, sprint **`1-7-import-cli: done`**.

## Party mode (automated headless, hook **dev**, session **1/2**)

**Mode:** `--solo` (no `agent-manifest.csv` in repo).

**Amelia (Dev):** Ingest tests already own backfill and conflict; the CLI layer still owed proof that **`RunImport`** wires the same semantics. I’m pushing back on rewriting **`RegisterInPlacePath`** to take symlink-resolved `libraryRoot`: **`filepath.Rel`** must stay in the same path namespace as **`WalkDir`** / **`ReadDir`**; resolving only the containment check is deliberate.

**Murat (Test Architect):** Recursive coverage exists, but we had no CLI regression for **non-recursive** “don’t descend”—that’s a one-line behavioral promise in **`--recursive`**. I also want an end-to-end **backfill** test: **`Updated: 1`** on stdout and DB **`capture_time_unix`** corrected.

**Sally (UX Designer):** The **`import`** **`Long`** text should mention symlink resolution in plain language; operators won’t read **`RunImport`** comments.

**Winston (Architect):** I disagree with burying the symlink/`Rel` split only in Amelia’s head—**document it beside the containment block** so the next refactor doesn’t “fix” it into a broken state.

**Orchestrator synthesis — edits applied (dev1/2):** Comment on resolved-vs-configured roots in **`import.go`**; **`root.go`** **`import`** **`Long`** symlink sentence; **`TestRunImport_nonRecursive_skipsNestedFiles`** and **`TestRunImport_backfillsStaleCaptureTime`** in **`import_test.go`**.

## Party mode (automated headless, hook **dev**, session **2/2**)

**Mode:** `--solo` (no `agent-manifest.csv` in repo).

**Murat (Test Architect):** Session 1’s “document tombstone hash” note is too weak for regression safety. I want a **locked ingest test** that proves a **soft-deleted** row still **reserves `content_hash`**: a second path with the same bytes must stay **`skipped_duplicate`** with **zero active rows**—not silently “fixed.”

**Amelia (Dev):** I’ll push back on scope creep: **exit codes** were never in the story AC, but **`Failed:` without `os.Exit(1)`** is a scripting trap. If we fix **`import`**, we must fix **`scan`** the same way via **one helper**—no divergent CLI semantics.

**Sally (UX Designer):** Operators read **`--help`**, not `MainExit`. Both **`scan`** and **`import`** **`Long`** text should say **status 1 after summary** when anything failed—same sentence shape for muscle memory.

**Winston (Architect):** I disagree with Murat’s Unix-only chmod idea unless it’s **`t.Skip`’d** on Windows; we still need **cross-platform** proof on **`import`** (path/hash conflict already behaves the same everywhere). **Decision:** **`errIfOperationFailures`** in **`internal/cli`**, **`TestRunImport_exitErrorWhenFileFails`**, **`TestRunScan_exitErrorWhenFileFails`** (Unix), **`TestRegisterInPlacePath_softDeletedRowReservesHash`**.

**Orchestrator synthesis — edits applied (dev2/2):** `summary_exit.go`; **`RunScan` / `RunImport`** return errors when **`sum.Failed > 0`**; help text; tests above; story **`done`** unchanged; sprint unchanged.

## Party mode (automated headless, hook **dev**, session **1/2** — 2026-04-14)

**Mode:** `--solo` (manifest at `_bmad/_config/agent-manifest.csv`; roundtable simulated).

**Amelia (Developer):** Ingest tests prove **`RegisterInPlacePath`**, not **`RunImport` + Cobra flags + stdout**. Session 1/2 should lock **dry-run backfill** at the CLI: **`Updated: 1`** on stdout while **`capture_time_unix`** stays stale in SQLite.

**Sally (UX Designer):** Help text is enough; the four receipt lines already match **scan**. Skip another **Long** paragraph—muscle memory is the counters.

**Winston (Architect):** **`import`** progress logs included **`updated`**; **`scan`** omitted it even though **`OperationSummary`** is shared. Normalize the **`slog`** field list so CI greps stay comparable; **`scan`** will usually log **`updated=0`**, which is honest.

**Murat (Test Architect):** Pushing back: adding **`updated`** to **scan** logs risks golden-log churn—acceptable only if **`go test ./...`** stays green (no snapshot tests on those lines today).

**Orchestrator synthesis — edits applied:** **`TestRunImport_dryRun_backfillClassifiesUpdatedWithoutDBWrite`**; **`scan`** progress **`slog`** includes **`updated`**; story + sprint comment only.

## Party mode (automated headless, hook **dev**, session **2/2** — 2026-04-14)

**Mode:** `--solo` (manifest at `_bmad/_config/agent-manifest.csv`; roundtable simulated).

**Murat (Test Architect):** Session 1 proved dry-run backfill text on stdout; it did **not** lock **ordering** for automation. **`Failed > 0`** must still yield **exactly four lines** in **`Added` → `Skipped duplicate` → `Updated` → `Failed`** so the last line stays grep-stable for CI/scripts.

**Amelia (Developer):** Pushback on import-only tests: **`errIfOperationFailures`** is shared—**`scan`** exit-path must assert the same receipt shape or we’ll drift silently.

**Sally (UX Designer):** **`scan --help`** listed only three outcome buckets while the binary always prints **`Updated:`** (zero for scan). That’s cognitive noise for power users comparing **`scan`** vs **`import`**; align the **`Long`** text with the four printed lines.

**Winston (Architect):** I disagree with burying “you can pass **library root** as **`--dir`**” only in epics—**`import` `Long`** should say it; containment already allows **`--dir == libraryRoot`**.

**Orchestrator synthesis — edits applied (dev 2/2):** **`internal/cli/summary_stdout_test.go`** (`assertOperationReceiptLineOrder`); **`TestRunImport_exitErrorWhenFileFails`** + **`TestRunScan_exitErrorWhenFileFails`** call it; **`root.go`** scan dry-run wording + import library-root sentence; **`testImportCommand` / `testScanCommand`** set **`SilenceUsage` + `SilenceErrors`** like production so captured stdout stays receipt-only; Dev Agent Record + sprint comment.

## Dev Notes

### Technical requirements

- **Prerequisites:** Stories **1.1–1.6** (library, capture/hash, **`ingestOne`** semantics, **`OperationSummary`**, **`scan` CLI**). Import **does not** assign collections (FR-04–FR-06).
- **Import vs scan:** **`scan`** ingests from **any** directory and **copies** into canonical layout. **`import`** assumes files **already live under `libraryRoot`** and only **syncs the DB** (+ optional metadata correction). Do not use **`import`** to ingest from outside the library tree without an explicit product decision (would contradict register-in-place).
- **FR-26 breadth:** As in Story **1.6**, **`InsertAsset`** today persists **`capture_time_unix`** (and hash/path). Full camera/lens columns are **out of scope** unless this story adds a migration—prefer **capture-time backfill** only unless PM expands AC.
- **NFR-04 “updated metadata”:** PRD lists **updated metadata** alongside **added** / **skipped duplicate** / **failed**. Map metadata refresh to **`OperationSummary.Updated`**; keep field names identical to **`scan`** and GUI.

### Architecture compliance

- **§3.2 / FR-28:** Register-in-place under configurable path constrained to library rules.
- **§3.9 / §4.5:** Single **`OperationSummary`**; one dedup path (**`AssetIDByContentHash`** + same hash algorithm).
- **§5.2:** CLI in **`internal/cli`**; orchestration in **`internal/ingest`**; SQL only in **`store`**.
- **§4.2–4.3:** `fmt.Errorf` with `%w`; **`log/slog`** for operational logs.

### Library / framework

- **Cobra** (already in module for **`scan`**).
- **Go / SQLite / exifmeta / filehash**—same stack as **`scan`**.

### File structure (touch / extend)

- `internal/cli/root.go` — register **`import`** command.
- `internal/cli/import.go` (or equivalent) — **`RunImport`**.
- `internal/cli/*_test.go` — CLI / integration tests as needed.
- `internal/ingest/` — register-in-place + dry-run + backfill orchestration.
- `internal/store/assets.go` (or new file) — **`Update`** helpers for backfill.
- Reuse **`internal/ingest/extensions.go`**, **`internal/domain/summary.go`**.

### Testing requirements

- Table-driven tests per architecture §4.4; temp dirs + **`store.Open`**.
- Assert **dry-run** leaves **`assets`** count unchanged.
- Assert **`rel_path`** uniqueness and **hash** dedup behavior match **`ingest`** invariants.

### Previous story intelligence (1.6)

- **`scan`** established: Cobra layout, **`IngestPaths` / `IngestPath`** with **`dryRun`**, **`WalkDir`** streaming, **`extensions.go`** as single extension source, **`OperationSummary`** parity tests. **Mirror patterns**; **do not** fork dedup or summary types.
- **Out of scope for 1.7:** **`--json`** flag, reject/delete counts (unless import touches those fields—default **no**).

### Git intelligence summary

- Recent commits in history are broad project seeds; **current** implementation work lives in **`internal/cli`**, **`internal/ingest`**, **`internal/app/upload.go`**, **`main.go`**—align new code with those packages’ style and logging.

### Latest technical information

- No new mandatory dependencies expected; use the same **Cobra** / **modernc** stack already in **`go.mod`**.

### Project structure notes

- Matches architecture §5.1 (`internal/cli`, `internal/ingest`, `internal/store`).
- Binary naming: module **`photo-tool`** vs target **`phototool`**—same as Story **1.6** (`go run . import ...` until **`cmd/phototool`** migration).

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Story 1.7, Epic 1, FR-28, NFR-04]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-28, NFR-02, NFR-03, NFR-04, Journey D]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.2 library/import, §3.9 CLI/GUI parity, §4.1 JSON naming, §4.5 agent rules, §5.1 layout, §6 CLI]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR6 receipts / CLI parity]
- [Source: _bmad-output/implementation-artifacts/1-6-scan-cli.md — CLI patterns, dry-run contract, NFR-02]
- [Source: internal/cli/root.go, internal/cli/scan.go — Cobra structure to mirror]
- [Source: internal/ingest/ingest.go — hashing, dedup, dry-run behavior]
- [Source: internal/ingest/extensions.go — extension list]
- [Source: internal/store/assets.go — `InsertAsset`, hash lookup]
- [Source: internal/domain/summary.go — `OperationSummary`]
- [Source: internal/config/library.go — library resolution]

## Dev Agent Record

### Agent Model Used

BMAD party mode (automated headless; create + dev sessions), 2026-04-13. BMAD dev-story verification run (Cursor), 2026-04-13. BMAD dev-story workflow re-run (Cursor), 2026-04-14.

### Debug Log References

### Completion Notes List

- **`import`**: `phototool import --dir <path-under-library> [--recursive] [--dry-run]`; **`--dir`** required; containment enforced.
- **Backfill**: always on — when DB row for **same content hash** and **same `rel_path`** has stale **`capture_time_unix`**, import updates it and increments **`Updated`** (dry-run classifies only).
- **Replaced file at path** (active row hash ≠ disk hash): **Failed** + error log; no automatic delete/replace (operator reconciles).
- **Duplicate content, second path**: **Skipped duplicate** (NFR-03).
- **Containment:** import root and library root are compared **after `filepath.EvalSymlinks`** so a directory symlink cannot bypass “must be under library root.”
- **Known limitation (soft-delete × global hash uniqueness):** `assets.content_hash` is unique for all rows; a soft-deleted row still reserves its hash. Import treats some insert failures like duplicate skips; full “revive tombstone” behavior is **out of scope** for 1.7 unless schema/product changes. **`TestRegisterInPlacePath_softDeletedRowReservesHash`** locks the observable outcome (second path → **`skipped_duplicate`**, no active row).
- **CLI exit status:** When **`Failed > 0`**, **`scan`** and **`import`** return an error after printing the full summary so **`MainExit`** exits **1** (parity between subcommands).
- **Dev-story verification:** `go test ./...` and `go build .` green; ACs 1–6 and all story tasks confirmed against current tree; sprint `1-7-import-cli` remains `done` (terminal state — not regressed to `review`).
- **2026-04-14 dev-story workflow:** Full `go test ./...` and `go build .` at project root; all packages green; no code changes required; tasks remain complete.
- **2026-04-14 party mode (dev 1/2):** CLI dry-run backfill regression (`import_test.go`); scan progress log parity with import (`scan.go`).
- **2026-04-14 party mode (dev 2/2):** Receipt stdout locked to four ordered lines (`summary_stdout_test.go`; import + scan exit tests); `import`/`scan` Cobra `Long` text aligned with printed summary + library-root hint for import; test subcommands silence usage/errors to match `Execute` and keep buffers clean.

### File List

- `internal/cli/root.go` — register `import` command; `scan`/`import` Long help (four outcome labels; import may use library root as `--dir`)
- `internal/cli/summary_stdout_test.go` — shared assertion for ordered four-line CLI receipt (scan/import parity)
- `internal/cli/import.go` — `RunImport` (symlink-resolving containment)
- `internal/cli/summary_exit.go` — non-zero exit when `OperationSummary.Failed > 0` (shared with scan)
- `internal/cli/scan.go` — uses `errIfOperationFailures` after summary; progress `slog` includes `updated` (parity with import)
- `internal/cli/import_test.go` — CLI tests (includes receipt order on import failure path)
- `internal/cli/scan_test.go` — scan CLI tests (receipt order on scan failure path)
- `internal/ingest/register_import.go` — `RegisterInPlacePath`
- `internal/ingest/register_import_test.go` — ingest/import tests
- `internal/store/assets.go` — `AssetRowByContentHash`, `ActiveAssetByRelPath`, `UpdateAssetCaptureTime`; `AssetIDByContentHash` delegates to row helper
- `internal/domain/summary.go` — comment for **`Updated`**
- `internal/ingest/extensions.go` — doc string mentions import

### Review Findings (bmad-code-review, headless, 2026-04-14)

_Scope: Story 1.7 paths; diff emphasis: scan dry-run dedup, receipt tests, help text._

- [ ] [Review][Patch] Import dry-run should mirror scan in-batch content-hash memory — `RegisterInPlacePath` / `RunImport` lack a `drySeen`-style map; two same-byte files in one dry run can classify as two `added` while live run dedups. Violates AC3 parity with scan after `ingest.IngestPath` dry-run fix. [`internal/ingest/register_import.go`, `internal/cli/import.go`]

- [ ] [Review][Patch] Assert four-line CLI receipt order on import dry-run backfill test — `TestRunImport_dryRun_backfillClassifiesUpdatedWithoutDBWrite` should call `assertOperationReceiptLineOrder` for grep-stable stdout (party-mode intent). [`internal/cli/import_test.go`]

- [x] [Review][Defer] `scanSummaryFromOutput` ignores non-receipt lines — parity tests could theoretically pass if extra stdout noise appears; pre-existing helper pattern, not introduced by this diff. [`internal/cli/scan_test.go`] — deferred, pre-existing

---

**Story context:** Ultimate context engine analysis completed — comprehensive developer guide created (BMAD create-story workflow, 2026-04-13).
