package store

import (
	"path/filepath"
	"testing"

	"photo-tool/internal/config"
)

func TestListCollectionAlbumListRows_orderAndCover(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	cAlpha, err := CreateCollection(db, "Alpha", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateCollection(db, "Beta", ""); err != nil {
		t.Fatal(err)
	}

	ins := func(rel, hash string, capUnix int64) int64 {
		t.Helper()
		res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rating, rejected, deleted_at_unix, camera_make, camera_model, camera_label)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			hash, rel, capUnix, 1, nil, 0, nil, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		return id
	}

	idOld := ins("a/old.jpg", "h1", 100)
	idNew := ins("a/new.jpg", "h2", 200)
	if err := LinkAssetsToCollection(db, cAlpha, []int64{idOld, idNew}); err != nil {
		t.Fatal(err)
	}

	rows, err := ListCollectionAlbumListRows(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows %+v", len(rows), rows)
	}
	if rows[0].Name != "Alpha" || rows[1].Name != "Beta" {
		t.Fatalf("order: %+v", rows)
	}
	if rows[0].CoverAssetID != idNew || rows[0].CoverRelPath != "a/new.jpg" || rows[0].CoverContentHash != "h2" {
		t.Fatalf("Alpha cover: %+v", rows[0])
	}
	if rows[1].CoverAssetID != 0 {
		t.Fatalf("empty album cover: %+v", rows[1])
	}
}
