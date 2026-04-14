package store

import (
	"database/sql"
	"fmt"
	"strings"

	"photo-tool/internal/domain"
)

// ReviewGridRow is the minimal row shape for Review thumbnail cells (Story 2.3).
// Predicate and sort match ListAssetsForReview / CountAssetsForReview contracts.
type ReviewGridRow struct {
	ID              int64
	RelPath         string
	ContentHash     string
	CaptureTimeUnix int64
	Rejected        int
	Rating          *int
	Mime            string
	Width           int
	Height          int
}

// ReviewBrowseBaseWhere is the shared SQL boolean expression for default Review asset visibility
// (rejected hidden, soft-deletes excluded; architecture §4.5). Count and list queries MUST use
// this same base predicate so Story 2.3 cannot drift from CountAssetsForReview.
const ReviewBrowseBaseWhere = `rejected = 0 AND deleted_at_unix IS NULL`

// ReviewRejectedBaseWhere is the shared SQL boolean for the Rejected/Hidden bucket (Story 2.6).
// Count/list for that surface MUST compose ReviewFilterWhereSuffix after this base so collection,
// min-rating, and tag filters behave like default Review (architecture §3.4).
const ReviewRejectedBaseWhere = `rejected = 1 AND deleted_at_unix IS NULL`

// ReviewFilterWhereSuffix returns AND … fragments and bound args for collection + min-rating filters.
// Story 2.3 list queries MUST compose the same suffix after the same base WHERE (rejected/deleted).
func ReviewFilterWhereSuffix(f domain.ReviewFilters) (suffix string, args []any, err error) {
	if err := f.Validate(); err != nil {
		return "", nil, err
	}
	args = []any{}
	if f.CollectionID != nil {
		suffix += `
 AND EXISTS (
    SELECT 1 FROM asset_collections ac
    WHERE ac.asset_id = assets.id AND ac.collection_id = ?
  )`
		args = append(args, *f.CollectionID)
	}
	if f.MinRating != nil {
		suffix += ` AND rating IS NOT NULL AND rating >= ?`
		args = append(args, *f.MinRating)
	}
	if f.TagID != nil {
		suffix += `
 AND EXISTS (
    SELECT 1 FROM asset_tags at
    WHERE at.asset_id = assets.id AND at.tag_id = ?
  )`
		args = append(args, *f.TagID)
	}
	return suffix, args, nil
}

// CountAssetsForReview returns how many active, non-rejected assets match the filter (Story 2.2).
// Predicate matches default browse rules: rejected = 0, deleted_at_unix IS NULL (architecture §4.5).
func CountAssetsForReview(db *sql.DB, f domain.ReviewFilters) (int64, error) {
	suffix, args, err := ReviewFilterWhereSuffix(f)
	if err != nil {
		return 0, err
	}
	q := `
SELECT COUNT(*) FROM assets
WHERE ` + ReviewBrowseBaseWhere + suffix

	var n int64
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count assets for review: %w", err)
	}
	return n, nil
}

// ListAssetsForReview returns a page of assets for the Review grid using the same
// WHERE clause and filter argument order as CountAssetsForReview (Story 2.3).
func ListAssetsForReview(db *sql.DB, f domain.ReviewFilters, limit, offset int) ([]ReviewGridRow, error) {
	if limit < 1 {
		return nil, fmt.Errorf("list assets for review: limit must be >= 1")
	}
	if offset < 0 {
		return nil, fmt.Errorf("list assets for review: offset must be >= 0")
	}
	suffix, args, err := ReviewFilterWhereSuffix(f)
	if err != nil {
		return nil, err
	}
	q := `
SELECT id, rel_path, content_hash, capture_time_unix, rejected, rating, mime, width, height
FROM assets
WHERE ` + ReviewBrowseBaseWhere + suffix + `
ORDER BY capture_time_unix DESC, id DESC
LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list assets for review: %w", err)
	}
	defer func() { _ = rows.Close() }()

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
			return nil, fmt.Errorf("scan review grid row: %w", err)
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
		return nil, fmt.Errorf("list assets for review: %w", err)
	}
	return out, nil
}

// ListAssetIDsForReview returns every asset id matching the Review grid predicate for f (Story 4.1 share filtered set).
func ListAssetIDsForReview(db *sql.DB, f domain.ReviewFilters) ([]int64, error) {
	suffix, args, err := ReviewFilterWhereSuffix(f)
	if err != nil {
		return nil, err
	}
	q := `
SELECT id FROM assets
WHERE ` + ReviewBrowseBaseWhere + suffix + `
ORDER BY capture_time_unix DESC, id DESC`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list asset ids for review: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("list asset ids for review scan: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list asset ids for review: %w", err)
	}
	return out, nil
}

// ListReviewGridRowsByIDsInOrder loads grid rows for ids in stable first-seen order (Story 4.1 package preview).
func ListReviewGridRowsByIDsInOrder(db *sql.DB, orderedIDs []int64) ([]ReviewGridRow, error) {
	order := domain.StableDedupeAssetIDs(orderedIDs)
	if len(order) == 0 {
		return nil, nil
	}
	m := make(map[int64]ReviewGridRow, len(order))
	const chunk = 400
	for i := 0; i < len(order); i += chunk {
		end := i + chunk
		if end > len(order) {
			end = len(order)
		}
		part := order[i:end]
		placeholders := strings.Repeat("?,", len(part))
		placeholders = strings.TrimSuffix(placeholders, ",")
		args := make([]any, 0, len(part))
		for _, id := range part {
			args = append(args, id)
		}
		q := `
SELECT id, rel_path, content_hash, capture_time_unix, rejected, rating, mime, width, height
FROM assets
WHERE id IN (` + placeholders + `)`
		rows, err := db.Query(q, args...)
		if err != nil {
			return nil, fmt.Errorf("list review rows by id: %w", err)
		}
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
				_ = rows.Close()
				return nil, fmt.Errorf("scan review grid row by id: %w", err)
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
			m[r.ID] = r
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}
	var out []ReviewGridRow
	for _, id := range order {
		if r, ok := m[id]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}

// CountRejectedForReview counts rejected, non-deleted assets matching the same filter suffix as
// default Review (Story 2.6). Predicate base is ReviewRejectedBaseWhere.
func CountRejectedForReview(db *sql.DB, f domain.ReviewFilters) (int64, error) {
	suffix, args, err := ReviewFilterWhereSuffix(f)
	if err != nil {
		return 0, err
	}
	q := `
SELECT COUNT(*) FROM assets
WHERE ` + ReviewRejectedBaseWhere + suffix

	var n int64
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count rejected for review: %w", err)
	}
	return n, nil
}

// ListRejectedForReview lists a page of rejected assets using the same filter suffix and sort as
// ListAssetsForReview (Story 2.6).
func ListRejectedForReview(db *sql.DB, f domain.ReviewFilters, limit, offset int) ([]ReviewGridRow, error) {
	if limit < 1 {
		return nil, fmt.Errorf("list rejected for review: limit must be >= 1")
	}
	if offset < 0 {
		return nil, fmt.Errorf("list rejected for review: offset must be >= 0")
	}
	suffix, args, err := ReviewFilterWhereSuffix(f)
	if err != nil {
		return nil, err
	}
	q := `
SELECT id, rel_path, content_hash, capture_time_unix, rejected, rating, mime, width, height
FROM assets
WHERE ` + ReviewRejectedBaseWhere + suffix + `
ORDER BY capture_time_unix DESC, id DESC
LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list rejected for review: %w", err)
	}
	defer func() { _ = rows.Close() }()

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
			return nil, fmt.Errorf("scan rejected grid row: %w", err)
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
		return nil, fmt.Errorf("list rejected for review: %w", err)
	}
	return out, nil
}
