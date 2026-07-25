// Package store is the sole owner of database/sql and the SQLite schema in
// the Nerv Ecosystem CLI. No other package may import database/sql or a
// SQLite driver; every domain package reaches persistence exclusively
// through the exported methods on Store.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store wraps a single *sql.DB connection to one embedded SQLite file.
type Store struct {
	db   *sql.DB
	path string
}

// Open creates the parent directory (if absent) and opens (creating and
// migrating if needed) a single embedded SQLite file in WAL mode.
//
// The DSN uses modernc.org/sqlite's driver-specific `_pragma=name(value)`
// syntax. The mattn/go-sqlite3-style "journal mode" query parameter (with a
// leading underscore, distinct from the `_pragma=` form used below) is
// silently ignored by this driver and leaves the store in the default
// rollback-journal mode with no error — never use that form here.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	// SQLite serializes writers at the file level even in WAL mode; a single
	// pooled connection avoids write/write pool contention that busy_timeout
	// alone cannot resolve.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := runMigrations(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate store: %w", err)
	}

	// The driver creates the file using the process umask (typically
	// world-readable, e.g. 0644), not a fixed 0600 — verified empirically
	// under store_perms_test.go, so this cannot be left to the driver's
	// default as originally assumed. Tighten explicitly rather than rely
	// on umask, which callers of this CLI do not control.
	if err := os.Chmod(path, 0o600); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set store file permissions: %w", err)
	}

	return &Store{db: db, path: path}, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close store: %w", err)
	}
	return nil
}

// JournalMode reads back the effective journal mode rather than assuming
// the DSN pragma was honored — this is the assertion that catches a
// silently-ignored DSN pragma (see Open's docs).
func (s *Store) JournalMode(ctx context.Context) (string, error) {
	var mode string
	if err := s.db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&mode); err != nil {
		return "", fmt.Errorf("read journal mode: %w", err)
	}
	return mode, nil
}

// SchemaObjects returns the names of every table and virtual table recorded
// in sqlite_master.
func (s *Store) SchemaObjects(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type IN ('table', 'view')`)
	if err != nil {
		return nil, fmt.Errorf("list schema objects: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan schema object: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema objects: %w", err)
	}
	return names, nil
}

// HasTable reports whether a table or virtual table with the given name
// exists, using a parameterized query rather than string-formatted SQL.
func (s *Store) HasTable(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type IN ('table', 'view') AND name = ?)`,
		name,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check table %q: %w", name, err)
	}
	return exists, nil
}
