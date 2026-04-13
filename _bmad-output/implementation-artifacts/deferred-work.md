## Deferred from: code review of 1-3-core-ingest.md (2026-04-13)

- `copyToFile` does not call `dst.Sync()` after `io.Copy`; durability under crash/power loss is OS-dependent (`internal/ingest/ingest.go`).

- Suggested filename uses a 12-hex prefix and second-resolution UTC timestamp; distinct digests could theoretically map to the same relative path, so `O_TRUNC` could overwrite another asset’s bytes before a DB constraint failure. Accept as negligible-risk MVP or address via longer prefixes / exclusive create semantics with a defined retry story (`internal/paths/canonical.go`, `internal/ingest/ingest.go`).

## Deferred from: code review of 1-2-capture-time-hash.md (2026-04-13)

- `ReadCapture` drops underlying EXIF parse/collect errors when falling back to mtime (`SourceMtimeExifUnusable`); callers only see provenance via `Source`, not the root failure (`internal/exifmeta/capture.go:59-64`). Revisit for observability/ingest logging.

- No use of `OffsetTimeOriginal` / sub-second EXIF fields; local-wall → UTC rule can disagree with camera-reported offset for placement (`internal/exifmeta/capture.go`). Document MVP limitation or schedule follow-up if PRD requires.

- Dependency `SearchFileAndExtractExif` reads from detected EXIF start to EOF (large allocations on big files); upstream `go-exif` behavior. Monitor NFR/memory if needed.

## Deferred from: code review of 1-5-upload-confirm-receipt.md (2026-04-13)

- `isUniqueContentHash` detects late-duplicate races by matching substrings in SQLite/driver error text; fragile if messages change (`internal/ingest/ingest.go`).
