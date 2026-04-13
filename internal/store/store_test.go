package store

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"photo-tool/internal/config"
)

func TestOpen_migratesFreshLibrary(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var v int
	if err := db.QueryRow(`SELECT version FROM schema_meta WHERE singleton = 1`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != 2 {
		t.Fatalf("schema version: got %d want 2", v)
	}

	for _, tbl := range []string{"assets", "collections", "asset_collections"} {
		var n int
		q := `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`
		if err := db.QueryRow(q, tbl).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("table %q missing", tbl)
		}
	}
}

func TestOpen_idempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db1, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	_ = db1.Close()
	db2, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	_ = db2.Close()
}

func TestOpen_partialUniqueRelPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ins := func(rel, hash string, deleted sql.NullInt64) error {
		_, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, deleted_at_unix)
VALUES (?, ?, 1, 1, ?)`, hash, rel, deleted)
		return err
	}
	if err := ins("2024/a.jpg", "hash1", sql.NullInt64{}); err != nil {
		t.Fatal(err)
	}
	if err := ins("2024/a.jpg", "hash2", sql.NullInt64{}); err == nil {
		t.Fatal("expected duplicate rel_path for active row")
	}
	if err := ins("2024/a.jpg", "hash3", sql.NullInt64{Int64: 99, Valid: true}); err != nil {
		t.Fatalf("soft-deleted duplicate rel_path should be allowed: %v", err)
	}
}

func TestAssetIDByContentHash(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().Unix()
	if err := InsertAsset(db, "hash-x", "2024/x.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	id, ok, err := AssetIDByContentHash(db, "hash-x")
	if err != nil || !ok || id <= 0 {
		t.Fatalf("AssetIDByContentHash: id=%d ok=%v err=%v", id, ok, err)
	}
	_, okMiss, err := AssetIDByContentHash(db, "no-such-hash")
	if err != nil || okMiss {
		t.Fatalf("missing hash: ok=%v err=%v", okMiss, err)
	}
}

func TestCollections_createLinkIdempotentDelete(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var v int
	if err := db.QueryRow(`SELECT version FROM schema_meta WHERE singleton = 1`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != 2 {
		t.Fatalf("schema version: got %d want 2", v)
	}

	now := time.Now().Unix()
	if err := InsertAsset(db, "hash-a", "2024/a.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	if err := InsertAsset(db, "hash-b", "2024/b.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var idA, idB int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'hash-a'`).Scan(&idA); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'hash-b'`).Scan(&idB); err != nil {
		t.Fatal(err)
	}

	collID, err := CreateCollection(db, "Vacation", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := LinkAssetsToCollection(db, collID, []int64{idA, idB}); err != nil {
		t.Fatal(err)
	}
	var junctionCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_collections WHERE collection_id = ?`, collID).Scan(&junctionCount); err != nil {
		t.Fatal(err)
	}
	if junctionCount != 2 {
		t.Fatalf("junction rows: got %d want 2", junctionCount)
	}
	if err := LinkAssetsToCollection(db, collID, []int64{idA, idB}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_collections WHERE collection_id = ?`, collID).Scan(&junctionCount); err != nil {
		t.Fatal(err)
	}
	if junctionCount != 2 {
		t.Fatalf("after idempotent link: junction rows: got %d want 2", junctionCount)
	}

	if err := DeleteCollection(db, collID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_collections WHERE collection_id = ?`, collID).Scan(&junctionCount); err != nil {
		t.Fatal(err)
	}
	if junctionCount != 0 {
		t.Fatalf("after delete: junction rows: got %d want 0", junctionCount)
	}
	var collLeft int
	if err := db.QueryRow(`SELECT COUNT(*) FROM collections WHERE id = ?`, collID).Scan(&collLeft); err != nil {
		t.Fatal(err)
	}
	if collLeft != 0 {
		t.Fatalf("collection row should be gone")
	}
	var assetCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets`).Scan(&assetCount); err != nil {
		t.Fatal(err)
	}
	if assetCount != 2 {
		t.Fatalf("assets unchanged: got %d want 2", assetCount)
	}
}

func TestCreateCollection_emptyName(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := CreateCollection(db, " ", ""); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestLinkAssetsToCollection_invalidReferences(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	collID, err := CreateCollection(db, "Album", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := LinkAssetsToCollection(db, collID, []int64{999999}); err == nil {
		t.Fatal("expected error for missing asset id")
	}
	now := time.Now().Unix()
	if err := InsertAsset(db, "hash-x", "2024/x.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var aid int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'hash-x'`).Scan(&aid); err != nil {
		t.Fatal(err)
	}
	if err := LinkAssetsToCollection(db, 999999, []int64{aid}); err == nil {
		t.Fatal("expected error for missing collection id")
	}
}
