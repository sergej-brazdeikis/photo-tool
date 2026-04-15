## Deferred from: code review of 2-11-layout-display-scaling-gate.md (2026-04-15)

- Collections NFR-01 gate assumes the first `widget.List` is the album list (`lists[0].Select(0)`); future UI that adds another List ahead of it can mis-target or flake (`internal/app/nfr01_layout_gate_test.go`).

- `assertNFR01GateThumbnailGridListsOnCanvas` may match any sufficiently large List when multiple lists are present; tighten only if CI false greens become likely (`internal/app/nfr01_layout_gate_test.go`).

- AC2 resize sweep: document or validate whether `win.SetContent(shell)` after each `Resize` is required for the test driver or is redundant (`internal/app/nfr01_layout_gate_test.go`).

## Deferred from: code review of 2-3-thumbnail-grid-rating-badges.md (2026-04-15)

- `refreshReviewData` and `refreshRejectedData` both embed a large `ListCollections` + `collectionOpts` / `collectionIDs` reconciliation block; consider a shared helper so album strip behavior cannot drift between Review and Rejected (`internal/app/review.go`, `internal/app/rejected.go`).

- `newMainShell` panics if `panels[it.key] == nil` for any primary nav item — acceptable fail-fast for programmer error; revisit only if product wants degraded startup instead of crash (`internal/app/shell.go`).

- `gotoReview` now runs `clearReviewUndoIfLeftReview` before updating nav selection / `selectPanel`; confirm undo semantics with a focused regression test or manual pass (Rejected → Review via rail vs in-panel actions) (`internal/app/shell.go`).

## Deferred from: code review of 4-1-multi-asset-snapshot-packages.md (2026-04-14)

- Package member bytes (`serveImage`) call `ResolvePackageShareLink` on every `/i/{token}/{n}` request, repeating work already done for HTML; acceptable MVP, optimize if share traffic becomes hot (`internal/share/handler.go`).

## Deferred from: code review of 3-5-share-performance-abuse.md (2026-04-14)

- Visitor-map eviction at cap is arbitrary (not LRU): bounded-memory tradeoff; revisit only if a future release needs fairer eviction under many distinct client keys (`internal/share/ratelimit.go`).

- NFR-05 cold-load gate is synthetic (httptest, small JPEG, median over nine trials); staging or larger fixtures may be needed if PRD tightens or CI flakes persist (`internal/share/nfr05_cold_load_test.go`, `docs/share-cold-load-nfr05.md`).

## Deferred from: code review of 3-3-share-html-readonly.md (2026-04-14)

- Package share HTML (`servePackageHTML`) captions include numeric asset id and file basename from live grid/DB data — stricter privacy/copy should be owned by Epic 4.1 if package pages must meet Story 3.3–style identifier hygiene (`internal/share/handler.go`).

- Identifier-leak regression test could be strengthened to fail if the concrete `assets.id` appears in rendered single-asset HTML (avoid collisions with small ids vs. “Rating: N” copy) (`internal/share/http_test.go`).

- Symlink follow on `os.Open` for share image bytes remains a documented residual; O_NOFOLLOW or explicit policy is a follow-up (`internal/share/handler.go`).

## Deferred from: code review of 3-1-share-preview-snapshot-mint.md (2026-04-14)

- Share preview uses `time.Unix(row.CaptureTimeUnix, 0)` for the label; when capture time is missing (0), the UI shows the Unix epoch. Revisit if ingest guarantees non-zero capture times or if empty/missing should display “—” (`internal/app/share_loupe.go`).

## Deferred from: code review of 1-3-core-ingest.md (2026-04-13)

- `copyToFile` does not call `dst.Sync()` after `io.Copy`; durability under crash/power loss is OS-dependent (`internal/ingest/ingest.go`).

- Suggested filename uses a 12-hex prefix and second-resolution UTC timestamp; distinct digests could theoretically map to the same relative path, so `O_TRUNC` could overwrite another asset’s bytes before a DB constraint failure. Accept as negligible-risk MVP or address via longer prefixes / exclusive create semantics with a defined retry story (`internal/paths/canonical.go`, `internal/ingest/ingest.go`).

## Deferred from: code review of 1-2-capture-time-hash.md (2026-04-13)

- `ReadCapture` drops underlying EXIF parse/collect errors when falling back to mtime (`SourceMtimeExifUnusable`); callers only see provenance via `Source`, not the root failure (`internal/exifmeta/capture.go:59-64`). Revisit for observability/ingest logging.

- No use of `OffsetTimeOriginal` / sub-second EXIF fields; local-wall → UTC rule can disagree with camera-reported offset for placement (`internal/exifmeta/capture.go`). Document MVP limitation or schedule follow-up if PRD requires.

- Dependency `SearchFileAndExtractExif` reads from detected EXIF start to EOF (large allocations on big files); upstream `go-exif` behavior. Monitor NFR/memory if needed.

## Deferred from: code review of 1-5-upload-confirm-receipt.md (2026-04-13)

- `isUniqueContentHash` detects late-duplicate races by matching substrings in SQLite/driver error text; fragile if messages change (`internal/ingest/ingest.go`).

## Deferred from: code review of 1-7-import-cli.md (2026-04-14)

- `scanSummaryFromOutput` ignores lines that do not match receipt prefixes; extra stdout noise could mask parity regressions in dry-vs-live CLI tests (`internal/cli/scan_test.go`).

## Deferred from: code review of 1-8-drag-drop-upload.md (2026-04-14)

- Drop hit-test for the designated zone uses `fyne.Driver.AbsolutePositionForObject` inside a `container.Scroll`; geometry is not unit-tested headless — rely on manual QA (off-target silent ignore, scrolled layout) before marking Story 1.8 done (`internal/app/upload.go`, `internal/app/drop_paths.go`).

- Upload view owns `Window.SetCloseIntercept` for import-in-flight; a future shell-level quit guard must chain with this callback instead of replacing it (`internal/app/upload.go`).

- The Story 1.8 working diff also touches async ingest, receipt UI, and transactional collection create+link — run a focused Story 1.5 / FR-06 regression smoke when closing 1.8 (`internal/app/upload.go`, `internal/store`).

- `uriLocalPath` uses `fyne.URI.Path()` for local `file:` drops; validate drive-letter and encoded-path behavior on Windows when CI or hardware is available (`internal/app/drop_paths.go`).

## Deferred from: code review of 2-1-app-shell-navigation-themes.md (2026-04-14)

- `selectPanel` has no guard for unknown keys; a bad `key` would pass `nil` into the center stack (`internal/app/shell.go:100-104`).

- Semantic role preview strip in the left rail competes for vertical space vs UX-DR16 compact shell baseline; confirm during Story 2.11 layout passes (`internal/app/shell.go:145-149`).

## Deferred from: code review of 2-1-app-shell-navigation-themes.md (2026-04-15)

- `gotoReview` still binds the Review destination to the second primary nav slot (`labels[1]`). Reordering `primaryNavItems` can make programmatic navigation disagree with button handlers that key off `item.key` (`internal/app/shell.go` ~69–81).

- `theme_test.go` color regressions use pointer/struct equality of swatches; they do not prove WCAG contrast or that focus rings are human-perceptible under AC10 — rely on manual QA or add stronger checks later (`internal/app/theme_test.go`).

## Deferred from: code review of 1-3-core-ingest.md (2026-04-14)

- Ingest holds a `sync.Map` of per-destination `*sync.Mutex` forever (`destCopyLocks`); consider eviction or a bounded cache if single-process imports can cover millions of unique canonical paths (`internal/ingest/ingest.go`).

## Deferred from: code review of 2-4-review-loupe-keyboard-rating.md (2026-04-14)

- Failed grid pages stay cached until `invalidatePages` / `reset`; a transient SQLite error can leave a page stuck in the user-visible failure state until the user changes filters or otherwise triggers a refresh (`internal/app/review_grid.go`).

## Deferred from: code review of 2-6-reject-undo-hidden-restore.md (2026-04-14)

- Same grid `pageFailed` behavior as Story 2.4 review: no automatic retry after a failed page query; applies to **Rejected** as well via shared `reviewAssetGrid` (`internal/app/review_grid.go` `ensurePageLocked`).

- `ListRejectedForReview` orders by `capture_time_unix DESC, id DESC` (aligned with default Review list), not by `rejected_at_unix`; confirm whether the Rejected surface should surface “most recently hidden” first (`internal/store/review_query.go`).

## Deferred from: code review of 2-7-delete-quarantine.md (2026-04-14)

- Orphan thumbnail files under `.cache/thumbnails` after delete — optional invalidation only; called out in Story 2.7 Dev Notes, not AC-blocking.

## Deferred from: code review of 2-9-collection-crud-multi-assign.md (2026-04-14)

- Duplicate album names make Fyne `Select` / checklist labels ambiguous (first match wins); schema allows duplicates — story risk, optional disambiguation (e.g. display date in labels).

- `LinkAssetsToCollection` with empty `assetIDs` is a documented no-op and does not validate `collectionID`; caller responsibility.

## Deferred from: code review of 2-12-empty-states-error-tone.md (2026-04-14)

- Default upload ingest runs on a background goroutine with `fyne.Do` completion; headless tests force `SynchronousIngest: true` per `UploadViewOptions` docs — async scheduling remains manual/field validation (`internal/app/upload.go`, `upload_fr06_flow_test.go`).

- UX-DR18 / AC-LIST-STATES (loading + error states for Review/collection grid) is captured as story backlog in `2-12-empty-states-error-tone.md`, not implemented in the Story 2.12 delivery slice.

## Deferred from: code review of 2-11-layout-display-scaling-gate.md (2026-04-14)

- Filter-strip layout tests assume the first three `Select` widgets in tree order are the strip; a future `Select` ahead of the strip would break `assertReviewFilterStripOnScreen` (`internal/app/review_test.go`, `internal/app/nfr01_layout_gate_test.go`).

- Loupe gate body duplicates `review_loupe.go` structure manually; drift risk if production loupe layout changes (`internal/app/nfr01_layout_gate_test.go`).

- UX-DR16 quantitative thresholds from the story backlog (thumb minimum, letterbox region, chrome budget, focus at NFR-01 minimum) are not automated; manual evidence only (`2-11-layout-display-scaling-gate.md` UX backlog delta).

## Deferred from: code review of 1-5-upload-confirm-receipt.md (2026-04-15)

- AC5 minimum thumbnail edge (140dp via `uploadPreviewThumbMin`) is not asserted in automated tests; implementation matches Direction E (`internal/app/upload.go`).
