package store

import (
	"path/filepath"
	"testing"

	"photo-tool/internal/config"
)

func TestUpdateAssetRating(t *testing.T) {
	root := filepath.Join(t.TempDir(), "lib")
	if err := config.EnsureLibraryLayout(root); err != nil {
		t.Fatal(err)
	}
	db, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	insert := func(rel, hash string, deletedUnix any) int64 {
		t.Helper()
		res, err := db.Exec(`
INSERT INTO assets (content_hash, rel_path, capture_time_unix, created_at_unix, rejected, deleted_at_unix, rating)
VALUES (?, ?, 1, 1, 0, ?, NULL)`, hash, rel, deletedUnix)
		if err != nil {
			t.Fatal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	activeID := insert("a/x.jpg", "h1", nil)

	t.Run("valid write", func(t *testing.T) {
		if err := UpdateAssetRating(db, activeID, 3); err != nil {
			t.Fatal(err)
		}
		var got int
		if err := db.QueryRow(`SELECT rating FROM assets WHERE id = ?`, activeID).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != 3 {
			t.Fatalf("rating: got %d want 3", got)
		}
	})

	t.Run("invalid rating low", func(t *testing.T) {
		err := UpdateAssetRating(db, activeID, 0)
		if err == nil {
			t.Fatal("want error for rating 0")
		}
	})

	t.Run("invalid rating high", func(t *testing.T) {
		err := UpdateAssetRating(db, activeID, 6)
		if err == nil {
			t.Fatal("want error for rating 6")
		}
	})

	t.Run("missing id", func(t *testing.T) {
		err := UpdateAssetRating(db, 99999, 2)
		if err == nil {
			t.Fatal("want error when no row updated")
		}
	})

	t.Run("soft-deleted row", func(t *testing.T) {
		delID := insert("a/y.jpg", "h2", 42)
		err := UpdateAssetRating(db, delID, 4)
		if err == nil {
			t.Fatal("want error when asset is soft-deleted")
		}
	})
}
