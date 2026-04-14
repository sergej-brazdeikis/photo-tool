package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// AssetPrimaryPath resolves rel_path under libraryRoot and rejects path-escape attempts (Story 2.7 / 3.3).
func AssetPrimaryPath(libraryRoot, relPath string) (abs string, err error) {
	return assetPrimaryPath(libraryRoot, relPath)
}

// assetPrimaryPath resolves rel_path under libraryRoot and rejects path-escape attempts (Story 2.7 hardening).
func assetPrimaryPath(libraryRoot, relPath string) (abs string, err error) {
	root := filepath.Clean(libraryRoot)
	joined := filepath.Join(root, filepath.FromSlash(relPath))
	clean := filepath.Clean(joined)
	rel, err := filepath.Rel(root, clean)
	if err != nil {
		return "", fmt.Errorf("asset path: %w", err)
	}
	if rel == "." {
		return "", fmt.Errorf("asset path resolves to library root")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("asset path escapes library root")
	}
	return clean, nil
}

func quarantineDestinationPath(quarantineDir, base string) (string, error) {
	candidate := filepath.Join(quarantineDir, base)
	_, err := os.Stat(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return candidate, nil
		}
		return "", err
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 1; i < 10000; i++ {
		c := filepath.Join(quarantineDir, fmt.Sprintf("%s_%d%s", stem, i, ext))
		_, err := os.Stat(c)
		if err != nil {
			if os.IsNotExist(err) {
				slog.Warn("delete: quarantine basename collision, using suffix", "base", base, "chosen", filepath.Base(c))
				return c, nil
			}
			return "", err
		}
	}
	return "", fmt.Errorf("quarantine: no free filename for %q", base)
}

// DeleteAssetToTrash soft-deletes an asset and moves its primary file into {libraryRoot}/.trash/{id}/ (Story 2.7).
// Preconditions at the store layer: row exists, deleted_at_unix IS NULL; rejected may be 0 or 1.
// Ordering: mkdir quarantine → rename (if file exists) → UPDATE deleted_at_unix.
// Returns changed=false for unknown id, already-deleted rows, or idempotent recall.
func DeleteAssetToTrash(db *sql.DB, libraryRoot string, id int64, deletedAtUnix int64) (changed bool, err error) {
	var relPath string
	var deleted sql.NullInt64
	err = db.QueryRow(`SELECT rel_path, deleted_at_unix FROM assets WHERE id = ?`, id).Scan(&relPath, &deleted)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("delete asset select: %w", err)
	}
	if deleted.Valid {
		return false, nil
	}

	absSrc, err := assetPrimaryPath(libraryRoot, relPath)
	if err != nil {
		return false, fmt.Errorf("delete asset path: %w", err)
	}

	quarantineDir := filepath.Join(libraryRoot, ".trash", strconv.FormatInt(id, 10))
	if err := os.MkdirAll(quarantineDir, 0o755); err != nil {
		return false, fmt.Errorf("delete mkdir quarantine: %w", err)
	}

	base := filepath.Base(filepath.FromSlash(relPath))
	if relPath == "" || base == "." {
		return false, fmt.Errorf("delete: invalid rel_path %q", relPath)
	}

	var movedFile bool
	var quarantineFile string // set when a primary file was renamed into .trash (for split-brain logs)
	if _, statErr := os.Stat(absSrc); statErr != nil {
		if os.IsNotExist(statErr) {
			slog.Warn("delete: primary file missing, soft-delete only", "asset_id", id, "path", absSrc)
		} else {
			return false, fmt.Errorf("delete stat source: %w", statErr)
		}
	} else {
		dest, qerr := quarantineDestinationPath(quarantineDir, base)
		if qerr != nil {
			return false, qerr
		}
		if err := os.Rename(absSrc, dest); err != nil {
			return false, fmt.Errorf("delete quarantine rename: %w", err)
		}
		movedFile = true
		quarantineFile = dest
	}

	res, err := db.Exec(`
UPDATE assets SET deleted_at_unix = ?
WHERE id = ? AND deleted_at_unix IS NULL`, deletedAtUnix, id)
	if err != nil {
		if movedFile {
			slog.Error("delete: file moved to quarantine but DB update failed", "asset_id", id, "src", absSrc, "quarantine_file", quarantineFile, "quarantine_dir", quarantineDir)
		}
		return false, fmt.Errorf("delete asset update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete asset rows affected: %w", err)
	}
	if n == 0 {
		if movedFile {
			slog.Error("delete: quarantine rename succeeded but row not marked deleted", "asset_id", id, "quarantine_file", quarantineFile, "quarantine_dir", quarantineDir)
			return false, fmt.Errorf("delete: lost update for asset %d after quarantine (check .trash)", id)
		}
		return false, nil
	}
	return true, nil
}
