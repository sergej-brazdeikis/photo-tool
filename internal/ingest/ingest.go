// Package ingest copies sources into canonical library storage and registers rows in assets (Story 1.3).
//
// Deduplication: the persisted key is the full-byte SHA-256 digest stored in assets.content_hash
// (lowercase hex). Matching digests imply matching file bytes and thus matching size; the schema does
// not store file_size separately (epic “size + hash” intent is satisfied by the single digest key).
//
// Filesystem/DB ordering: capture time and SHA-256 are read from the source using one open file
// (hash via [photo-tool/internal/filehash.ReaderHex], then seek to start for copy). Destination path
// is derived only after the hash is known. After a successful copy, InsertAsset runs; if insert fails,
// the copied file is usually removed, the error is logged, and Failed is incremented. A
// SQLITE_CONSTRAINT_UNIQUE after copy is resolved with [resolveUniqueIngestCollision]: late duplicate
// on the same canonical rel_path must not unlink the winner’s file; duplicate-bytes under another
// rel_path removes only the orphan copy.
//
// Concurrent ingest of the same destination path is serialized with a per-dest mutex so two writers
// cannot truncate each other’s file (O_TRUNC). [copyToFile] Syncs and checks source vs dest size.
package ingest

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"photo-tool/internal/domain"
	"photo-tool/internal/exifmeta"
	"photo-tool/internal/filehash"
	"photo-tool/internal/paths"
	"photo-tool/internal/store"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// destCopyLocks serialize copy+insert for a single canonical destination path. Concurrent ingest of the
// same logical file used to open the same dest with O_TRUNC and clobber in-flight copies (see ingest tests).
var destCopyLocks sync.Map // map[string]*sync.Mutex — key: filepath.Clean(abs dest)

func destCopyLock(destAbs string) *sync.Mutex {
	v, _ := destCopyLocks.LoadOrStore(filepath.Clean(destAbs), new(sync.Mutex))
	return v.(*sync.Mutex)
}

// Ingest processes each path in order; per-file errors increment Failed and do not stop the batch.
func Ingest(db *sql.DB, libraryRoot string, sourcePaths []string) domain.OperationSummary {
	return IngestPaths(db, libraryRoot, sourcePaths, false)
}

// IngestPaths is like [Ingest] but can run in dry-run mode (hash + read-only dedup only; no copy/insert).
func IngestPaths(db *sql.DB, libraryRoot string, sourcePaths []string, dryRun bool) domain.OperationSummary {
	var sum domain.OperationSummary
	libRoot := filepath.Clean(libraryRoot)
	var drySeen map[string]struct{}
	if dryRun {
		drySeen = make(map[string]struct{})
	}
	for _, p := range sourcePaths {
		_ = ingestOne(db, libRoot, p, &sum, dryRun, drySeen)
	}
	return sum
}

// IngestPath processes a single file and updates sum. When dryRun is true, no files are copied and no
// assets rows are written; Added/SkippedDuplicate/Failed still reflect the would-be outcome.
// When dryRun is true, drySeen records content hashes already classified as Added in this batch so a
// second path with identical bytes matches live behavior (DB dedup only sees prior rows after insert).
// Pass nil when processing a single path or when in-run duplicate classification is irrelevant.
func IngestPath(db *sql.DB, libraryRoot, srcPath string, sum *domain.OperationSummary, dryRun bool, drySeen map[string]struct{}) int64 {
	return ingestOne(db, filepath.Clean(libraryRoot), filepath.Clean(srcPath), sum, dryRun, drySeen)
}

// IngestWithAssetIDs runs the same pipeline as [Ingest]. For each source path, the parallel element in
// the returned slice is the canonical assets.id when ingest resolved to a row (added or duplicate), or 0
// when the file failed to ingest.
func IngestWithAssetIDs(db *sql.DB, libraryRoot string, sourcePaths []string) (domain.OperationSummary, []int64) {
	var sum domain.OperationSummary
	libRoot := filepath.Clean(libraryRoot)
	ids := make([]int64, len(sourcePaths))
	for i, p := range sourcePaths {
		ids[i] = ingestOne(db, libRoot, p, &sum, false, nil)
	}
	return sum, ids
}

func ingestOne(db *sql.DB, libraryRoot, srcPath string, sum *domain.OperationSummary, dryRun bool, drySeen map[string]struct{}) int64 {
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
		if drySeen != nil {
			if _, ok := drySeen[hashHex]; ok {
				sum.SkippedDuplicate++
				return 0
			}
			drySeen[hashHex] = struct{}{}
		}
		sum.Added++
		return 0
	}

	ext := filepath.Ext(srcPath)
	dayDir := paths.CanonicalDayDir(libraryRoot, capRes.UTC)
	name := paths.SuggestedFilename(capRes.UTC, hashHex, ext)
	destAbs := filepath.Join(dayDir, name)

	destMu := destCopyLock(destAbs)
	destMu.Lock()
	defer destMu.Unlock()

	existingID2, exists2, err := store.AssetIDByContentHash(db, hashHex)
	if err != nil {
		sum.Failed++
		slog.Error("ingest: dedup lookup under dest lock", "path", srcPath, "err", err)
		return 0
	}
	if exists2 {
		sum.SkippedDuplicate++
		return existingID2
	}

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

	idNew, err := store.InsertAssetWithCamera(db, hashHex, relPath, captureUnix, createdAt, cam.Make, cam.Model)
	if err != nil {
		if isConstraintUnique(err) {
			idLate, skip, removeDest, err2 := resolveUniqueIngestCollision(db, hashHex, relPath)
			if err2 != nil {
				if removeDest {
					_ = os.Remove(destAbs)
				}
				sum.Failed++
				slog.Error("ingest: resolve unique collision", "path", srcPath, "err", err2)
				return 0
			}
			if removeDest {
				_ = os.Remove(destAbs)
			}
			if skip {
				sum.SkippedDuplicate++
				return idLate
			}
			sum.Failed++
			slog.Error("ingest: insert asset", "path", srcPath, "rel_path", relPath, "err", err)
			return 0
		}
		_ = os.Remove(destAbs)
		sum.Failed++
		slog.Error("ingest: insert asset", "path", srcPath, "rel_path", relPath, "err", err)
		return 0
	}

	sum.Added++
	return idNew
}

// resolveUniqueIngestCollision interprets SQLITE_CONSTRAINT_UNIQUE after a failed asset insert.
// When skip is true, the ingest is treated as skipped_duplicate; removeDest is set when the copied
// file at the attempted path should be deleted (orphan or wrong-path duplicate).
func resolveUniqueIngestCollision(db *sql.DB, contentHash, relPath string) (id int64, skip bool, removeDest bool, err error) {
	idH, relH, _, okH, errH := store.AssetRowByContentHash(db, contentHash)
	if errH != nil {
		return 0, false, true, errH
	}
	if okH && filepath.ToSlash(relH) == filepath.ToSlash(relPath) {
		return idH, true, false, nil
	}
	idP, hashP, _, okP, errP := store.ActiveAssetByRelPath(db, relPath)
	if errP != nil {
		return 0, false, true, errP
	}
	if okP && hashP == contentHash {
		return idP, true, false, nil
	}
	if okH {
		return idH, true, true, nil
	}
	if okP && hashP != contentHash {
		return 0, false, true, nil
	}
	return 0, false, true, nil
}

func isConstraintUnique(err error) bool {
	if err == nil {
		return false
	}
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	return se.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}

func copyToFile(src *os.File, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("mkdir dest dir: %w", err)
	}
	srcInfo, err := src.Stat()
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}
	wantSize := srcInfo.Size()

	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("copy: %w", err)
	}
	if err := dst.Sync(); err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("sync dest: %w", err)
	}
	fi, err := os.Stat(destPath)
	if err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("stat dest: %w", err)
	}
	if fi.Size() != wantSize {
		_ = os.Remove(destPath)
		return fmt.Errorf("copy size mismatch: got %d want %d", fi.Size(), wantSize)
	}
	return nil
}

func isUniqueContentHash(err error) bool {
	if err == nil {
		return false
	}
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	// modernc returns extended SQLITE_CONSTRAINT_UNIQUE for UNIQUE index violations.
	if se.Code() != sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return false
	}
	msg := strings.ToLower(se.Error())
	return strings.Contains(msg, "content_hash") || strings.Contains(msg, "idx_assets_content_hash")
}
