package cli

import (
	"database/sql"
	"fmt"
	"log/slog"

	"photo-tool/internal/config"
	"photo-tool/internal/store"
)

func openLibrary() (db *sql.DB, root string, cleanup func(), err error) {
	root, err = config.ResolveLibraryRoot()
	if err != nil {
		return nil, "", nil, fmt.Errorf("library root: %w", err)
	}
	if err := config.EnsureLibraryLayout(root); err != nil {
		return nil, "", nil, fmt.Errorf("library layout: %w", err)
	}
	db, err = store.Open(root)
	if err != nil {
		return nil, "", nil, fmt.Errorf("open store: %w", err)
	}
	cleanup = func() {
		if err := db.Close(); err != nil {
			slog.Warn("close store", "err", err)
		}
	}
	return db, root, cleanup, nil
}
