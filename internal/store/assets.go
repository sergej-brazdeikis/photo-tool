package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// AssetRowByContentHash returns id, rel_path, and capture_time_unix for the row with the given hash.
func AssetRowByContentHash(db *sql.DB, contentHash string) (id int64, relPath string, captureUnix int64, ok bool, err error) {
	err = db.QueryRow(
		`SELECT id, rel_path, capture_time_unix FROM assets WHERE content_hash = ? LIMIT 1`,
		contentHash,
	).Scan(&id, &relPath, &captureUnix)
	if err == sql.ErrNoRows {
		return 0, "", 0, false, nil
	}
	if err != nil {
		return 0, "", 0, false, fmt.Errorf("asset row by content_hash: %w", err)
	}
	return id, relPath, captureUnix, true, nil
}

// AssetIDByContentHash returns the primary key for an asset with the given content hash, if any.
func AssetIDByContentHash(db *sql.DB, contentHash string) (id int64, ok bool, err error) {
	id, _, _, ok, err = AssetRowByContentHash(db, contentHash)
	return id, ok, err
}

// FindAssetByContentHash reports whether an asset row exists for the given SHA-256 hex digest.
func FindAssetByContentHash(db *sql.DB, contentHash string) (bool, error) {
	_, ok, err := AssetIDByContentHash(db, contentHash)
	return ok, err
}

// InsertAsset inserts a new active asset row. createdAtUnix is typically time.Now().Unix().
func InsertAsset(db *sql.DB, contentHash, relPath string, captureTimeUnix, createdAtUnix int64) error {
	_, err := InsertAssetWithCamera(db, contentHash, relPath, captureTimeUnix, createdAtUnix, "", "")
	return err
}

// InsertAssetWithCamera inserts a new active asset row with optional EXIF camera fields (Story 2.8).
// Empty strings after trim are stored as NULL; camera_label is derived via [CameraLabelFromParts].
// On success it returns the new row id (SQLite last_insert_rowid).
func InsertAssetWithCamera(db *sql.DB, contentHash, relPath string, captureTimeUnix, createdAtUnix int64, cameraMake, cameraModel string) (int64, error) {
	makeNS := sql.NullString{}
	if strings.TrimSpace(cameraMake) != "" {
		makeNS = sql.NullString{String: NormalizeCameraField(cameraMake), Valid: true}
	}
	modelNS := sql.NullString{}
	if strings.TrimSpace(cameraModel) != "" {
		modelNS = sql.NullString{String: NormalizeCameraField(cameraModel), Valid: true}
	}
	labelNS := CameraLabelForStorage(cameraMake, cameraModel)

	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, camera_make, camera_model, camera_label)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		contentHash, relPath, captureTimeUnix, createdAtUnix,
		ScanStringPtr(makeNS), ScanStringPtr(modelNS), ScanStringPtr(labelNS))
	if err != nil {
		return 0, fmt.Errorf("insert asset: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("insert asset last id: %w", err)
	}
	return id, nil
}

// ActiveAssetByRelPath returns the active (non-deleted) row at rel_path, if any.
func ActiveAssetByRelPath(db *sql.DB, relPath string) (id int64, contentHash string, captureUnix int64, ok bool, err error) {
	err = db.QueryRow(`
SELECT id, content_hash, capture_time_unix
FROM assets
WHERE rel_path = ? AND deleted_at_unix IS NULL
LIMIT 1`, relPath).Scan(&id, &contentHash, &captureUnix)
	if err == sql.ErrNoRows {
		return 0, "", 0, false, nil
	}
	if err != nil {
		return 0, "", 0, false, fmt.Errorf("active asset by rel_path: %w", err)
	}
	return id, contentHash, captureUnix, true, nil
}

// UpdateAssetCaptureTime sets capture_time_unix for an asset by primary key (metadata backfill).
func UpdateAssetCaptureTime(db *sql.DB, id int64, captureUnix int64) error {
	res, err := db.Exec(`UPDATE assets SET capture_time_unix = ? WHERE id = ? AND deleted_at_unix IS NULL`, captureUnix, id)
	if err != nil {
		return fmt.Errorf("update asset capture_time: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected capture_time: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("update asset capture_time: no row updated for id %d", id)
	}
	return nil
}

// UpdateAssetRating sets assets.rating for an active (non-deleted) row (Story 2.4).
// Rating must be 1..5 per migration CHECK; NULL clear is out of scope for this API.
func UpdateAssetRating(db *sql.DB, id int64, rating int) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("update asset rating: rating must be 1..5, got %d", rating)
	}
	res, err := db.Exec(`
UPDATE assets SET rating = ?
WHERE id = ? AND deleted_at_unix IS NULL`, rating, id)
	if err != nil {
		return fmt.Errorf("update asset rating: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update asset rating rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("update asset rating: no active row updated for id %d", id)
	}
	return nil
}
