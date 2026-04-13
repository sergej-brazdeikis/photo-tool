package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// CreateCollection inserts a row into collections. name must be non-empty (after trim).
// displayDateISO is optional: when empty, the default is today's local calendar date as
// ISO TEXT "YYYY-MM-DD" (FR-18 calendar semantics; not capture time). Non-empty values
// are stored as-is after trim.
func CreateCollection(db *sql.DB, name string, displayDateISO string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("create collection: name is required")
	}
	dd := strings.TrimSpace(displayDateISO)
	if dd == "" {
		dd = time.Now().In(time.Local).Format("2006-01-02")
	}
	created := time.Now().Unix()
	res, err := db.Exec(`
INSERT INTO collections (name, display_date, created_at_unix) VALUES (?, ?, ?)`,
		name, dd, created)
	if err != nil {
		return 0, fmt.Errorf("create collection: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("create collection: last insert id: %w", err)
	}
	return id, nil
}

// LinkAssetsToCollection creates asset_collections rows for each assetID.
// Idempotency: duplicate (asset_id, collection_id) pairs are skipped via ON CONFLICT DO NOTHING.
// Invalid asset_id or collection_id yields a foreign-key error wrapped with context.
func LinkAssetsToCollection(db *sql.DB, collectionID int64, assetIDs []int64) error {
	if len(assetIDs) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("link assets to collection: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
INSERT INTO asset_collections (asset_id, collection_id, created_at_unix) VALUES (?, ?, ?)
ON CONFLICT(asset_id, collection_id) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("link assets to collection: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	now := time.Now().Unix()
	for _, aid := range assetIDs {
		if _, err := stmt.Exec(aid, collectionID, now); err != nil {
			return fmt.Errorf("link asset %d to collection %d: %w", aid, collectionID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("link assets to collection: %w", err)
	}
	return nil
}

// DeleteCollection removes the collection and all asset_collections rows for it (FR-20).
// Junction rows are removed via ON DELETE CASCADE on asset_collections.collection_id.
func DeleteCollection(db *sql.DB, collectionID int64) error {
	res, err := db.Exec(`DELETE FROM collections WHERE id = ?`, collectionID)
	if err != nil {
		return fmt.Errorf("delete collection %d: %w", collectionID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete collection %d: %w", collectionID, err)
	}
	if n == 0 {
		return fmt.Errorf("delete collection: no row with id %d", collectionID)
	}
	return nil
}
