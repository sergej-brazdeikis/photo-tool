package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CollectionDetail is a collections row for edit surfaces (Story 2.9).
type CollectionDetail struct {
	ID            int64
	Name          string
	DisplayDate   string
	CreatedAtUnix int64
}

// ErrCollectionNotFound is returned when a collection id has no row (Story 2.8 AC9).
// Callers should use errors.Is rather than string-matching driver errors.
var ErrCollectionNotFound = errors.New("collection not found")

// CollectionRow is a lightweight row for UI lists (Story 2.2 filter strip).
type CollectionRow struct {
	ID   int64
	Name string
}

// CollectionAlbumListRow is a collection row plus an optional cover asset for image-forward album lists.
// CoverAssetID is zero when the album has no visible members ([ReviewBrowseBaseWhere]).
type CollectionAlbumListRow struct {
	ID               int64
	Name             string
	CoverAssetID     int64
	CoverRelPath     string
	CoverContentHash string
}

// ListCollections returns all collections sorted by name (case-insensitive).
func ListCollections(db *sql.DB) ([]CollectionRow, error) {
	rows, err := db.Query(`SELECT id, name FROM collections ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list collections: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []CollectionRow
	for rows.Next() {
		var r CollectionRow
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			return nil, fmt.Errorf("list collections scan: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list collections rows: %w", err)
	}
	return out, nil
}

// ListCollectionAlbumListRows returns every collection ordered by name with at most one cover asset each:
// the newest by capture_time_unix (then id) among members matching default Review visibility.
func ListCollectionAlbumListRows(db *sql.DB) ([]CollectionAlbumListRow, error) {
	q := `
WITH cover AS (
  SELECT ac.collection_id,
         a.id AS asset_id,
         a.rel_path AS rel_path,
         a.content_hash AS content_hash,
         ROW_NUMBER() OVER (
           PARTITION BY ac.collection_id
           ORDER BY a.capture_time_unix DESC, a.id DESC
         ) AS rn
  FROM asset_collections ac
  INNER JOIN assets a ON a.id = ac.asset_id
  WHERE ` + ReviewBrowseBaseWhere + `
)
SELECT c.id, c.name,
       cover.asset_id, cover.rel_path, cover.content_hash
FROM collections c
LEFT JOIN cover ON cover.collection_id = c.id AND cover.rn = 1
ORDER BY c.name COLLATE NOCASE`
	rows, err := db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("list collection album rows: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []CollectionAlbumListRow
	for rows.Next() {
		var r CollectionAlbumListRow
		var aid sql.NullInt64
		var relPath, hash sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &aid, &relPath, &hash); err != nil {
			return nil, fmt.Errorf("list collection album rows scan: %w", err)
		}
		if aid.Valid {
			r.CoverAssetID = aid.Int64
			r.CoverRelPath = relPath.String
			r.CoverContentHash = hash.String
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list collection album rows rows: %w", err)
	}
	return out, nil
}

// GetCollection loads name, display_date, and created_at_unix for id.
// Missing id returns an error wrapping [ErrCollectionNotFound].
func GetCollection(db *sql.DB, id int64) (CollectionDetail, error) {
	var d CollectionDetail
	err := db.QueryRow(`
SELECT id, name, display_date, created_at_unix FROM collections WHERE id = ?`, id).
		Scan(&d.ID, &d.Name, &d.DisplayDate, &d.CreatedAtUnix)
	if errors.Is(err, sql.ErrNoRows) {
		return CollectionDetail{}, fmt.Errorf("get collection %d: %w", id, ErrCollectionNotFound)
	}
	if err != nil {
		return CollectionDetail{}, fmt.Errorf("get collection: %w", err)
	}
	return d, nil
}

func normalizeUpdateCollectionFields(name string, displayDateISO string, createdAtUnix int64) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", fmt.Errorf("update collection: name is required")
	}
	dd := strings.TrimSpace(displayDateISO)
	if dd == "" {
		// AC2: cleared optional date → local calendar day of collection creation (created_at_unix), not previous display_date.
		dd = time.Unix(createdAtUnix, 0).In(time.Local).Format("2006-01-02")
	} else {
		if _, err := time.ParseInLocation("2006-01-02", dd, time.Local); err != nil {
			return "", "", fmt.Errorf("update collection: display date must be YYYY-MM-DD: %w", err)
		}
	}
	return name, dd, nil
}

// UpdateCollection sets name and display_date after validation. name must be non-empty (trim).
// displayDateISO is optional: empty after trim recomputes YYYY-MM-DD from the row's created_at_unix in local TZ (AC2).
// Non-empty displayDateISO must parse as a calendar ISO date in local TZ. Missing collection id wraps [ErrCollectionNotFound].
func UpdateCollection(db *sql.DB, id int64, name string, displayDateISO string) error {
	var createdAtUnix int64
	err := db.QueryRow(`SELECT created_at_unix FROM collections WHERE id = ?`, id).Scan(&createdAtUnix)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("update collection %d: %w", id, ErrCollectionNotFound)
	}
	if err != nil {
		return fmt.Errorf("update collection: %w", err)
	}
	name, dd, err := normalizeUpdateCollectionFields(name, displayDateISO, createdAtUnix)
	if err != nil {
		return err
	}
	res, err := db.Exec(`UPDATE collections SET name = ?, display_date = ? WHERE id = ?`, name, dd, id)
	if err != nil {
		return fmt.Errorf("update collection %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update collection %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("update collection %d: %w", id, ErrCollectionNotFound)
	}
	return nil
}

// ListCollectionIDsForAsset returns collection ids the asset belongs to, stable order: name COLLATE NOCASE, then id.
func ListCollectionIDsForAsset(db *sql.DB, assetID int64) ([]int64, error) {
	rows, err := db.Query(`
SELECT c.id FROM collections c
INNER JOIN asset_collections ac ON ac.collection_id = c.id AND ac.asset_id = ?
ORDER BY c.name COLLATE NOCASE, c.id`, assetID)
	if err != nil {
		return nil, fmt.Errorf("list collection ids for asset: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("list collection ids for asset scan: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list collection ids for asset rows: %w", err)
	}
	return out, nil
}

// UnlinkAssetFromCollection deletes the asset_collections row if it exists.
// Idempotency: no matching row is success (no error); only real DB failures are returned.
// Missing collection or asset FK does not apply — DELETE simply affects zero rows.
func UnlinkAssetFromCollection(db *sql.DB, assetID int64, collectionID int64) error {
	_, err := db.Exec(`DELETE FROM asset_collections WHERE asset_id = ? AND collection_id = ?`, assetID, collectionID)
	if err != nil {
		return fmt.Errorf("unlink asset %d from collection %d: %w", assetID, collectionID, err)
	}
	return nil
}

func normalizeNewCollectionFields(name string, displayDateISO string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", fmt.Errorf("create collection: name is required")
	}
	dd := strings.TrimSpace(displayDateISO)
	if dd == "" {
		dd = time.Now().In(time.Local).Format("2006-01-02")
	} else {
		if _, err := time.ParseInLocation("2006-01-02", dd, time.Local); err != nil {
			return "", "", fmt.Errorf("create collection: display date must be YYYY-MM-DD: %w", err)
		}
	}
	return name, dd, nil
}

// CreateCollection inserts a row into collections. name must be non-empty (after trim).
// displayDateISO is optional: when empty, the default is today's local calendar date as
// ISO TEXT "YYYY-MM-DD" (FR-18 calendar semantics; not capture time). Non-empty values
// must parse as that same calendar date in [time.Local]; invalid strings return an error.
func CreateCollection(db *sql.DB, name string, displayDateISO string) (int64, error) {
	name, dd, err := normalizeNewCollectionFields(name, displayDateISO)
	if err != nil {
		return 0, err
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

// CreateCollectionAndLinkAssets inserts a collection and links assets in one transaction.
// If any link fails (e.g. FK), the collection insert is rolled back — no orphan row (FR-19 loupe flow; Story 2.9).
// Empty assetIDs is allowed (same as CreateCollection + no links).
func CreateCollectionAndLinkAssets(db *sql.DB, name string, displayDateISO string, assetIDs []int64) (int64, error) {
	name, dd, err := normalizeNewCollectionFields(name, displayDateISO)
	if err != nil {
		return 0, err
	}
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("create collection: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	created := time.Now().Unix()
	res, err := tx.Exec(`
INSERT INTO collections (name, display_date, created_at_unix) VALUES (?, ?, ?)`,
		name, dd, created)
	if err != nil {
		return 0, fmt.Errorf("create collection: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("create collection: last insert id: %w", err)
	}
	if err := linkAssetsToCollectionTx(tx, id, assetIDs); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("create collection: %w", err)
	}
	return id, nil
}

func linkAssetsToCollectionTx(tx *sql.Tx, collectionID int64, assetIDs []int64) error {
	if len(assetIDs) == 0 {
		return nil
	}
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
	return nil
}

// LinkAssetsToCollection creates asset_collections rows for each assetID in one transaction.
// Idempotency: duplicate (asset_id, collection_id) pairs are skipped via ON CONFLICT DO NOTHING,
// including when the same asset ID appears multiple times in assetIDs.
// If any insert fails (e.g. FK), the transaction rolls back — no partial links.
// Invalid asset_id or collection_id yields a foreign-key error wrapped with context.
// An empty assetIDs slice is a no-op (no transaction opened); collection existence is not checked.
func LinkAssetsToCollection(db *sql.DB, collectionID int64, assetIDs []int64) error {
	if len(assetIDs) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("link assets to collection: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := linkAssetsToCollectionTx(tx, collectionID, assetIDs); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("link assets to collection: %w", err)
	}
	return nil
}

// DeleteCollection deletes the collection row. Referencing asset_collections rows are removed
// by foreign-key ON DELETE CASCADE, satisfying FR-20 (no orphaned membership).
// If the id is missing (never existed or already deleted), wraps [ErrCollectionNotFound].
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
		return fmt.Errorf("delete collection %d: %w", collectionID, ErrCollectionNotFound)
	}
	return nil
}
