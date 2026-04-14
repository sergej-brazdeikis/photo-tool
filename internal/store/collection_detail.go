package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// Collection paging strategy (Story 2.8 AC10): **(B) per-section LIMIT/OFFSET**.
// Each grouping bucket (star rating, calendar day, or camera_label, including unknown) is queried independently
// with its own ORDER BY capture_time_unix DESC, id DESC. Empty buckets are omitted at the summary layer, so
// pagination never slices across section headers for a different key.
//
// **Scroll / cap contract (Epic 2.10 prep):** the UI gives each section its own widget.List; scrolling that list
// loads additional rows using that section’s OFFSET chain only. There is no global cursor across sections—each
// section’s “load more” is bounded by its bucket COUNT and the same SQL LIMIT as the Review thumbnail grid.

// CollectionExists reports whether a collections row exists for id.
func CollectionExists(db *sql.DB, id int64) (bool, error) {
	var one int
	err := db.QueryRow(`SELECT 1 FROM collections WHERE id = ? LIMIT 1`, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("collection exists: %w", err)
	}
	return true, nil
}

func requireCollectionRow(db *sql.DB, collectionID int64) error {
	ok, err := CollectionExists(db, collectionID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCollectionNotFound
	}
	return nil
}

// CountCollectionVisibleAssets counts assets in the collection matching [ReviewBrowseBaseWhere].
func CountCollectionVisibleAssets(db *sql.DB, collectionID int64) (int64, error) {
	if err := requireCollectionRow(db, collectionID); err != nil {
		return 0, err
	}
	q := `
SELECT COUNT(*) FROM assets
WHERE ` + ReviewBrowseBaseWhere + `
AND EXISTS (
  SELECT 1 FROM asset_collections ac
  WHERE ac.asset_id = assets.id AND ac.collection_id = ?
)`
	var n int64
	if err := db.QueryRow(q, collectionID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count collection assets: %w", err)
	}
	return n, nil
}

// StarSection describes one non-empty star bucket for collection detail (Story 2.8 AC3).
type StarSection struct {
	Rating *int // nil means unrated
	Count  int64
}

// ListCollectionStarSections returns star buckets 5→1 then unrated when non-empty.
func ListCollectionStarSections(db *sql.DB, collectionID int64) ([]StarSection, error) {
	if err := requireCollectionRow(db, collectionID); err != nil {
		return nil, err
	}
	q := `
SELECT rating, COUNT(*) AS c
FROM assets
WHERE ` + ReviewBrowseBaseWhere + `
AND EXISTS (
  SELECT 1 FROM asset_collections ac
  WHERE ac.asset_id = assets.id AND ac.collection_id = ?
)
GROUP BY rating
ORDER BY rating IS NULL ASC, rating DESC`
	rows, err := db.Query(q, collectionID)
	if err != nil {
		return nil, fmt.Errorf("list collection star sections: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var byRated [6]int64 // index 1..5
	var unrated int64
	for rows.Next() {
		var rating sql.NullInt64
		var c int64
		if err := rows.Scan(&rating, &c); err != nil {
			return nil, fmt.Errorf("scan star section: %w", err)
		}
		if !rating.Valid {
			unrated += c
			continue
		}
		v := int(rating.Int64)
		if v >= 1 && v <= 5 {
			byRated[v] += c
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list collection star sections rows: %w", err)
	}

	var out []StarSection
	for stars := 5; stars >= 1; stars-- {
		if byRated[stars] == 0 {
			continue
		}
		s := stars
		out = append(out, StarSection{Rating: &s, Count: byRated[stars]})
	}
	if unrated > 0 {
		out = append(out, StarSection{Rating: nil, Count: unrated})
	}
	return out, nil
}

// ListCollectionStarSectionPage lists one page inside a star bucket using per-section OFFSET (AC10 strategy B).
func ListCollectionStarSectionPage(db *sql.DB, collectionID int64, rating *int, limit, offset int) ([]ReviewGridRow, error) {
	if err := requireCollectionRow(db, collectionID); err != nil {
		return nil, err
	}
	if limit < 1 {
		return nil, fmt.Errorf("list collection star section page: limit must be >= 1")
	}
	if offset < 0 {
		return nil, fmt.Errorf("list collection star section page: offset must be >= 0")
	}
	var q string
	var args []any
	base := `
SELECT id, rel_path, content_hash, capture_time_unix, rejected, rating, mime, width, height
FROM assets
WHERE ` + ReviewBrowseBaseWhere + `
AND EXISTS (
  SELECT 1 FROM asset_collections ac
  WHERE ac.asset_id = assets.id AND ac.collection_id = ?
)`
	if rating == nil {
		q = base + ` AND rating IS NULL
ORDER BY capture_time_unix DESC, id DESC
LIMIT ? OFFSET ?`
		args = []any{collectionID, limit, offset}
	} else {
		q = base + ` AND rating = ?
ORDER BY capture_time_unix DESC, id DESC
LIMIT ? OFFSET ?`
		args = []any{collectionID, *rating, limit, offset}
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list collection star section page: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanReviewGridRows(rows)
}

// DaySection is one local-calendar day bucket (Story 2.8 AC4). DayKey is YYYY-MM-DD in time.Local wall time.
type DaySection struct {
	DayKey string
	Count  int64
}

// ListCollectionDaySections returns non-empty local calendar days newest-first.
func ListCollectionDaySections(db *sql.DB, collectionID int64) ([]DaySection, error) {
	if err := requireCollectionRow(db, collectionID); err != nil {
		return nil, err
	}
	// Calendar day rule (Story 2.8 AC4): partition by local wall date from Unix epoch seconds, aligned with
	// SQLite's 'unixepoch' + 'localtime' (same intent as time.Local for operator-visible dates elsewhere).
	q := `
SELECT strftime('%Y-%m-%d', capture_time_unix, 'unixepoch', 'localtime') AS d, COUNT(*) AS c
FROM assets
WHERE ` + ReviewBrowseBaseWhere + `
AND EXISTS (
  SELECT 1 FROM asset_collections ac
  WHERE ac.asset_id = assets.id AND ac.collection_id = ?
)
GROUP BY d
HAVING d IS NOT NULL AND d != ''
ORDER BY d DESC`
	rows, err := db.Query(q, collectionID)
	if err != nil {
		return nil, fmt.Errorf("list collection day sections: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []DaySection
	for rows.Next() {
		var d string
		var c int64
		if err := rows.Scan(&d, &c); err != nil {
			return nil, fmt.Errorf("scan day section: %w", err)
		}
		if c == 0 {
			continue
		}
		out = append(out, DaySection{DayKey: d, Count: c})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list collection day sections rows: %w", err)
	}
	return out, nil
}

// ListCollectionDaySectionPage lists assets for one local calendar day (per-section paging).
func ListCollectionDaySectionPage(db *sql.DB, collectionID int64, dayKey string, limit, offset int) ([]ReviewGridRow, error) {
	if err := requireCollectionRow(db, collectionID); err != nil {
		return nil, err
	}
	if limit < 1 {
		return nil, fmt.Errorf("list collection day section page: limit must be >= 1")
	}
	if offset < 0 {
		return nil, fmt.Errorf("list collection day section page: offset must be >= 0")
	}
	q := `
SELECT id, rel_path, content_hash, capture_time_unix, rejected, rating, mime, width, height
FROM assets
WHERE ` + ReviewBrowseBaseWhere + `
AND EXISTS (
  SELECT 1 FROM asset_collections ac
  WHERE ac.asset_id = assets.id AND ac.collection_id = ?
)
AND strftime('%Y-%m-%d', capture_time_unix, 'unixepoch', 'localtime') = ?
ORDER BY capture_time_unix DESC, id DESC
LIMIT ? OFFSET ?`
	rows, err := db.Query(q, collectionID, dayKey, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list collection day section page: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanReviewGridRows(rows)
}

// CameraSection is one camera_label bucket; Label nil means unknown (NULL camera_label).
type CameraSection struct {
	Label *string
	Count int64
}

// ListCollectionCameraSections returns non-empty camera buckets; unknown (NULL label) last.
func ListCollectionCameraSections(db *sql.DB, collectionID int64) ([]CameraSection, error) {
	if err := requireCollectionRow(db, collectionID); err != nil {
		return nil, err
	}
	q := `
SELECT camera_label, COUNT(*) AS c
FROM assets
WHERE ` + ReviewBrowseBaseWhere + `
AND EXISTS (
  SELECT 1 FROM asset_collections ac
  WHERE ac.asset_id = assets.id AND ac.collection_id = ?
)
GROUP BY camera_label
ORDER BY (camera_label IS NOT NULL) DESC, camera_label COLLATE NOCASE ASC`
	rows, err := db.Query(q, collectionID)
	if err != nil {
		return nil, fmt.Errorf("list collection camera sections: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []CameraSection
	for rows.Next() {
		var lbl sql.NullString
		var c int64
		if err := rows.Scan(&lbl, &c); err != nil {
			return nil, fmt.Errorf("scan camera section: %w", err)
		}
		if c == 0 {
			continue
		}
		if !lbl.Valid {
			out = append(out, CameraSection{Label: nil, Count: c})
			continue
		}
		s := lbl.String
		out = append(out, CameraSection{Label: &s, Count: c})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list collection camera sections rows: %w", err)
	}
	return out, nil
}

// ListCollectionCameraSectionPage lists assets for one camera_label bucket (per-section paging).
func ListCollectionCameraSectionPage(db *sql.DB, collectionID int64, label *string, limit, offset int) ([]ReviewGridRow, error) {
	if err := requireCollectionRow(db, collectionID); err != nil {
		return nil, err
	}
	if limit < 1 {
		return nil, fmt.Errorf("list collection camera section page: limit must be >= 1")
	}
	if offset < 0 {
		return nil, fmt.Errorf("list collection camera section page: offset must be >= 0")
	}
	var q string
	var args []any
	base := `
SELECT id, rel_path, content_hash, capture_time_unix, rejected, rating, mime, width, height
FROM assets
WHERE ` + ReviewBrowseBaseWhere + `
AND EXISTS (
  SELECT 1 FROM asset_collections ac
  WHERE ac.asset_id = assets.id AND ac.collection_id = ?
)`
	if label == nil {
		q = base + ` AND camera_label IS NULL
ORDER BY capture_time_unix DESC, id DESC
LIMIT ? OFFSET ?`
		args = []any{collectionID, limit, offset}
	} else {
		q = base + ` AND camera_label = ?
ORDER BY capture_time_unix DESC, id DESC
LIMIT ? OFFSET ?`
		args = []any{collectionID, *label, limit, offset}
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list collection camera section page: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanReviewGridRows(rows)
}

func scanReviewGridRows(rows *sql.Rows) ([]ReviewGridRow, error) {
	var out []ReviewGridRow
	for rows.Next() {
		var r ReviewGridRow
		var rating sql.NullInt64
		var mime sql.NullString
		var width, height sql.NullInt64
		if err := rows.Scan(
			&r.ID,
			&r.RelPath,
			&r.ContentHash,
			&r.CaptureTimeUnix,
			&r.Rejected,
			&rating,
			&mime,
			&width,
			&height,
		); err != nil {
			return nil, fmt.Errorf("scan collection grid row: %w", err)
		}
		if rating.Valid {
			v := int(rating.Int64)
			r.Rating = &v
		}
		if mime.Valid {
			r.Mime = mime.String
		}
		if width.Valid {
			r.Width = int(width.Int64)
		}
		if height.Valid {
			r.Height = int(height.Int64)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("collection grid rows: %w", err)
	}
	return out, nil
}
