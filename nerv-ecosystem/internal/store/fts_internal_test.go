package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file lives in the in-package `store` test suite (not `store_test`)
// specifically to reach the unexported *sql.DB for raw SQL round-trips.
// No production writer for `projects` is added by this test: Phase 2's
// `generate` (GEN-08) owns that API. Every value — including the MATCH
// term — is bound with `?`; a literal term inlined into the query string
// would defeat the point of this regression lock (threat T-1-01).
func TestFTS5Sync_InsertUpdateDeleteKeepIndexConsistent(t *testing.T) {
	t.Parallel()

	t.Run("insert makes the row findable by MATCH", func(t *testing.T) {
		t.Parallel()
		s, ctx := newFTSTestStore(t)

		mustExec(t, ctx, s, `INSERT INTO projects (name, team, language, path) VALUES (?, ?, ?, ?)`,
			"proj-insert", "alphateam", "go", "/tmp/proj-insert")

		var name string
		err := s.db.QueryRowContext(ctx, `
			SELECT p.name FROM projects p
			JOIN projects_fts f ON f.rowid = p.id
			WHERE projects_fts MATCH ?`, "alphateam").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "proj-insert", name)
	})

	t.Run("update makes the new team findable and the old team not", func(t *testing.T) {
		t.Parallel()
		s, ctx := newFTSTestStore(t)

		mustExec(t, ctx, s, `INSERT INTO projects (name, team, language, path) VALUES (?, ?, ?, ?)`,
			"proj-update", "bravoteam", "go", "/tmp/proj-update")
		mustExec(t, ctx, s, `UPDATE projects SET team = ? WHERE name = ?`, "charlieteam", "proj-update")

		var name string
		err := s.db.QueryRowContext(ctx, `
			SELECT p.name FROM projects p
			JOIN projects_fts f ON f.rowid = p.id
			WHERE projects_fts MATCH ?`, "charlieteam").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "proj-update", name)

		err = s.db.QueryRowContext(ctx, `
			SELECT p.name FROM projects p
			JOIN projects_fts f ON f.rowid = p.id
			WHERE projects_fts MATCH ?`, "bravoteam").Scan(&name)
		require.ErrorIs(t, err, sql.ErrNoRows, "the pre-update team must no longer be findable")
	})

	t.Run("delete removes the row from the index with no orphan", func(t *testing.T) {
		t.Parallel()
		s, ctx := newFTSTestStore(t)

		mustExec(t, ctx, s, `INSERT INTO projects (name, team, language, path) VALUES (?, ?, ?, ?)`,
			"proj-delete", "deltateam", "go", "/tmp/proj-delete")
		mustExec(t, ctx, s, `DELETE FROM projects WHERE name = ?`, "proj-delete")

		var name string
		err := s.db.QueryRowContext(ctx, `
			SELECT p.name FROM projects p
			JOIN projects_fts f ON f.rowid = p.id
			WHERE projects_fts MATCH ?`, "deltateam").Scan(&name)
		require.ErrorIs(t, err, sql.ErrNoRows, "a deleted row must leave no orphaned index entry")
	})
}

func newFTSTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "registry.db")
	s, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s, context.Background()
}

func mustExec(t *testing.T, ctx context.Context, s *Store, query string, args ...any) {
	t.Helper()
	_, err := s.db.ExecContext(ctx, query, args...)
	require.NoError(t, err)
}
