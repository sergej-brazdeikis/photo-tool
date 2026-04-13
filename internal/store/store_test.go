package store

import (
	"database/sql"
	"path/filepath"
	"testing"

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
	if v != 1 {
		t.Fatalf("schema version: got %d want 1", v)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'assets'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("assets table missing")
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
