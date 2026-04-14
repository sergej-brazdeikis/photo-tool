# Story 1.6: Scan CLI (`--dir`, `--recursive`, `--dry-run`)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **power user**,  
I want **to scan a folder tree into the canonical library with optional dry-run**,  
So that **I can reconcile large archives safely**.

**Implements:** FR-27; NFR-02, NFR-04.

## Acceptance Criteria

1. **Given** `phototool scan --dir PATH` (exact CLI shape per implementation; see Dev Notes for cobra layout), **when** run **without** dry-run, **then** supported images are discovered **recursively when `--recursive` is set**, hashed, deduped per Story 1.3, copied if new, and the database is updated (FR-27).
2. **Given** `--dry-run` is enabled (boolean flag; align with cobra conventions, e.g. default `false`), **when** scan runs, **then** **no** files are copied and **no** database **writes** occur (no `INSERT`/`UPDATE`/`DELETE` for assets), but **`OperationSummary`** counts are still computed and emitted as if the run completed—using **read-only** dedup checks (`content_hash` lookup) to classify **added** vs **skipped_duplicate** vs **failed** (FR-27).
3. **Given** scan completes, **when** output is printed, **then** summary categories use **`domain.OperationSummary`** field names and meanings matching the GUI ingest receipt (`added`, `skipped_duplicate`, `updated`, `failed`; JSON `snake_case` tags per NFR-04 / architecture §4.1).
4. **And** a **10,000-file** dry-run does not grow memory unbounded (NFR-02)—verified by a **streaming directory walk** (e.g. `filepath.WalkDir` or equivalent) that processes **one file at a time** without accumulating all paths in memory, plus a **test or benchmark comment** in code documenting the approach.

## Tasks / Subtasks

- [x] **CLI bootstrap** (AC: 1–3)
  - [x] Add **`github.com/spf13/cobra`** root command per [architecture.md §6](_bmad-output/planning-artifacts/architecture.md): default **no subcommand** → launch existing Fyne UI (`main.go` behavior today); **`scan`** subcommand runs headless (no Fyne init).
  - [x] Resolve library root with `config.ResolveLibraryRoot`, `config.EnsureLibraryLayout`, `store.Open` — mirror GUI bootstrap; exit non-zero on failure with `slog` + `os.Exit` pattern consistent with current `main.go`.
- [x] **`scan` command flags** (AC: 1–2)
  - [x] `--dir` (required): absolute or relative path; `filepath.Clean` / validate directory exists and is readable.
  - [x] `--recursive` (default **false** per epic “when flag set”): if false, only immediate files in `--dir`; if true, walk subdirectories.
  - [x] `--dry-run` (default false): force read-only mode for filesystem copies and DB mutations (see ingest extension below).
- [x] **Discovery + ingest** (AC: 1–2,4)
  - [x] Restrict candidates to the **same extension set** as GUI upload via **`internal/ingest/extensions.go`** (single source: common raster types plus `.heic` / `.dng` where the pipeline is enabled; GUI picker uses `PickerFilterExtensions()` so CLI and GUI cannot drift).
  - [x] For each file: reuse **one** pipeline with upload/scan/import: **`exifmeta.ReadCapture` → hash (`filehash.ReaderHex`) → dedup via `store.AssetIDByContentHash` / existing ingest helpers** — **do not** duplicate dedup logic (architecture §4.5, NFR-03).
  - [x] Implement **`dry-run` branch** inside `internal/ingest` (or a thin `scan` orchestrator that calls package-private helpers): perform capture + hash + existence check; increment **`Added`** if hash absent, **`SkippedDuplicate`** if present; **never** `copyToFile` / `InsertAsset` when dry-run. Live run delegates to existing **`ingest.Ingest`-equivalent** path (single file or batch API—refactor as needed so CLI does not copy-paste `ingestOne`).
- [x] **Output** (AC: 3)
  - [x] Print human-readable summary lines (labels aligned with GUI receipt: **Added**, **Skipped duplicate**, **Updated**, **Failed**). If **`Updated` is always zero** for scan, still print `0` or omit per product choice—**document** and stay consistent with future `--json` (architecture §3.9).
- [x] **NFR-02 evidence** (AC: 4)
  - [x] Add **integration test** and/or **benchmark** that walks a large temp tree (or uses a loop to simulate 10k entries) proving **no slice-of-all-paths** pattern; comment in test or `ingest`/`scan` package referencing NFR-02.
- [x] **Tests / regression**
  - [x] `go test ./...` green; **no Fyne imports** under `internal/ingest` / new CLI package (architecture §5.2).
  - [x] Tests: dry-run produces **identical counts** to a live run when DB starts empty and paths are unique (**flat** and **`--recursive`** trees on **separate** empty libraries); second live run **skips duplicates**; dry-run after live run shows **all skipped_duplicate** for same tree.

## Dev Notes

### Technical requirements

- **Prerequisites:** Story 1.1 (library + store), 1.2 (capture + hash), 1.3 (`ingestOne` semantics, `OperationSummary`). Scan does **not** create collections (FR-04–FR-06)—only assets + files.
- **FR-27 EXIF breadth:** PRD lists minimum EXIF fields (capture time, camera, lens). **Today** `InsertAsset` persists **`capture_time_unix`** (and hash/path) only—camera/lens persistence is **FR-26 / Epic 2** breadth. Scan **must** use **`ReadCapture`** (and thus capture time) for placement; **do not** block Story 1.6 on new DB columns for camera/lens unless you explicitly split a migration task (out of scope unless PM expands AC).
- **Dry-run vs reads:** SQLite **SELECT** for dedup during dry-run is allowed; **no** writes means no new/changed `assets` rows and no copied files under the library tree.
- **Paths under library:** If `--dir` points **inside** the managed library root, scan may re-hash canonical files and count **skipped_duplicate**—acceptable; optional **WARN** log if `--dir` is contained in `libraryRoot` to reduce operator confusion.
- **Progress / logs:** NFR-02 also expects user-visible progress for long jobs—**MVP:** periodic `slog.Info` every N files or per directory is enough; full progress bar out of scope unless trivial.
- **Binary name:** Module is `photo-tool`; architecture targets executable **`phototool`**. Until `cmd/phototool` exists, document **`go run . scan ...`** vs **`go build -o phototool`** in Dev Agent Record when implementing.

### Architecture compliance

- **§6 CLI layout:** Cobra root; `scan` calls **`internal/ingest`** (and `store`, `config`) with same **`domain.OperationSummary`** as GUI (§3.9, §4.5).
- **§5.2 boundaries:** CLI wiring in **`internal/cli`** (new) or **`cmd/phototool`** only; **no** SQL in command files—delegate to `store` / `ingest`.
- **§4.2–4.3:** `fmt.Errorf` with `%w`; `log/slog` for operational messages.

### Library / framework

- **New dependency:** `github.com/spf13/cobra` (add to `go.mod` with `go get`; use a current stable minor).
- **Existing:** Go **1.25.4**, Fyne **v2.7.3** (GUI path only), `modernc.org/sqlite`, `dsoprea/go-exif/v3` via `internal/exifmeta`.

### File structure (touch / extend)

- `main.go` — thin entry: detect CLI vs GUI (cobra **Execute** vs launch Fyne) **or** move to `cmd/phototool/main.go` with root `main.go` wrapper per architecture §6 migration path.
- `internal/cli/` (recommended) — cobra command definitions: `root.go`, `scan.go`.
- `internal/ingest/ingest.go` (and tests) — extend with **dry-run-capable** API or shared `ingestOne` variant; avoid duplicating hash/dedup.
- `internal/domain/summary.go` — unchanged contract unless adding scan-specific fields (prefer not).
- Optional: small `internal/ingest/extensions.go` (or similar) for shared image extension list used by `internal/app/upload.go`.

### Testing requirements

- Table-driven tests per architecture §4.4; tempdir + `store.Open` + synthetic JPEGs (reuse patterns from `internal/ingest/ingest_test.go`).
- Explicit test that **dry-run** leaves **`assets` row count** unchanged and **no new files** under `libraryRoot`.

### Previous story intelligence (1.5)

- GUI uses **`ingest.IngestWithAssetIDs`** and **`domain.OperationSummary`** for receipts—**reuse exact field semantics** for CLI output (NFR-04 “one voice”).
- **1.5 review items** (orphan collection, non-atomic create+link) are **upload-only**; scan does not touch collections.
- **Known ingest risk:** late-duplicate detection via SQLite error string matching—scan should rely on **pre-insert hash lookup** like the happy path; same caveats as 1.3 review if insert path is shared.

### Git intelligence summary

- Recent work centers on **`internal/ingest`**, **`internal/app/upload.go`**, **`internal/store`**, and Fyne bootstrap in **`main.go`**—CLI should follow the same library open sequence and logging style as `main()` today.

### Latest technical information

- **Cobra:** Standard Go choice for subcommands; use `RunE` for consistent error propagation; **avoid** global mutable state—pass `db`, `libraryRoot`, and flags into scan runner.

### Scope boundaries

- **In scope:** `scan` subcommand only (Story 1.6). **`import`** CLI is Story 1.7.
- **Out of scope:** Collection assignment, reject/delete semantics, `--json` flag (optional follow-up if architecture §3.9 “later” is activated).

### Project structure notes

- Aligns with architecture §5.1 target tree (`internal/cli`, optional `cmd/phototool`).
- Implementation readiness report notes sprint keys for 1.6–1.8 were missing—this story restores **1-6** tracking.

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Story 1.6, Epic 1, FR-27, NFR-02, NFR-04]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-27, NFR-02, NFR-03, NFR-04, operational tooling]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.2 dedup, §3.9 CLI/GUI parity, §4.1 JSON naming, §4.5 agent rules, §5.1 layout, §5.3 FR mapping, §6 CLI and binary layout]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md — UX-DR6 receipts / CLI parity themes]
- [Source: internal/ingest/ingest.go — `Ingest`, `ingestOne`, hashing + dedup ordering]
- [Source: internal/domain/summary.go — `OperationSummary`]
- [Source: internal/app/upload.go — supported extensions, receipt labels]
- [Source: internal/config/library.go — library resolution]
- [Source: internal/store/open.go — DB location]
- [Source: _bmad-output/implementation-artifacts/1-3-core-ingest.md — ingest contract, streaming per-file pattern]
- [Source: _bmad-output/implementation-artifacts/1-5-upload-confirm-receipt.md — OperationSummary parity with GUI]

## Dev Agent Record

### Agent Model Used

Party mode (automated headless), hook **dev**, sessions **1/2** and **2/2** — roundtables with **simulated** agent voices (`_bmad/_config/agent-manifest.csv` not present in tree).

### Party round 1/2 (dev hook — simulated voices)

- **Murat (Test Architect):** Ingest-layer `TestIngestPaths_dryRun_matchesLiveWhenUnique` already compares summaries, but it runs **both** calls on one DB in-process. I want a **CLI-level** proof on **two empty libraries**: same `--dir`, one `--dry-run` and one live — the four receipt integers must match. That is the user-visible AC2 story, not just package internals.

- **Amelia (Dev):** The wiring from `RunScan` to `IngestPath` is thin, but one integration test is still cheap insurance and documents the contract for QA. **`scanDirInsideLibrary`** is easy to get wrong on path edge cases — a small unit test saves a `..`/prefix regression.

- **Mary (Analyst):** The story task text still listed only the older extension enum; implementation correctly centralizes **`extensions.go`**. Update the story checklist so reviewers do not file false "scope creep" bugs on `.heic`/`.dng`.

- **Winston (Architect):** Fine with Amelia's tests as long as we do not duplicate parsing logic in production — keep summary parsing in **tests only**. NFR-02 is already satisfied by `WalkDir`/`ReadDir` streaming; Murat's CLI parity test does not fight that.

**Orchestrator synthesis:** Add `TestRunScan_dryRun_countsMatch_liveSeparateLibraries` and `TestScanDirInsideLibrary`; refresh the story task bullet to cite `internal/ingest/extensions.go`; leave `sprint-status.yaml` at **review** for `1-6-scan-cli`.

### Party round 2/2 (dev hook — deepen / disagree)

- **Murat (Test Architect):** Round 1 proved parity on a **flat** tree. That never touches `filepath.WalkDir` ordering or skip-dir behavior. I want the **same two-library dry vs live** assertion with **`--recursive`** and files **only under a subdirectory** — if we regress walk+ingest, this catches it; ingest-only tests would not.

- **Amelia (Dev):** I'll take Murat's recursive case. I **disagree** with adding a slog-capture test for the library-root WARN — high friction, low signal now that `scanDirInsideLibrary` has unit coverage. If product wants that, wire a fake `slog.Handler` in a later QA story.

- **Sally (UX Designer):** Power users skim `--help`. Epic text implies recursive is opt-in; the **Short** line doesn't say the default is non-recursive. Add a **`Long`** on `scan` that states "only top-level files unless `--recursive`" so we don't get false bug reports.

- **Winston (Architect):** Sally's `Long` is doc-only — good. Murat's test stays in `internal/cli` to keep the contract at the cobra boundary. **Do not** add a 10k-file integration test in CI; NFR-02 remains design + benchmark comment, not a minutes-long job.

**Orchestrator synthesis:** Add `TestRunScan_dryRun_countsMatch_liveSeparateLibraries_recursive`; add `scan` command `Long` in `internal/cli/root.go`; extend regression task wording; mark story and sprint **done** for `1-6-scan-cli`.


### Debug Log References

### Completion Notes List

- `phototool scan --dir DIR [--recursive] [--dry-run]` via Cobra; **no CLI args** → `main` opens Fyne UI (`internal/cli` stays Fyne-free).
- Dry-run: `ingest.IngestPath` / `IngestPaths` with `dryRun=true` — hash + SELECT dedup only; no copy/insert.
- Scan uses `filepath.WalkDir` (recursive) or `ReadDir` (flat) — one file at a time; NFR-02 note on `RunScan` + `BenchmarkScanWalkDir_processingPattern`.
- Fyne app ID centralized as `internal/app.FyneAppID` (was duplicated from `main`).
- Single extension source in `internal/ingest/extensions.go` — `.heic`/`.dng` included for both CLI scan and GUI picker (`PickerFilterExtensions`); `TestRunScan_secondPass_skipsDuplicates` exercises end-to-end duplicate counts after a second scan.
- **Session 1/2 party:** `TestRunScan_dryRun_countsMatch_liveSeparateLibraries` (CLI dry vs live on two fresh libraries, same tree); `TestScanDirInsideLibrary` for `scanDirInsideLibrary`; story task text aligned with `extensions.go`.
- **Session 2/2 party:** `TestRunScan_dryRun_countsMatch_liveSeparateLibraries_recursive` (same parity under `--recursive`); `scan` **`Long`** help text for non-recursive default; story + sprint closed **done**.
- **Verification (2026-04-13):** `go test ./...` and `go build .` green.

### File List

- `main.go`
- `main_test.go`, `main_fyne_ci_test.go`
- `go.mod`, `go.sum`
- `internal/app/fyne_app_id.go`
- `internal/cli/bootstrap.go`, `internal/cli/root.go` (`scan` Long), `internal/cli/scan.go`, `internal/cli/scan_test.go`
- `internal/ingest/ingest.go`, `internal/ingest/extensions.go`, `internal/ingest/extensions_test.go`, `internal/ingest/ingest_test.go`
- `internal/app/upload.go` — `PickerFilterExtensions()` for picker parity with scan

## Change Log

- 2026-04-13: Party mode (dev hook) session **1/2** — CLI parity test for dry-run vs live summaries on separate empty libraries; `scanDirInsideLibrary` unit test; story tasks synced with `internal/ingest/extensions.go`.
- 2026-04-13: Party mode (dev hook) session **2/2** — **Recursive** dry vs live parity on separate libraries; `scan` command `Long` (non-recursive default); story **done**, sprint `1-6-scan-cli` **done**.
- 2026-04-13: Dev-story run — `go test ./...` / `go build .` green; initial `scan` tests (e.g. `TestRunScan_dryRun_afterLive_skipsAllDuplicates`); status advanced to **review**, then **done** after party sessions **1/2–2/2**.

---

**Story context:** Ultimate context engine analysis completed — comprehensive developer guide created (BMAD create-story workflow, 2026-04-13).
