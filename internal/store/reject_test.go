package store

import (
	"path/filepath"
	"testing"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
)

func TestRejectAsset_RestoreAsset_roundtripAndIdempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('h1', 'a.jpg', 1, 1, 0, NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	okShare, err := AssetEligibleForDefaultShare(db, id)
	if err != nil || !okShare {
		t.Fatalf("eligible before reject: ok=%v err=%v", okShare, err)
	}

	changed, err := RejectAsset(db, id, 4242)
	if err != nil || !changed {
		t.Fatalf("reject: changed=%v err=%v", changed, err)
	}
	changed, err = RejectAsset(db, id, 9999)
	if err != nil || changed {
		t.Fatalf("reject again idempotent: changed=%v err=%v", changed, err)
	}

	okShare, err = AssetEligibleForDefaultShare(db, id)
	if err != nil || okShare {
		t.Fatalf("eligible after reject: ok=%v err=%v", okShare, err)
	}

	var rejAt int64
	if err := db.QueryRow(`SELECT rejected_at_unix FROM assets WHERE id = ?`, id).Scan(&rejAt); err != nil {
		t.Fatal(err)
	}
	if rejAt != 4242 {
		t.Fatalf("rejected_at_unix: got %d want 4242", rejAt)
	}

	changed, err = RestoreAsset(db, id)
	if err != nil || !changed {
		t.Fatalf("restore: changed=%v err=%v", changed, err)
	}
	changed, err = RestoreAsset(db, id)
	if err != nil || changed {
		t.Fatalf("restore again idempotent: changed=%v err=%v", changed, err)
	}

	okShare, err = AssetEligibleForDefaultShare(db, id)
	if err != nil || !okShare {
		t.Fatalf("eligible after restore: ok=%v err=%v", okShare, err)
	}
}

func TestRejectAsset_RestoreAsset_softDeletedNoop(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('h1', 'gone.jpg', 1, 1, 0, 99)`)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	changed, err := RejectAsset(db, id, 1)
	if err != nil || changed {
		t.Fatalf("reject deleted: changed=%v err=%v", changed, err)
	}
	changed, err = RestoreAsset(db, id)
	if err != nil || changed {
		t.Fatalf("restore deleted: changed=%v err=%v", changed, err)
	}

	ok, err := AssetEligibleForDefaultShare(db, id)
	if err != nil || ok {
		t.Fatalf("share eligible deleted: ok=%v err=%v", ok, err)
	}
}

func TestRejectAsset_defaultReviewCountDrops(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('h1', 'a.jpg', 1, 1, 0, NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	f := domain.ReviewFilters{}
	before, err := CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if before != 1 {
		t.Fatalf("before: got %d want 1", before)
	}
	changed, err := RejectAsset(db, id, 100)
	if err != nil || !changed {
		t.Fatalf("reject: changed=%v err=%v", changed, err)
	}
	after, err := CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if after != 0 {
		t.Fatalf("after reject default count: got %d want 0", after)
	}
	nRej, err := CountRejectedForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if nRej != 1 {
		t.Fatalf("rejected bucket count: got %d want 1", nRej)
	}
}

func TestAssetEligibleForDefaultShare_missingRow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ok, err := AssetEligibleForDefaultShare(db, 999999)
	if err != nil || ok {
		t.Fatalf("missing id: ok=%v err=%v", ok, err)
	}
}
