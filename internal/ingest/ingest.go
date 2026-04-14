// Package ingest copies sources into canonical library storage and registers rows in assets (Story 1.3).
//
// Deduplication: the persisted key is the full-byte SHA-256 digest stored in assets.content_hash
// (lowercase hex). Matching digests imply matching file bytes and thus matching size; the schema does
// not store file_size separately (epic “size + hash” intent is satisfied by the single digest key).
//
// Filesystem/DB ordering: capture time and SHA-256 are read from the source using one open file
// (hash via [photo-tool/internal/filehash.ReaderHex], then seek to start for copy). Destination path
// is derived only after the hash is known. After a successful copy, InsertAsset runs; if insert fails,
// the copied file is removed, the error is logged, and Failed is incremented. A UNIQUE violation on
// content_hash after copy is treated as a late duplicate (e.g. race): the file is removed and
// SkippedDuplicate is incremented.
package ingest

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"photo-tool/internal/domain"
	"photo-tool/internal/exifmeta"
	"photo-tool/internal/filehash"
	"photo-tool/internal/paths"
	"photo-tool/internal/store"
)

// Ingest processes each path in order; per-file errors increment Failed and do not stop the batch.
func Ingest(db *sql.DB, libraryRoot string, sourcePaths []string) domain.OperationSummary {
	return IngestPaths(db, libraryRoot, sourcePaths, false)
}

// IngestPaths is like [Ingest] but can run in dry-run mode (hash + read-only dedup only; no copy/insert).
func IngestPaths(db *sql.DB, libraryRoot string, sourcePaths []string, dryRun bool) domain.OperationSummary {
	var sum domain.OperationSummary
	libRoot := filepath.Clean(libraryRoot)
	for _, p := range sourcePaths {
		_ = ingestOne(db, libRoot, p, &sum, dryRun)
	}
	return sum
}

// IngestPath processes a single file and updates sum. When dryRun is true, no files are copied and no
// assets rows are written; Added/SkippedDuplicate/Failed still reflect the would-be outcome.
func IngestPath(db *sql.DB, libraryRoot, srcPath string, sum *domain.OperationSummary, dryRun bool) int64 {
	return ingestOne(db, filepath.Clean(libraryRoot), filepath.Clean(srcPath), sum, dryRun)
}

// IngestWithAssetIDs runs the same pipeline as [Ingest]. For each source path, the parallel element in
// the returned slice is the canonical assets.id when ingest resolved to a row (added or duplicate), or 0
// when the file failed to ingest.
func IngestWithAssetIDs(db *sql.DB, libraryRoot string, sourcePaths []string) (domain.OperationSummary, []int64) {
	var sum domain.OperationSummary
	libRoot := filepath.Clean(libraryRoot)
	ids := make([]int64, len(sourcePaths))
	for i, p := range sourcePaths {
		ids[i] = ingestOne(db, libRoot, p, &sum, false)
	}
	return sum, ids
}

func ingestOne(db *sql.DB, libraryRoot, srcPath string, sum *domain.OperationSummary, dryRun bool) int64 {
	srcPath = filepath.Clean(srcPath)
	capRes, err := exifmeta.ReadCapture(srcPath)
	if err != nil {
		sum.Failed++
		slog.Error("ingest: read capture", "path", srcPath, "err", err)
		return 0
	}
	cam, camErr := exifmeta.ReadCamera(srcPath)
	if camErr != nil {
		slog.Warn("ingest: read camera", "path", srcPath, "err", camErr)
		cam = exifmeta.CameraStrings{}
	}

	f, err := os.Open(srcPath)
	if err != nil {
		sum.Failed++
		slog.Error("ingest: open source", "path", srcPath, "err", err)
		return 0
	}
	defer f.Close()

	hashHex, err := filehash.ReaderHex(f)
	if err != nil {
		sum.Failed++
		slog.Error("ingest: hash", "path", srcPath, "err", err)
		return 0
	}

	existingID, exists, err := store.AssetIDByContentHash(db, hashHex)
	if err != nil {
		sum.Failed++
		slog.Error("ingest: dedup lookup", "path", srcPath, "err", err)
		return 0
	}
	if exists {
		sum.SkippedDuplicate++
		return existingID
	}

	if dryRun {
		sum.Added++
		return 0
	}

	ext := filepath.Ext(srcPath)
	dayDir := paths.CanonicalDayDir(libraryRoot, capRes.UTC)
	name := paths.SuggestedFilename(capRes.UTC, hashHex, ext)
	destAbs := filepath.Join(dayDir, name)

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		sum.Failed++
		slog.Error("ingest: seek for copy", "path", srcPath, "err", err)
		return 0
	}
	if err := copyToFile(f, destAbs); err != nil {
		sum.Failed++
		slog.Error("ingest: copy", "path", srcPath, "dest", destAbs, "err", err)
		return 0
	}

	relPath, err := filepath.Rel(libraryRoot, destAbs)
	if err != nil {
		_ = os.Remove(destAbs)
		sum.Failed++
		slog.Error("ingest: rel path", "library", libraryRoot, "dest", destAbs, "err", err)
		return 0
	}
	relPath = filepath.ToSlash(relPath)

	captureUnix := capRes.UTC.Unix()
	createdAt := time.Now().Unix()

	err = store.InsertAssetWithCamera(db, hashHex, relPath, captureUnix, createdAt, cam.Make, cam.Model)
	if err != nil {
		_ = os.Remove(destAbs)
		if isUniqueContentHash(err) {
			sum.SkippedDuplicate++
			idLate, ok, err2 := store.AssetIDByContentHash(db, hashHex)
			if err2 != nil || !ok {
				if err2 != nil {
					slog.Error("ingest: resolve id after unique", "path", srcPath, "err", err2)
				} else {
					slog.Error("ingest: missing row after unique race", "path", srcPath)
				}
				sum.Failed++
				sum.SkippedDuplicate--
				return 0
			}
			return idLate
		}
		sum.Failed++
		slog.Error("ingest: insert asset", "path", srcPath, "rel_path", relPath, "err", err)
		return 0
	}

	sum.Added++
	idNew, ok, err := store.AssetIDByContentHash(db, hashHex)
	if err != nil {
		slog.Error("ingest: resolve id after insert", "path", srcPath, "err", err)
		sum.Failed++
		sum.Added--
		return 0
	}
	if !ok {
		slog.Error("ingest: missing row after insert", "path", srcPath)
		sum.Failed++
		sum.Added--
		return 0
	}
	return idNew
}

func copyToFile(src *os.File, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("mkdir dest dir: %w", err)
	}
	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

func isUniqueContentHash(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if !strings.Contains(msg, "UNIQUE constraint failed") {
		return false
	}
	return strings.Contains(msg, "content_hash") || strings.Contains(msg, "idx_assets_content_hash")
}
