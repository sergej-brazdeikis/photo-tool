package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // register "sqlite" driver
)

// Open opens (or creates) the library SQLite database under libraryRoot/.phototool/library.sqlite
// and applies embedded migrations. The directory libraryRoot/.phototool must already exist (typically via config.EnsureLibraryLayout).
func Open(libraryRoot string) (*sql.DB, error) {
	if libraryRoot == "" {
		return nil, fmt.Errorf("library root is empty")
	}
	dir := filepath.Join(libraryRoot, ".phototool")
	fi, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("library metadata dir %q: %w (call config.EnsureLibraryLayout before store.Open)", dir, err)
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("library metadata path %q is not a directory", dir)
	}
	dbPath := filepath.Join(dir, "library.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dbPath, err)
	}
	// Single connection: SQLite writer locks + busy_timeout PRAGMA apply per-connection in database/sql.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pragma busy_timeout: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
