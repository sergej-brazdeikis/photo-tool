package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"photo-tool/internal/domain"
)

// packageShareMaxEligibleAssets is the MVP cap on eligible assets in one package (Story 4.1 AC3c).
const packageShareMaxEligibleAssets = 500

// ErrShareAssetIneligible means the asset is missing, rejected, or soft-deleted (FR-29 / §3.4).
var ErrShareAssetIneligible = errors.New("share mint: asset not eligible for default share")

// ErrPackageTooManyAssets is returned when a confirmed package would exceed the MVP eligible-asset cap (Story 4.1 AC3).
var ErrPackageTooManyAssets = errors.New("share package mint: too many eligible assets")

// ErrPackageNoEligibleAssets is returned when every candidate is omitted as ineligible after manifest construction (Story 4.1 AC3).
var ErrPackageNoEligibleAssets = errors.New("share package mint: no eligible assets")

// MintDefaultShareLink creates a share_links row with SHA-256 (lowercase hex) of the raw token,
// returns the raw token to the caller only (never persisted). Eligibility is enforced inside the transaction.
func MintDefaultShareLink(ctx context.Context, db *sql.DB, assetID int64, createdAtUnix int64) (rawToken string, shareLinkID int64, err error) {
	if assetID <= 0 {
		return "", 0, fmt.Errorf("share mint: invalid asset id")
	}
	if createdAtUnix <= 0 {
		return "", 0, fmt.Errorf("share mint: invalid created time")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", 0, fmt.Errorf("share mint begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ok, err := assetEligibleForDefaultShareTx(tx, assetID)
	if err != nil {
		return "", 0, err
	}
	if !ok {
		return "", 0, ErrShareAssetIneligible
	}

	var rating sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT rating FROM assets WHERE id = ?`, assetID).Scan(&rating); err != nil {
		return "", 0, fmt.Errorf("share mint read rating: %w", err)
	}
	payload, err := json.Marshal(ShareSnapshotPayload{Rating: nullIntFromSQL(rating)})
	if err != nil {
		return "", 0, fmt.Errorf("share mint payload: %w", err)
	}

	for attempt := 0; attempt < 8; attempt++ {
		raw, err := randomURLSafeToken(32)
		if err != nil {
			return "", 0, err
		}
		hash := sha256.Sum256([]byte(raw))
		tokenHash := hex.EncodeToString(hash[:])

		res, err := tx.ExecContext(ctx, `
INSERT INTO share_links (token_hash, asset_id, created_at_unix, payload, link_kind)
VALUES (?, ?, ?, ?, 'single')`, tokenHash, assetID, createdAtUnix, string(payload))
		if err != nil {
			if isSQLiteUniqueTokenHash(err) && attempt+1 < 8 {
				continue
			}
			return "", 0, fmt.Errorf("share mint insert: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return "", 0, fmt.Errorf("share mint last id: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return "", 0, fmt.Errorf("share mint commit: %w", err)
		}
		return raw, id, nil
	}
	return "", 0, fmt.Errorf("share mint: exhausted token hash retries")
}

// ShareSnapshotPayload is the JSON shape stored in share_links.payload (Stories 3.1 / 3.3 / 4.1).
// Single-asset rows omit Kind (or use "single"); package rows use Kind=="package" plus optional display metadata.
type ShareSnapshotPayload struct {
	Kind          string `json:"kind,omitempty"`
	Rating        *int   `json:"rating,omitempty"`
	DisplayTitle  string `json:"display_title,omitempty"`
	AudienceLabel string `json:"audience_label,omitempty"`
}

// ParseShareSnapshotPayloadJSON unmarshals payload JSON for share page rendering.
// Unknown JSON keys are ignored (encoding/json); they must never be bound into HTML without an explicit story + sanitization pass.
// Empty input yields an empty payload (unrated). Invalid JSON returns an error.
func ParseShareSnapshotPayloadJSON(s string) (ShareSnapshotPayload, error) {
	if strings.TrimSpace(s) == "" {
		return ShareSnapshotPayload{}, nil
	}
	var p ShareSnapshotPayload
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return ShareSnapshotPayload{}, fmt.Errorf("share payload json: %w", err)
	}
	return p, nil
}

func nullIntFromSQL(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

// PackagePrepareEligibleForMint filters deduped manifest candidates to eligible (non-rejected, not trashed, existing)
// rows, preserving order. Returns ErrPackageNoEligibleAssets when candidates were non-empty but all ineligible.
// Returns ErrPackageTooManyAssets when the eligible set exceeds packageShareMaxEligibleAssets (MVP cap).
//
// Call after domain.StableDedupeAssetIDs at the manifest boundary; mint still re-checks eligibility in a transaction.
func PackagePrepareEligibleForMint(ctx context.Context, db *sql.DB, dedupedOrdered []int64) ([]int64, error) {
	if db == nil {
		return nil, fmt.Errorf("share package prepare: nil db")
	}
	if len(dedupedOrdered) == 0 {
		return nil, nil
	}
	eligible, err := filterAssetIDsEligibleForDefaultShare(ctx, db, dedupedOrdered)
	if err != nil {
		return nil, err
	}
	var out []int64
	for _, id := range dedupedOrdered {
		if eligible[id] {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, ErrPackageNoEligibleAssets
	}
	if len(out) > packageShareMaxEligibleAssets {
		return nil, ErrPackageTooManyAssets
	}
	return out, nil
}

func filterAssetIDsEligibleForDefaultShare(ctx context.Context, db *sql.DB, ids []int64) (map[int64]bool, error) {
	out := make(map[int64]bool)
	const chunk = 400
	for i := 0; i < len(ids); i += chunk {
		end := i + chunk
		if end > len(ids) {
			end = len(ids)
		}
		part := ids[i:end]
		placeholders := strings.Repeat("?,", len(part))
		placeholders = strings.TrimSuffix(placeholders, ",")
		args := make([]any, 0, len(part))
		for _, id := range part {
			args = append(args, id)
		}
		q := `SELECT id FROM assets WHERE id IN (` + placeholders + `) AND rejected = 0 AND deleted_at_unix IS NULL`
		rows, err := db.QueryContext(ctx, q, args...)
		if err != nil {
			return nil, fmt.Errorf("share package filter eligible: %w", err)
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("share package filter eligible scan: %w", err)
			}
			out[id] = true
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}
	return out, nil
}

// MintPackageShareLink creates one share_links row (link_kind='package', asset_id NULL, token hash only)
// and ordered share_link_members child rows in a single transaction (Story 4.1 AC4).
// Pass eligibleOrdered from PackagePrepareEligibleForMint (or equivalent); each id is eligibility-checked again here.
// Recipient HTML index MUST load members via ResolvePackageShareLink, not ResolveDefaultShareLink (AC5 snapshot vs live join).
func MintPackageShareLink(ctx context.Context, db *sql.DB, eligibleOrdered []int64, createdAtUnix int64, payload ShareSnapshotPayload) (rawToken string, shareLinkID int64, err error) {
	if db == nil {
		return "", 0, fmt.Errorf("share package mint: nil db")
	}
	if createdAtUnix <= 0 {
		return "", 0, fmt.Errorf("share package mint: invalid created time")
	}
	eligibleOrdered = domain.StableDedupeAssetIDs(eligibleOrdered)
	if len(eligibleOrdered) == 0 {
		return "", 0, ErrPackageNoEligibleAssets
	}
	if len(eligibleOrdered) > packageShareMaxEligibleAssets {
		return "", 0, ErrPackageTooManyAssets
	}

	p := payload
	p.Kind = "package"
	body, err := json.Marshal(p)
	if err != nil {
		return "", 0, fmt.Errorf("share package mint payload: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", 0, fmt.Errorf("share package mint begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, id := range eligibleOrdered {
		ok, err := assetEligibleForDefaultShareTx(tx, id)
		if err != nil {
			return "", 0, err
		}
		if !ok {
			return "", 0, ErrShareAssetIneligible
		}
	}

	for attempt := 0; attempt < 8; attempt++ {
		raw, err := randomURLSafeToken(32)
		if err != nil {
			return "", 0, err
		}
		hash := sha256.Sum256([]byte(raw))
		tokenHash := hex.EncodeToString(hash[:])

		res, err := tx.ExecContext(ctx, `
INSERT INTO share_links (token_hash, asset_id, created_at_unix, payload, link_kind)
VALUES (?, NULL, ?, ?, 'package')`, tokenHash, createdAtUnix, string(body))
		if err != nil {
			if isSQLiteUniqueTokenHash(err) && attempt+1 < 8 {
				continue
			}
			return "", 0, fmt.Errorf("share package mint insert: %w", err)
		}
		linkID, err := res.LastInsertId()
		if err != nil {
			return "", 0, fmt.Errorf("share package mint last id: %w", err)
		}
		for pos, aid := range eligibleOrdered {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO share_link_members (share_link_id, position, asset_id) VALUES (?, ?, ?)`, linkID, pos, aid); err != nil {
				return "", 0, fmt.Errorf("share package mint member: %w", err)
			}
		}
		if err := tx.Commit(); err != nil {
			return "", 0, fmt.Errorf("share package mint commit: %w", err)
		}
		return raw, linkID, nil
	}
	return "", 0, fmt.Errorf("share package mint: exhausted token hash retries")
}

// ResolvedPackageShareLink is package share metadata for GET /s/{token} HTML (ordered snapshot members).
type ResolvedPackageShareLink struct {
	ShareLinkID int64
	Payload     string
	MemberIDs   []int64
}

// ResolvePackageShareLink loads a package share by raw token hash without joining assets for live eligibility (AC5).
// Single-asset links and unknown tokens yield (nil, nil). Database failures are non-nil errors.
func ResolvePackageShareLink(ctx context.Context, db *sql.DB, rawToken string) (*ResolvedPackageShareLink, error) {
	if db == nil {
		return nil, fmt.Errorf("share package resolve: nil db")
	}
	sum := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(sum[:])

	var linkID int64
	var payload string
	var kind string
	err := db.QueryRowContext(ctx, `
SELECT id, IFNULL(payload, ''), link_kind FROM share_links WHERE token_hash = ?`, tokenHash).Scan(&linkID, &payload, &kind)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("share package resolve parent: %w", err)
	}
	if kind != "package" {
		return nil, nil
	}

	rows, err := db.QueryContext(ctx, `
SELECT asset_id FROM share_link_members WHERE share_link_id = ? ORDER BY position ASC`, linkID)
	if err != nil {
		return nil, fmt.Errorf("share package resolve members: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var members []int64
	for rows.Next() {
		var aid int64
		if err := rows.Scan(&aid); err != nil {
			return nil, fmt.Errorf("share package resolve member scan: %w", err)
		}
		members = append(members, aid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &ResolvedPackageShareLink{
		ShareLinkID: linkID,
		Payload:     payload,
		MemberIDs:   members,
	}, nil
}

func assetEligibleForDefaultShareTx(tx *sql.Tx, id int64) (ok bool, err error) {
	var rejected int
	var deleted sql.NullInt64
	err = tx.QueryRow(`
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

func randomURLSafeToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("share mint token entropy: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func isSQLiteUniqueTokenHash(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "unique constraint") && strings.Contains(s, "token_hash")
}

// DefaultShareBlockedUserMessage returns a non-empty message when the default share flow should stop
// before preview/mint (rejected, trash, or missing row). Empty string means proceed.
// Eligibility matches AssetEligibleForDefaultShare (AC4); a follow-up read only runs when ineligible to pick user-facing copy.
func DefaultShareBlockedUserMessage(db *sql.DB, assetID int64) (msg string, err error) {
	if assetID <= 0 {
		return "No photo is selected.", nil
	}
	ok, err := AssetEligibleForDefaultShare(db, assetID)
	if err != nil {
		return "", err
	}
	if ok {
		return "", nil
	}
	var rejected int
	var deleted sql.NullInt64
	qerr := db.QueryRow(`
SELECT rejected, deleted_at_unix FROM assets WHERE id = ?`, assetID).Scan(&rejected, &deleted)
	if qerr == sql.ErrNoRows {
		return "This photo is no longer in the library.", nil
	}
	if qerr != nil {
		return "", fmt.Errorf("share gate: %w", qerr)
	}
	if deleted.Valid {
		return "This photo is in library trash and can't be shared.", nil
	}
	if rejected != 0 {
		return "Rejected photos can't be shared. Restore the photo first.", nil
	}
	return "This photo can't be shared.", nil
}

// ResolvedDefaultShareLink is the minimal row material for GET /s/{token} before Story 3.3 serves bytes/HTML.
type ResolvedDefaultShareLink struct {
	ShareLinkID int64
	AssetID     int64
	Payload     string
}

// ResolveDefaultShareLink hashes rawToken like MintDefaultShareLink, then returns the share row only when
// token_hash matches and the asset is still eligible (not rejected, not soft-deleted). Any miss returns (nil, nil).
// A non-nil error indicates a database failure.
//
// Story 4.1 package links must not reuse this function for recipient index listing: the JOIN on assets enforces
// live eligibility and would drop snapshot members that were later rejected. Package HTML index must load members
// from persisted package rows only; per-member image bytes still use eligibility checks like single-asset /i/{token}.
func ResolveDefaultShareLink(ctx context.Context, db *sql.DB, rawToken string) (*ResolvedDefaultShareLink, error) {
	if db == nil {
		return nil, fmt.Errorf("share resolve: nil db")
	}
	sum := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(sum[:])

	var out ResolvedDefaultShareLink
	err := db.QueryRowContext(ctx, `
SELECT sl.id, sl.asset_id, IFNULL(sl.payload, '')
FROM share_links sl
JOIN assets a ON a.id = sl.asset_id
WHERE sl.token_hash = ?
  AND sl.link_kind = 'single'
  AND a.deleted_at_unix IS NULL
  AND a.rejected = 0`, tokenHash).Scan(&out.ShareLinkID, &out.AssetID, &out.Payload)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("share resolve: %w", err)
	}
	return &out, nil
}

// AssetLibraryFileForShare returns the primary library-relative path and optional mime for an asset
// that is still eligible for sharing (Story 3.3 image bytes). ok is false when the row is missing or ineligible.
func AssetLibraryFileForShare(ctx context.Context, db *sql.DB, assetID int64) (relPath string, mime sql.NullString, ok bool, err error) {
	if db == nil {
		return "", sql.NullString{}, false, fmt.Errorf("share asset file: nil db")
	}
	if assetID <= 0 {
		return "", sql.NullString{}, false, nil
	}
	err = db.QueryRowContext(ctx, `
SELECT rel_path, mime FROM assets
WHERE id = ?
  AND deleted_at_unix IS NULL
  AND rejected = 0`, assetID).Scan(&relPath, &mime)
	if err == sql.ErrNoRows {
		return "", sql.NullString{}, false, nil
	}
	if err != nil {
		return "", sql.NullString{}, false, fmt.Errorf("share asset file: %w", err)
	}
	if relPath == "" {
		return "", sql.NullString{}, false, nil
	}
	return relPath, mime, true, nil
}

// CountShareLinks returns the number of share_links rows (tests / diagnostics).
func CountShareLinks(db *sql.DB) (int64, error) {
	var n int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM share_links`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count share_links: %w", err)
	}
	return n, nil
}
