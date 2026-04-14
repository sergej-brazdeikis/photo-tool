package store

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"photo-tool/internal/config"
)

func TestNormalizeTagLabel(t *testing.T) {
	got, err := NormalizeTagLabel("  a   b\tc ")
	if err != nil || got != "a b c" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := NormalizeTagLabel("   "); err == nil {
		t.Fatal("expected error for whitespace-only")
	}
}

func TestFindOrCreateTagByLabel_caseFolding(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	id1, err := FindOrCreateTagByLabel(db, "Beach")
	if err != nil || id1 <= 0 {
		t.Fatalf("first: id=%d err=%v", id1, err)
	}
	id2, err := FindOrCreateTagByLabel(db, "  beach  ")
	if err != nil || id2 != id1 {
		t.Fatalf("second: id=%d want %d err=%v", id2, id1, err)
	}
}

func TestFindTagByLabel(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, ok, err := FindTagByLabel(db, "nope"); err != nil || ok {
		t.Fatalf("missing: ok=%v err=%v", ok, err)
	}
	id, err := FindOrCreateTagByLabel(db, "x")
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := FindTagByLabel(db, "X")
	if err != nil || !ok || got != id {
		t.Fatalf("got id=%d ok=%v err=%v", got, ok, err)
	}
}

func TestAssetDelete_cascadesAssetTags(t *testing.T) {
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
	tid, err := FindOrCreateTagByLabel(db, "t")
	if err != nil {
		t.Fatal(err)
	}
	if err := LinkTagToAssets(db, tid, []int64{aid}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM assets WHERE id = ?`, aid); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_tags WHERE asset_id = ?`, aid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("junction rows after asset delete: %d", n)
	}
}

func TestForeignKeyCheck_afterMigrate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := ForeignKeyCheck(db); err != nil {
		t.Fatal(err)
	}
}

func TestLinkTagToAssets_partialChunkErrorMessage(t *testing.T) {
	orig := tagBulkChunk
	tagBulkChunk = 2
	t.Cleanup(func() { tagBulkChunk = orig })

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
	for _, h := range []string{"c1", "c2"} {
		if err := InsertAsset(db, h, h+".jpg", now, now); err != nil {
			t.Fatal(err)
		}
	}
	var a1, a2 int64
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'c1'`).Scan(&a1); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM assets WHERE content_hash = 'c2'`).Scan(&a2); err != nil {
		t.Fatal(err)
	}
	tid, err := FindOrCreateTagByLabel(db, "partial")
	if err != nil {
		t.Fatal(err)
	}
	badID := int64(9_999_999_999)
	err = LinkTagToAssets(db, tid, []int64{a1, a2, badID})
	if err == nil {
		t.Fatal("expected FK error on bad asset id")
	}
	if !strings.Contains(err.Error(), "first 2 selected assets were updated") {
		t.Fatalf("expected partial-apply hint, got: %v", err)
	}
	if err := ForeignKeyCheck(db); err != nil {
		t.Fatal(err)
	}
}

func TestListTags_order(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, l := range []string{"zebra", "Alpha", "beta"} {
		if _, err := FindOrCreateTagByLabel(db, l); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := ListTags(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Label != "Alpha" || rows[1].Label != "beta" || rows[2].Label != "zebra" {
		t.Fatalf("order: %#v", rows)
	}
}
