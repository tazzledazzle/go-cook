package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tazzledazzle/go-cook/nerv-ecosystem/internal/store"
)

func TestOpen_CreatesWALModeStoreWithFTS5Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "fresh database file in an existing directory"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dbPath := filepath.Join(t.TempDir(), "registry.db")

			st, err := store.Open(dbPath)
			require.NoError(t, err)
			require.NotNil(t, st)
			defer func() { _ = st.Close() }()

			ctx := context.Background()

			mode, err := st.JournalMode(ctx)
			require.NoError(t, err)
			require.Equal(t, "wal", mode)

			objects, err := st.SchemaObjects(ctx)
			require.NoError(t, err)
			require.Contains(t, objects, "projects")
			require.Contains(t, objects, "projects_fts")
		})
	}
}

func TestOpen_CreatesMissingParentDirectory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "parent directory does not yet exist"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dbPath := filepath.Join(t.TempDir(), "nested", "does-not-exist-yet", "registry.db")

			st, err := store.Open(dbPath)
			require.NoError(t, err)
			require.NotNil(t, st)
			defer func() { _ = st.Close() }()

			_, statErr := os.Stat(filepath.Dir(dbPath))
			require.NoError(t, statErr)
		})
	}
}

func TestStore_HasTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tableName string
		want      bool
	}{
		{name: "projects table exists", tableName: "projects", want: true},
		{name: "projects_fts table exists", tableName: "projects_fts", want: true},
		{name: "versions table does not exist in phase 1", tableName: "versions", want: false},
		{name: "edges table does not exist in phase 1", tableName: "edges", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dbPath := filepath.Join(t.TempDir(), "registry.db")
			st, err := store.Open(dbPath)
			require.NoError(t, err)
			defer func() { _ = st.Close() }()

			has, err := st.HasTable(context.Background(), tt.tableName)
			require.NoError(t, err)
			require.Equal(t, tt.want, has)
		})
	}
}
