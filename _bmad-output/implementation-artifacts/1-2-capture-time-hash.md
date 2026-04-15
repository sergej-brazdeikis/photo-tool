# Story 1.2: Capture time and content hash for ingestion

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a **photographer**,  
I want **the system to read capture time and hash files consistently**,  
So that **placement and deduplication match the PRD and architecture**.

**Implements:** FR-01 (input to placement), FR-26 (partial), FR-02/FR-03 inputs, NFR-03.

## Acceptance Criteria

1. **Given** a supported image file with readable EXIF/TIFF capture metadata, **when** the extractor runs, **then** it returns a UTC (or documented) capture instant used for folder placement (FR-01, FR-26 subset).
2. **Given** a file without usable EXIF, **when** the extractor runs, **then** fallback order is **documented in code** (e.g. embedded time → filesystem mtime) and non-silent.
3. **Given** any file path, **when** hashing completes, **then** the result is **SHA-256** lowercase hex matching architecture/NFR-03.
4. **And** unit tests cover at least one EXIF sample (or golden file) and the “no EXIF” fallback path.

## Tasks / Subtasks

- [x] Implement `internal/exifmeta` facade: `ReadCapture` with documented EXIF → mtime fallback and explicit `Source` values (never silent) (AC: 1, 2).
- [x] Use **dsoprea/go-exif/v3** as primary EXIF path per architecture §3.7; document MVP-supported formats in package comment (AC: 1).
- [x] Ensure returned instant is **UTC** and consistent with `paths.CanonicalDayDir` / `paths.SuggestedFilename` (both use UTC date/time parts) (AC: 1).
- [x] Implement `internal/filehash`: `SumHex`, `ReaderHex` — **SHA-256**, **lowercase hex** over full bytes (AC: 3, NFR-03).
- [x] Unit tests: EXIF-backed capture (synthetic JPEG + EXIF blob) and no-EXIF → mtime fallback (AC: 4).
- [x] **Review closure:** resolve any findings from `review` status; add tests if gaps appear (e.g. `SourceMtimeExifUnusable`, `ReaderHex` usage parity with ingest).
- [x] **Party mode dev 1/2 (2026-04-14):** `%w` error context on non-`ErrNoExif` extract + mtime `Stat`; IFD0 `DateTime` / `DateTimeOriginal` coverage + Exif-vs-IFD0 precedence test (AC 1, 4; architecture §4.2–4.3).
- [x] **Party mode dev 2/2 (2026-04-14):** Challenge session 1 — MVP tag/time limits + decode-vs-mtime policy in package doc; malformed EXIF datetime → mtime; empty-file SHA-256 vector; `ReaderHex` read errors use `filehash: read` `%w` chain (AC 2–4, NFR-03).
- [x] **Scope boundary:** wiring capture + hash into the ingest pipeline is **Story 1.3** — do not expand this story into full ingest.

## Dev Notes

### Technical requirements

- **Capture time:** EXIF ASCII date/time tags have no timezone; parse as **local wall time** (`time.Local`), then store/pass **UTC** for layout. This matches `internal/paths.CanonicalDayDir` (“capture time in UTC”) and `SuggestedFilename` (UTC stamp).
- **Fallback:** Architecture §3.2 suggests ordering like EXIF → … → file mtime. MVP implementation documents an explicit chain in `internal/exifmeta` package comment and `Source` constants; filesystem mtime is used when there is no usable EXIF datetime.
- **Hash:** Single algorithm **SHA-256**, lowercase hex, full digest for dedup; filename uses a **fixed-length prefix** (12 hex chars) in Story 1.3 via `paths.SuggestedFilename` — this story owns the **full** hex string API.
- **Determinism (NFR-03):** Same bytes must always yield the same `filehash` output; ingest/scan/import must reuse this package later.

### Architecture compliance

- Follow [Source: _bmad-output/planning-artifacts/architecture.md — §3.2 Library storage and deduplication, §3.7 EXIF and metadata, §4.2–4.3 logging/errors (`log/slog`, `%w` wrapping) when extending].
- Module layout: `internal/exifmeta/` for metadata extraction facade; `internal/filehash/` for hashing — aligns with architecture §5.1 tree (`exifmeta`, hashing alongside `paths`, `store`).

### Library / module

- **Direct dependency:** `github.com/dsoprea/go-exif/v3` **v3.0.1** (see `go.mod`). Do not swap to ExifTool or other parsers without an architecture change.

### File structure (touch / extend)

- `internal/exifmeta/capture.go` — `ReadCapture`, `Result`, `Source`
- `internal/exifmeta/capture_test.go` — EXIF + no-EXIF tests
- `internal/filehash/filehash.go` — `SumHex`, `ReaderHex`
- `internal/filehash/filehash_test.go` — known-vector SHA-256 test
- `internal/paths/canonical.go` — consumers of `captureUTC` (Story 1.3); do not break tests

### Testing requirements

- `go test ./internal/exifmeta/... ./internal/filehash/... ./internal/paths/...`
- At least one test with **readable EXIF capture metadata** and one **no EXIF / mtime fallback** path (AC: 4).
- Prefer small generated fixtures (as in existing tests) or checked-in minimal golden files if CI needs them.

### Previous story intelligence (1.1)

- **Story 1.1** (local library foundation): library root, `.phototool`, SQLite migrations, `assets` uniqueness — see [Source: _bmad-output/planning-artifacts/epics.md — Story 1.1]. Sprint: `1-1-library-foundation: done`. This story must not regress store or path helpers.

### Cross-story context

- **Story 1.3** depends on this story: ingest must call `exifmeta.ReadCapture` and `filehash` for placement, naming, and dedup. See [Source: _bmad-output/implementation-artifacts/1-3-core-ingest.md].

### Project Structure Notes

- Keep EXIF and hashing **out of Fyne** — pure libraries for CLI/GUI parity later.
- Avoid duplicating hash or EXIF logic in `internal/ingest`; call shared packages (architecture §3.2 “one shared function” for dedup inputs).

### References

- [Source: _bmad-output/planning-artifacts/epics.md — Epic 1, Story 1.2]
- [Source: _bmad-output/planning-artifacts/architecture.md — §3.2, §3.7, §5.1 module tree]
- [Source: _bmad-output/planning-artifacts/PRD.md — FR-01, FR-02, FR-26, NFR-03, Provenance]
- [Source: internal/exifmeta/capture.go — package documentation, fallback chain, timezone rule]
- [Source: internal/filehash/filehash.go — SHA-256 hex API]
- [Source: internal/paths/canonical.go — `CanonicalDayDir`, `SuggestedFilename`]

## Dev Agent Record

### Agent Model Used

Cursor agent (implementation); verify pass via `scripts/bmad-story-workflow.sh --phase=verify 1.2`.

### Debug Log References

- `scripts/bmad-story-workflow.sh`: `EXTRA_AGENT_ARGS[@]` with `set -u` — fixed earlier with conditional expansion in `run_agent`.

### Completion Notes List

- Added `internal/exifmeta` with `ReadCapture`, documented EXIF → mtime fallback and `Source` strings; EXIF datetime parsed as local wall time then `.UTC()`.
- `go test ./...` and `go build .` pass locally; `github.com/dsoprea/go-exif/v3` is a direct module dependency.
- Ingest wiring to `exifmeta` / `filehash` deferred to a later story (per verify notes).
- **Review closure (2026-04-13):** Added `TestReadCapture_exifWithoutDateTimeUsesMtimeUnusable` for `SourceMtimeExifUnusable`; added `TestReaderHex_matchesSumHex` and `TestReaderHex_matchesSumHex_afterSeekFromEnd` so `ReaderHex` on `*os.File` matches `SumHex` and matches the ingest pattern (seek then hash). Scope boundary unchanged: no additional ingest wiring in this story.
- **Party mode dev session 1/2 (2026-04-14):** Simulated dev round (John / Sally / Winston / Amelia) → align `exifmeta` with architecture error-wrapping, close IFD0 fallback/precedence test gaps, keep ingest wiring deferred to 1.3.
- **Review closure (2026-04-14):** `SumHex` now wraps `os.Open` with `fmt.Errorf("filehash: open %q: %w", path, err)` (parity with `ReaderHex` `%w` chain). Added `TestSumHex_openErrorWraps` asserting `errors.Is(..., os.ErrNotExist)`.
- **Party patch (2026-04-14):** `ReadCapture` wraps non-`ErrNoExif` extract errors and `mtimeResult` `Stat` errors; tests for unreadable-file extract wrap (skipped on Windows where `chmod` is not a reliable read-deny probe), IFD0 datetime sources, and Exif `DateTimeOriginal` precedence over IFD0 `DateTime`.
- **Party mode dev 2/2 (2026-04-14):** Documented MVP exclusions (offset tags, sub-second, `DateTimeDigitized` / `CreateDate`) and intentional mtime fallback when internal EXIF decode fails or datetime strings do not parse; `TestReadCapture_malformedDateTimeOriginalFallsBackToMtime`; `TestSumHex_emptyFile`; `TestReaderHex_readErrorWraps`; `ReaderHex` error prefix aligned with `SumHex` (`filehash: read: %w`).

### File List

- `internal/exifmeta/capture.go`
- `internal/exifmeta/capture_test.go`
- `internal/filehash/filehash.go`
- `internal/filehash/filehash_test.go`
- `go.mod` / `go.sum` (tidy; direct `go-exif` require)

### Change Log

- **2026-04-14:** Party mode dev 2/2 — `exifmeta` package doc (MVP tag/decode policy) + malformed-datetime test; `filehash` empty digest + `ReaderHex` read-wrap test; story → `done`; sprint `1-2-capture-time-hash` → `done`.
- **2026-04-14:** Party mode dev 1/2 — `exifmeta` extract/stat `%w` wraps + IFD0 + precedence tests; story + sprint note; status remains `review`.
- **2026-04-14:** BMAD dev-story — `SumHex` open-error `%w` wrap + `TestSumHex_openErrorWraps`; review finding closed; story status → `review`; sprint `1-2-capture-time-hash` → `review`.
- **2026-04-13:** Review-closure tests — `SourceMtimeExifUnusable` path and `ReaderHex` parity with `SumHex` / ingest-style seek; story tasks completed; sprint status → `review`.
- **2026-04-13:** BMAD code review — findings appended under Review Findings; status → `in-progress` until patch items addressed.

### Review Findings

- [x] [Review][Patch] Wrap `os.Open` errors in `SumHex` for consistent `%w` error chain like `ReaderHex` — `internal/filehash/filehash.go:12-18`
- [x] [Review][Defer] `ReadCapture` drops underlying EXIF parse/collect errors when falling back to mtime (`SourceMtimeExifUnusable`); callers only see provenance via `Source`, not the root failure — `internal/exifmeta/capture.go:59-64` — deferred, MVP acceptable; revisit for observability/ingest logging
- [x] [Review][Defer] No use of `OffsetTimeOriginal` / sub-second EXIF fields; local-wall → UTC rule can disagree with camera-reported offset for placement — `internal/exifmeta/capture.go` — deferred, document MVP limitation or schedule follow-up if PRD requires
- [x] [Review][Defer] Dependency `SearchFileAndExtractExif` reads from detected EXIF start to EOF (large allocations on big files) — upstream `go-exif` behavior — deferred, monitor NFR/memory if needed
- [x] [Review][Patch] Wrap non-`ErrNoExif` `SearchFileAndExtractExif` failures and mtime `Stat` with `%w` + path context — `internal/exifmeta/capture.go` (party dev 1/2)
