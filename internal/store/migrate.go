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
	v, err := currentVersion(db)
	if err != nil {
		return err
	}
	if v >= 1 {
		return nil
	}
	body, err := migrationFS.ReadFile("migrations/001_initial.sql")
	if err != nil {
		return err
	}
	if err := execStatements(db, string(body)); err != nil {
		return fmt.Errorf("migration 001: %w", err)
	}
	if _, err := db.Exec(`
INSERT INTO schema_meta (singleton, version) VALUES (1, 1)
ON CONFLICT(singleton) DO UPDATE SET version = excluded.version
`); err != nil {
		return fmt.Errorf("schema_meta version: %w", err)
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
