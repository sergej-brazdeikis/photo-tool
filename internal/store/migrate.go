package store

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

const targetSchemaVersion = 7

func currentVersion(db *sql.DB) (int, error) {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_meta'`).Scan(&n); err != nil {
		return 0, err
	}
	if n == 0 {
		return -1, nil
	}
	var v int
	err := db.QueryRow(`SELECT version FROM schema_meta WHERE singleton = 1`).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return v, err
}

func migrate(db *sql.DB) error {
	for {
		v, err := currentVersion(db)
		if err != nil {
			return err
		}
		if v >= targetSchemaVersion {
			return nil
		}
		switch {
		case v < 1:
			if err := applyMigration(db, "migrations/001_initial.sql", 1); err != nil {
				return fmt.Errorf("migration 001: %w", err)
			}
		case v < 2:
			if err := applyMigration(db, "migrations/002_collections.sql", 2); err != nil {
				return fmt.Errorf("migration 002: %w", err)
			}
		case v < 3:
			if err := applyMigration(db, "migrations/003_review_filters.sql", 3); err != nil {
				return fmt.Errorf("migration 003: %w", err)
			}
		case v < 4:
			if err := applyMigration(db, "migrations/004_tags.sql", 4); err != nil {
				return fmt.Errorf("migration 004: %w", err)
			}
		case v < 5:
			if err := applyMigration(db, "migrations/005_camera_meta.sql", 5); err != nil {
				return fmt.Errorf("migration 005: %w", err)
			}
		case v < 6:
			if err := applyMigration(db, "migrations/006_share_links.sql", 6); err != nil {
				return fmt.Errorf("migration 006: %w", err)
			}
		case v < 7:
			if err := applyMigration(db, "migrations/007_share_packages.sql", 7); err != nil {
				return fmt.Errorf("migration 007: %w", err)
			}
		default:
			return fmt.Errorf("store migrate: unsupported schema version %d (target %d)", v, targetSchemaVersion)
		}
	}
}

func applyMigration(db *sql.DB, relPath string, newVersion int) error {
	body, err := migrationFS.ReadFile(relPath)
	if err != nil {
		return err
	}
	if err := execStatements(db, string(body)); err != nil {
		return fmt.Errorf("exec %s: %w", relPath, err)
	}
	if _, err := db.Exec(`
INSERT INTO schema_meta (singleton, version) VALUES (1, ?)
ON CONFLICT(singleton) DO UPDATE SET version = excluded.version
`, newVersion); err != nil {
		return fmt.Errorf("schema_meta version %d: %w", newVersion, err)
	}
	return nil
}

func execStatements(db *sql.DB, sqlText string) error {
	for _, stmt := range splitSQLStatements(sqlText) {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("%w: %q", err, stmt)
		}
	}
	return nil
}

func splitSQLStatements(sqlText string) []string {
	var out []string
	var b strings.Builder
	for _, line := range strings.Split(sqlText, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	chunk := strings.TrimSpace(b.String())
	if chunk == "" {
		return nil
	}
	for _, part := range strings.Split(chunk, ";") {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
