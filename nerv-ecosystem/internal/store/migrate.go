package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// runMigrations applies every embedded migration that has not yet been
// recorded in schema_migrations, each inside its own transaction so the
// statement execution and the tracking-table insert commit or roll back
// together.
func runMigrations(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	entries, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(entries) // "0001_init.sql" < "0002_....sql" lexically

	for _, name := range entries {
		var version int
		if _, err := fmt.Sscanf(name, "migrations/%04d_", &version); err != nil {
			return fmt.Errorf("parse migration version from %q: %w", name, err)
		}

		var applied bool
		if err := db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)`, version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if applied {
			continue
		}

		sqlBytes, err := migrationsFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", name, err)
		}

		if err := applyMigration(ctx, db, version, string(sqlBytes)); err != nil {
			return err
		}
	}
	return nil
}

// MigrationRecord describes one row of schema_migrations: the version
// number and the timestamp it was recorded. Reopening an unchanged store
// must return the same AppliedAt value for a given version — a changed
// value means the migration was silently re-applied.
type MigrationRecord struct {
	Version   int
	AppliedAt string
}

// AppliedMigrations returns every row in schema_migrations ordered by
// version, giving callers (this package's own tests today, and future
// phases eventually) a way to confirm schema state without reaching into
// the unexported *sql.DB.
func (s *Store) AppliedMigrations(ctx context.Context) ([]MigrationRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT version, applied_at FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []MigrationRecord
	for rows.Next() {
		var rec MigrationRecord
		if err := rows.Scan(&rec.Version, &rec.AppliedAt); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}
	return records, nil
}

func applyMigration(ctx context.Context, db *sql.DB, version int, sqlText string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %d: %w", version, err)
	}

	if _, err := tx.ExecContext(ctx, sqlText); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("apply migration %d: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version) VALUES (?)`, version,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %d: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d: %w", version, err)
	}
	return nil
}
