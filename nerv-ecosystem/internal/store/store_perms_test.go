package store_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tazzledazzle/go-cook/nerv-ecosystem/internal/store"
)

func TestOpen_StoreDirectoryAndFilePermissions(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}

	tests := []struct {
		name string
	}{
		{name: "freshly created store directory and file"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storeDir := filepath.Join(t.TempDir(), "created-by-store")
			dbPath := filepath.Join(storeDir, "registry.db")

			st, err := store.Open(dbPath)
			require.NoError(t, err)
			defer func() { _ = st.Close() }()

			dirInfo, err := os.Stat(storeDir)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm(),
				"store directory must be exactly 0700")

			fileInfo, err := os.Stat(dbPath)
			require.NoError(t, err)
			require.Zero(t, fileInfo.Mode().Perm()&0o077,
				"database file must carry no group or other read/write bits")
		})
	}
}

func TestOpen_UnopenableParentReturnsWrappedError(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("creating a directory nested under a regular file behaves differently on Windows")
	}

	tests := []struct {
		name string
	}{
		{name: "store path nested under an existing regular file"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			blocker := filepath.Join(t.TempDir(), "blocker")
			require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

			dbPath := filepath.Join(blocker, "nested", "registry.db")

			st, err := store.Open(dbPath)
			require.Error(t, err)
			require.Nil(t, st)
			require.Contains(t, err.Error(), "create store dir",
				"error must name the failed operation, not just surface a driver-internal string")
		})
	}
}
