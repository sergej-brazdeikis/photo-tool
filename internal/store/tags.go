package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// TagRow is a row from tags (stable id for filters — AC6).
type TagRow struct {
	ID    int64
	Label string
}

// tagBulkChunk is the max asset ids committed per transaction for bulk link/unlink (SQLite-friendly).
// Tests may temporarily lower it; production default is 500.
var tagBulkChunk = 500

// NormalizeTagLabel trims surrounding space, collapses internal whitespace runs to a single ASCII space
// (via strings.Fields), and returns an error if the result is empty.
func NormalizeTagLabel(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("tag label is empty")
	}
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "", fmt.Errorf("tag label is empty")
	}
	return s, nil
}

// ListTags returns all tag rows sorted by label ascending (case-insensitive).
func ListTags(db *sql.DB) ([]TagRow, error) {
	rows, err := db.Query(`
SELECT id, label FROM tags ORDER BY label COLLATE NOCASE ASC`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []TagRow
	for rows.Next() {
		var r TagRow
		if err := rows.Scan(&r.ID, &r.Label); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return out, nil
}

// FindTagByLabel returns the tag id if a row exists for the normalized label.
func FindTagByLabel(db *sql.DB, rawLabel string) (id int64, ok bool, err error) {
	label, err := NormalizeTagLabel(rawLabel)
	if err != nil {
		return 0, false, err
	}
	err = db.QueryRow(`SELECT id FROM tags WHERE label = ? COLLATE NOCASE LIMIT 1`, label).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("find tag by label: %w", err)
	}
	return id, true, nil
}

// FindOrCreateTagByLabel normalizes label, then returns the tag id (inserting if needed).
func FindOrCreateTagByLabel(db *sql.DB, rawLabel string) (int64, error) {
	label, err := NormalizeTagLabel(rawLabel)
	if err != nil {
		return 0, err
	}
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("find or create tag: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`INSERT OR IGNORE INTO tags (label) VALUES (?)`, label); err != nil {
		return 0, fmt.Errorf("find or create tag insert: %w", err)
	}
	var id int64
	err = tx.QueryRow(`SELECT id FROM tags WHERE label = ? COLLATE NOCASE LIMIT 1`, label).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("find or create tag select: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("find or create tag commit: %w", err)
	}
	return id, nil
}

func dedupeAssetIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// LinkTagToAssets adds the tag to each asset if not already linked (idempotent).
// Large batches commit one SQLite transaction per chunk (see tagBulkChunk). Earlier chunks stay committed
// if a later chunk fails — returned error includes how many leading assets were already processed when applicable.
func LinkTagToAssets(db *sql.DB, tagID int64, assetIDs []int64) error {
	if tagID <= 0 {
		return fmt.Errorf("link tag to assets: invalid tag id %d", tagID)
	}
	ids := dedupeAssetIDs(assetIDs)
	if len(ids) == 0 {
		return nil
	}
	for i := 0; i < len(ids); i += tagBulkChunk {
		end := i + tagBulkChunk
		if end > len(ids) {
			end = len(ids)
		}
		if err := linkTagToAssetsChunk(db, tagID, ids[i:end]); err != nil {
			slog.Error("link tag to assets chunk failed", "chunkStart", i, "err", err)
			if i > 0 {
				return fmt.Errorf("bulk tag add: first %d selected assets were updated; remaining assets unchanged: %w", i, err)
			}
			return err
		}
	}
	return nil
}

func linkTagToAssetsChunk(db *sql.DB, tagID int64, assetIDs []int64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("link tag to assets: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO asset_tags (asset_id, tag_id) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("link tag to assets prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, aid := range assetIDs {
		if _, err := stmt.Exec(aid, tagID); err != nil {
			return fmt.Errorf("link tag to assets exec: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("link tag to assets commit: %w", err)
	}
	return nil
}

// UnlinkTagFromAssets removes the tag link from each listed asset (idempotent).
// Chunking semantics match LinkTagToAssets.
func UnlinkTagFromAssets(db *sql.DB, tagID int64, assetIDs []int64) error {
	if tagID <= 0 {
		return fmt.Errorf("unlink tag from assets: invalid tag id %d", tagID)
	}
	ids := dedupeAssetIDs(assetIDs)
	if len(ids) == 0 {
		return nil
	}
	for i := 0; i < len(ids); i += tagBulkChunk {
		end := i + tagBulkChunk
		if end > len(ids) {
			end = len(ids)
		}
		if err := unlinkTagFromAssetsChunk(db, tagID, ids[i:end]); err != nil {
			slog.Error("unlink tag from assets chunk failed", "chunkStart", i, "err", err)
			if i > 0 {
				return fmt.Errorf("bulk tag remove: first %d selected assets were updated; remaining assets unchanged: %w", i, err)
			}
			return err
		}
	}
	return nil
}

func unlinkTagFromAssetsChunk(db *sql.DB, tagID int64, assetIDs []int64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("unlink tag from assets: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`DELETE FROM asset_tags WHERE asset_id = ? AND tag_id = ?`)
	if err != nil {
		return fmt.Errorf("unlink tag from assets prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for _, aid := range assetIDs {
		if _, err := stmt.Exec(aid, tagID); err != nil {
			return fmt.Errorf("unlink tag from assets exec: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("unlink tag from assets commit: %w", err)
	}
	return nil
}

// ListTagsUnionForAssets returns distinct tags linked to any of the given assets, sorted by label.
func ListTagsUnionForAssets(db *sql.DB, assetIDs []int64) ([]TagRow, error) {
	ids := dedupeAssetIDs(assetIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	q := `
SELECT DISTINCT t.id, t.label
FROM tags t
JOIN asset_tags at ON at.tag_id = t.id
WHERE at.asset_id IN (` + placeholders + `)
ORDER BY t.label COLLATE NOCASE ASC`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list tags union for assets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []TagRow
	for rows.Next() {
		var r TagRow
		if err := rows.Scan(&r.ID, &r.Label); err != nil {
			return nil, fmt.Errorf("scan tag union: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tags union for assets: %w", err)
	}
	return out, nil
}

// ForeignKeyCheck runs PRAGMA foreign_key_check (empty result = ok).
func ForeignKeyCheck(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("foreign_key_check: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		return fmt.Errorf("foreign_key_check: violations present")
	}
	return rows.Err()
}
