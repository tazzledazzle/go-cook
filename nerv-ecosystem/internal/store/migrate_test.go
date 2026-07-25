package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/tazzledazzle/go-cook/nerv-ecosystem/internal/store"
)

func TestReopen_DoesNotReapplyMigrations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "second open on an existing store"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dbPath := filepath.Join(t.TempDir(), "registry.db")
			ctx := context.Background()

			first, err := store.Open(dbPath)
			require.NoError(t, err)

			firstRecords, err := first.AppliedMigrations(ctx)
			require.NoError(t, err)
			require.Len(t, firstRecords, 1)
			require.Equal(t, 1, firstRecords[0].Version)
			require.NoError(t, first.Close())

			second, err := store.Open(dbPath)
			require.NoError(t, err)
			defer func() { _ = second.Close() }()

			secondRecords, err := second.AppliedMigrations(ctx)
			require.NoError(t, err)
			require.Len(t, secondRecords, 1, "reopen must not duplicate the schema_migrations row")
			require.Equal(t, 1, secondRecords[0].Version)
			require.Equal(t, firstRecords[0].AppliedAt, secondRecords[0].AppliedAt,
				"reopen must not re-apply (and re-timestamp) an already-applied migration")
		})
	}
}

func TestWALMultiReader_SecondHandleSeesFirstHandlesWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "two independently opened handles on the same store file"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dbPath := filepath.Join(t.TempDir(), "registry.db")

			// Bootstrap the schema through the store package once, then
			// close it: the remaining assertions open two independent raw
			// connections directly onto the resulting file, proving the
			// file itself supports WAL multi-reader visibility. No
			// production `projects` writer exists yet (Phase 2's `generate`
			// owns that API), so opening two raw connections onto the
			// bootstrapped file is how this external test package proves
			// the guarantee without reaching into the unexported *sql.DB.
			bootstrap, err := store.Open(dbPath)
			require.NoError(t, err)
			require.NoError(t, bootstrap.Close())

			dsn := "file:" + dbPath +
				"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

			handleA, err := sql.Open("sqlite", dsn)
			require.NoError(t, err)
			defer func() { _ = handleA.Close() }()

			handleB, err := sql.Open("sqlite", dsn)
			require.NoError(t, err)
			defer func() { _ = handleB.Close() }()

			ctx := context.Background()

			_, err = handleA.ExecContext(ctx,
				`INSERT INTO projects (name, team, language, path) VALUES (?, ?, ?, ?)`,
				"wal-reader-project", "wal-team", "go", "/tmp/wal-reader-project")
			require.NoError(t, err)

			var team string
			err = handleB.QueryRowContext(ctx,
				`SELECT team FROM projects WHERE name = ?`, "wal-reader-project").Scan(&team)
			require.NoError(t, err, "handle B must see handle A's committed write under WAL")
			require.Equal(t, "wal-team", team)
		})
	}
}
