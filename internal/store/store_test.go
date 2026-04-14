package store

import (
	"database/sql"
	"errors"
	"fmt"
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
	if v != 7 {
		t.Fatalf("schema version: got %d want 7", v)
	}

	for _, tbl := range []string{"assets", "collections", "asset_collections", "tags", "asset_tags", "share_links", "share_link_members"} {
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
	if v != 7 {
		t.Fatalf("schema version: got %d want 7", v)
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

func TestCreateCollection_invalidDisplayDate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := CreateCollection(db, "Album", "not-a-date"); err == nil {
		t.Fatal("expected error for invalid display date")
	}
}

func TestCreateCollectionAndLinkAssets_success(t *testing.T) {
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
	if err := InsertAsset(db, "hash-a", "2024/a.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var aid int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'hash-a'`).Scan(&aid); err != nil {
		t.Fatal(err)
	}

	collID, err := CreateCollectionAndLinkAssets(db, "LoupeAlbum", "", []int64{aid})
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM collections WHERE id = ?`, collID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("collections row: n=%d err=%v", n, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_collections WHERE collection_id = ? AND asset_id = ?`, collID, aid).Scan(&n); err != nil || n != 1 {
		t.Fatalf("junction: n=%d err=%v", n, err)
	}
}

func TestCreateCollectionAndLinkAssets_rollsBackOnBadAsset(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := CreateCollectionAndLinkAssets(db, "OrphanRisk", "", []int64{999999}); err == nil {
		t.Fatal("expected error for missing asset id")
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM collections WHERE name = 'OrphanRisk'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("collection should not exist after failed link, got %d rows", n)
	}
}

func TestDeleteCollection_notFound(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	err = DeleteCollection(db, 999999)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("expected ErrCollectionNotFound, got %v", err)
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

func TestErrCollectionNotFound_wrappedIs(t *testing.T) {
	err := fmt.Errorf("collection detail: %w", ErrCollectionNotFound)
	if !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("errors.Is: got %v", err)
	}
}

func TestGetCollection_notFound(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = GetCollection(db, 99999)
	if err == nil || !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("expected ErrCollectionNotFound, got %v", err)
	}
}

func TestGetCollection_ok(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	id, err := CreateCollection(db, "Trip", "2024-06-01")
	if err != nil {
		t.Fatal(err)
	}
	d, err := GetCollection(db, id)
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != id || d.Name != "Trip" || d.DisplayDate != "2024-06-01" {
		t.Fatalf("detail: %+v", d)
	}
	if d.CreatedAtUnix <= 0 {
		t.Fatal("created_at_unix")
	}
}

func TestUpdateCollection_clearDisplayDateUsesCreatedAtUnixLocal(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ts := time.Date(2022, 7, 4, 23, 30, 0, 0, time.Local).Unix()
	if _, err := db.Exec(`
INSERT INTO collections (name, display_date, created_at_unix) VALUES ('Alb', '1999-01-01', ?)`, ts); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM collections WHERE name = 'Alb'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	want := time.Unix(ts, 0).In(time.Local).Format("2006-01-02")
	if err := UpdateCollection(db, id, "Alb", ""); err != nil {
		t.Fatal(err)
	}
	var dd string
	if err := db.QueryRow(`SELECT display_date FROM collections WHERE id = ?`, id).Scan(&dd); err != nil {
		t.Fatal(err)
	}
	if dd != want {
		t.Fatalf("display_date: got %q want %q", dd, want)
	}
}

func TestUpdateCollection_validationAndNotFound(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	id, err := CreateCollection(db, "X", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := UpdateCollection(db, id, " ", ""); err == nil {
		t.Fatal("empty name")
	}
	if err := UpdateCollection(db, id, "X", "nope"); err == nil {
		t.Fatal("bad date")
	}
	if err := UpdateCollection(db, 999999, "Y", ""); err == nil || !errors.Is(err, ErrCollectionNotFound) {
		t.Fatalf("not found: %v", err)
	}
}

func TestListCollectionIDsForAsset_andUnlink(t *testing.T) {
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
	if err := InsertAsset(db, "h1", "a.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var aid int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'h1'`).Scan(&aid); err != nil {
		t.Fatal(err)
	}
	c1, err := CreateCollection(db, "Zebra", "")
	if err != nil {
		t.Fatal(err)
	}
	c2, err := CreateCollection(db, "Alpha", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := LinkAssetsToCollection(db, c1, []int64{aid}); err != nil {
		t.Fatal(err)
	}
	if err := LinkAssetsToCollection(db, c2, []int64{aid}); err != nil {
		t.Fatal(err)
	}
	ids, err := ListCollectionIDsForAsset(db, aid)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != c2 || ids[1] != c1 {
		t.Fatalf("order/name stable: got %v want [%d,%d]", ids, c2, c1)
	}
	if err := UnlinkAssetFromCollection(db, aid, c1); err != nil {
		t.Fatal(err)
	}
	ids, err = ListCollectionIDsForAsset(db, aid)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != c2 {
		t.Fatalf("after unlink: %v", ids)
	}
	if err := UnlinkAssetFromCollection(db, aid, c1); err != nil {
		t.Fatal("idempotent unlink")
	}
	if err := UnlinkAssetFromCollection(db, aid, 99999); err != nil {
		t.Fatal("unlink missing collection junction")
	}
}

func TestUnlinkAssetFromCollection_idempotentNeverLinked(t *testing.T) {
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
	if err := InsertAsset(db, "h1", "a.jpg", now, now); err != nil {
		t.Fatal(err)
	}
	var aid int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'h1'`).Scan(&aid); err != nil {
		t.Fatal(err)
	}
	cid, err := CreateCollection(db, "C", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := UnlinkAssetFromCollection(db, aid, cid); err != nil {
		t.Fatal(err)
	}
}
