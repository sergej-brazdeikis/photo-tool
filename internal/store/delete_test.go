package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"photo-tool/internal/config"
	"photo-tool/internal/domain"
)

func TestDeleteAssetToTrash_happyPath_movesFileAndSoftDeletes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dayDir := filepath.Join(root, "2024", "01", "02")
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("2024", "01", "02", "a.jpg")
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('h1', ?, 1, 1, 0, NULL)`, rel)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	okShare, err := AssetEligibleForDefaultShare(db, id)
	if err != nil || !okShare {
		t.Fatalf("eligible before delete: ok=%v err=%v", okShare, err)
	}

	changed, err := DeleteAssetToTrash(db, root, id, 9001)
	if err != nil || !changed {
		t.Fatalf("delete: changed=%v err=%v", changed, err)
	}

	if _, err := os.Stat(abs); !os.IsNotExist(err) {
		t.Fatalf("source should be gone: stat err=%v", err)
	}
	qfile := filepath.Join(root, ".trash", strconv.FormatInt(id, 10), "a.jpg")
	if _, err := os.Stat(qfile); err != nil {
		t.Fatalf("quarantine file: %v", err)
	}

	okShare, err = AssetEligibleForDefaultShare(db, id)
	if err != nil || okShare {
		t.Fatalf("eligible after delete: ok=%v err=%v", okShare, err)
	}

	f := domain.ReviewFilters{}
	n, err := CountAssetsForReview(db, f)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("default review count: got %d want 0", n)
	}
}

func TestDeleteAssetToTrash_idempotentSecondCall(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dayDir := filepath.Join(root, "2024", "01", "02")
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("2024", "01", "02", "b.jpg")
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.WriteFile(abs, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('h2', ?, 1, 1, 0, NULL)`, rel)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	changed, err := DeleteAssetToTrash(db, root, id, 1)
	if err != nil || !changed {
		t.Fatalf("first: changed=%v err=%v", changed, err)
	}
	changed, err = DeleteAssetToTrash(db, root, id, 2)
	if err != nil || changed {
		t.Fatalf("second: changed=%v err=%v", changed, err)
	}
}

func TestDeleteAssetToTrash_unknownID(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	changed, err := DeleteAssetToTrash(db, root, 999999, 1)
	if err != nil || changed {
		t.Fatalf("unknown: changed=%v err=%v", changed, err)
	}
}

func TestDeleteAssetToTrash_missingPrimaryFileStillSoftDeletes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	rel := filepath.Join("2024", "01", "02", "gone.jpg")
	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('h3', ?, 1, 1, 0, NULL)`, rel)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	changed, err := DeleteAssetToTrash(db, root, id, 42)
	if err != nil || !changed {
		t.Fatalf("delete missing file: changed=%v err=%v", changed, err)
	}
	var del sql.NullInt64
	if err := db.QueryRow(`SELECT deleted_at_unix FROM assets WHERE id = ?`, id).Scan(&del); err != nil {
		t.Fatal(err)
	}
	if !del.Valid || del.Int64 != 42 {
		t.Fatalf("deleted_at: %+v", del)
	}
}

func TestDeleteAssetToTrash_rejectedRow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dayDir := filepath.Join(root, "2024", "01", "02")
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("2024", "01", "02", "rej.jpg")
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.WriteFile(abs, []byte("z"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, rejected_at_unix, deleted_at_unix)
VALUES ('h4', ?, 1, 1, 1, 9, NULL)`, rel)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	changed, err := DeleteAssetToTrash(db, root, id, 77)
	if err != nil || !changed {
		t.Fatalf("delete rejected row: changed=%v err=%v", changed, err)
	}

	nRej, err := CountRejectedForReview(db, domain.ReviewFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if nRej != 0 {
		t.Fatalf("rejected count after delete: got %d want 0", nRej)
	}
}

func TestDeleteAssetToTrash_quarantineCollisionUsesSuffix(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	dayDir := filepath.Join(root, "2024", "01", "02")
	if err := os.MkdirAll(dayDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("2024", "01", "02", "a.jpg")
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.WriteFile(abs, []byte("live"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('hc', ?, 1, 1, 0, NULL)`, rel)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	// Simulate crash-retry or stray file: basename already present under .trash/{id}/.
	qDir := filepath.Join(root, ".trash", strconv.FormatInt(id, 10))
	if err := os.MkdirAll(qDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(qDir, "a.jpg")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := DeleteAssetToTrash(db, root, id, 100)
	if err != nil || !changed {
		t.Fatalf("delete: changed=%v err=%v", changed, err)
	}

	if _, err := os.Stat(abs); !os.IsNotExist(err) {
		t.Fatalf("source should be gone: stat err=%v", err)
	}
	if b, err := os.ReadFile(stale); err != nil || string(b) != "stale" {
		t.Fatalf("pre-existing quarantine file: err=%v body=%q", err, b)
	}
	suffixed := filepath.Join(qDir, "a_1.jpg")
	b, err := os.ReadFile(suffixed)
	if err != nil || string(b) != "live" {
		t.Fatalf("suffixed quarantine file: err=%v body=%q", err, b)
	}
}

func TestDeleteAssetToTrash_relPathCleansToLibraryRootRejected(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Normalizes to library root via foo/.. — must not treat as a primary file (Story 2.7 path hardening).
	tail := filepath.Join("nest", "..")
	res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix)
VALUES ('hr', ?, 1, 1, 0, NULL)`, tail)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	changed, err := DeleteAssetToTrash(db, root, id, 1)
	if err == nil || changed {
		t.Fatalf("expected root-resolve error: changed=%v err=%v", changed, err)
	}
}

func TestDeleteAssetToTrash_invalidRelPathDotRejected(t *testing.T) {
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
VALUES ('hd', ?, 1, 1, 0, NULL)`, ".")
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	changed, err := DeleteAssetToTrash(db, root, id, 1)
	if err == nil || changed {
		t.Fatalf("expected invalid rel_path error: changed=%v err=%v", changed, err)
	}
}

func TestDeleteAssetToTrash_pathEscapeRejected(t *testing.T) {
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
VALUES ('hx', ?, 1, 1, 0, NULL)`, `..`+string(filepath.Separator)+`..`+string(filepath.Separator)+`etc`+string(filepath.Separator)+`passwd`)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}

	changed, err := DeleteAssetToTrash(db, root, id, 1)
	if err == nil || changed {
		t.Fatalf("expected path error: changed=%v err=%v", changed, err)
	}
}
