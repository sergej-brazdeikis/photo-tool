package store

import (
	"database/sql"
	"fmt"
)

// RejectAsset sets rejected=1 and rejected_at_unix for an active, non-rejected row (Story 2.6).
// Returns changed=false when the row is missing, soft-deleted, or already rejected (idempotent no-op).
func RejectAsset(db *sql.DB, id int64, rejectedAtUnix int64) (changed bool, err error) {
	res, err := db.Exec(`
UPDATE assets SET rejected = 1, rejected_at_unix = ?
WHERE id = ? AND deleted_at_unix IS NULL AND rejected = 0`, rejectedAtUnix, id)
	if err != nil {
		return false, fmt.Errorf("reject asset: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("reject asset rows affected: %w", err)
	}
	return n > 0, nil
}

// RestoreAsset clears reject flags for an active, rejected row (Story 2.6).
// Returns changed=false when the row is missing, soft-deleted, or not rejected (idempotent no-op).
func RestoreAsset(db *sql.DB, id int64) (changed bool, err error) {
	res, err := db.Exec(`
UPDATE assets SET rejected = 0, rejected_at_unix = NULL
WHERE id = ? AND deleted_at_unix IS NULL AND rejected = 1`, id)
	if err != nil {
		return false, fmt.Errorf("restore asset: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("restore asset rows affected: %w", err)
	}
	return n > 0, nil
}

// AssetEligibleForDefaultShare reports whether an asset may participate in the default share mint
// (Epic 3). Rejected and soft-deleted rows must be excluded per FR-29 / architecture §3.4.
// MintDefaultShareLink re-checks eligibility inside its transaction so the choke point cannot be bypassed.
func AssetEligibleForDefaultShare(db *sql.DB, id int64) (ok bool, err error) {
	var rejected int
	var deleted sql.NullInt64
	err = db.QueryRow(`
SELECT rejected, deleted_at_unix FROM assets WHERE id = ?`, id).Scan(&rejected, &deleted)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("asset eligible for share: %w", err)
	}
	if deleted.Valid {
		return false, nil
	}
	return rejected == 0, nil
}
