package store

import (
	"database/sql"
	"fmt"
)

// AssetIDByContentHash returns the primary key for an asset with the given content hash, if any.
func AssetIDByContentHash(db *sql.DB, contentHash string) (id int64, ok bool, err error) {
	err = db.QueryRow(`SELECT id FROM assets WHERE content_hash = ? LIMIT 1`, contentHash).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("asset id by content_hash: %w", err)
	}
	return id, true, nil
}

// FindAssetByContentHash reports whether an asset row exists for the given SHA-256 hex digest.
func FindAssetByContentHash(db *sql.DB, contentHash string) (bool, error) {
	_, ok, err := AssetIDByContentHash(db, contentHash)
	return ok, err
}

// InsertAsset inserts a new active asset row. createdAtUnix is typically time.Now().Unix().
func InsertAsset(db *sql.DB, contentHash, relPath string, captureTimeUnix, createdAtUnix int64) error {
	_, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix)
VALUES (?, ?, ?, ?)`,
		contentHash, relPath, captureTimeUnix, createdAtUnix)
	if err != nil {
		return fmt.Errorf("insert asset: %w", err)
	}
	return nil
}
