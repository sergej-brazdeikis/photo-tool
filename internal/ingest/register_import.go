package ingest

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"photo-tool/internal/domain"
	"photo-tool/internal/exifmeta"
	"photo-tool/internal/filehash"
	"photo-tool/internal/store"
)

// RegisterInPlacePath registers a supported image already under libraryRoot (Story 1.7: no copy).
// Hashing and dedup match [ingestOne] / scan (NFR-03). Walk callers use filepath.WalkDir one path at a time (NFR-02).
//
// Backfill (AC4): when content_hash already matches the row at this rel_path but capture_time_unix
// differs from [exifmeta.ReadCapture], the row is updated and sum.Updated is incremented (dry-run: count only).
func RegisterInPlacePath(db *sql.DB, libraryRoot, absPath string, sum *domain.OperationSummary, dryRun bool) {
	libRoot := filepath.Clean(libraryRoot)
	absPath = filepath.Clean(absPath)

	relPath, err := filepath.Rel(libRoot, absPath)
	if err != nil {
		sum.Failed++
		slog.Error("import: rel to library", "path", absPath, "library", libRoot, "err", err)
		return
	}
	relPath = filepath.ToSlash(relPath)

	capRes, err := exifmeta.ReadCapture(absPath)
	if err != nil {
		sum.Failed++
		slog.Error("import: read capture", "path", absPath, "err", err)
		return
	}
	cam, camErr := exifmeta.ReadCamera(absPath)
	if camErr != nil {
		slog.Warn("import: read camera", "path", absPath, "err", camErr)
		cam = exifmeta.CameraStrings{}
	}
	captureUnix := capRes.UTC.Unix()

	f, err := os.Open(absPath)
	if err != nil {
		sum.Failed++
		slog.Error("import: open", "path", absPath, "err", err)
		return
	}
	defer f.Close()

	hashHex, err := filehash.ReaderHex(f)
	if err != nil {
		sum.Failed++
		slog.Error("import: hash", "path", absPath, "err", err)
		return
	}

	idByHash, relForHash, capForHash, hashExists, err := store.AssetRowByContentHash(db, hashHex)
	if err != nil {
		sum.Failed++
		slog.Error("import: hash lookup", "path", absPath, "err", err)
		return
	}

	if hashExists {
		if filepath.ToSlash(relForHash) == relPath {
			if capForHash != captureUnix {
				if dryRun {
					sum.Updated++
					return
				}
				if err := store.UpdateAssetCaptureTime(db, idByHash, captureUnix); err != nil {
					sum.Failed++
					slog.Error("import: backfill capture", "path", absPath, "id", idByHash, "err", err)
					return
				}
				sum.Updated++
				return
			}
			sum.SkippedDuplicate++
			return
		}
		sum.SkippedDuplicate++
		return
	}

	idPath, hashPath, _, pathExists, err := store.ActiveAssetByRelPath(db, relPath)
	if err != nil {
		sum.Failed++
		slog.Error("import: path lookup", "path", absPath, "err", err)
		return
	}
	if pathExists && hashPath != hashHex {
		sum.Failed++
		slog.Error("import: rel_path occupied by different content; resolve manually", "rel_path", relPath, "db_hash", hashPath, "disk_hash", hashHex)
		return
	}
	if pathExists && hashPath == hashHex {
		// Should have matched hashExists; defensive.
		sum.SkippedDuplicate++
		_ = idPath
		return
	}

	if dryRun {
		sum.Added++
		return
	}

	createdAt := time.Now().Unix()
	err = store.InsertAssetWithCamera(db, hashHex, relPath, captureUnix, createdAt, cam.Make, cam.Model)
	if err != nil {
		if isUniqueContentHash(err) {
			sum.SkippedDuplicate++
			return
		}
		if isUniqueRelPath(err) {
			sum.Failed++
			slog.Error("import: rel_path unique conflict after insert race", "rel_path", relPath, "err", err)
			return
		}
		sum.Failed++
		slog.Error("import: insert asset", "path", absPath, "rel_path", relPath, "err", err)
		return
	}

	sum.Added++
}

func isUniqueRelPath(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if !strings.Contains(msg, "UNIQUE constraint failed") {
		return false
	}
	return strings.Contains(msg, "rel_path") || strings.Contains(msg, "idx_assets_rel_path_active")
}
