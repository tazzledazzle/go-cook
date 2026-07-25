package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tazzledazzle/go-cook/nerv-ecosystem/cmd"
)

func TestStatusCommand_UnopenableStorePathFailsLoudly(t *testing.T) {
	t.Parallel()

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
			storePath := filepath.Join(blocker, "nested", "registry.db")

			root := cmd.NewRootCommand()

			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{"status", "--store-path", storePath})

			err := root.Execute()
			require.Error(t, err)
			require.NotContains(t, stdout.String(), "Usage:")
			require.NotContains(t, stderr.String(), "Usage:")
		})
	}
}

func TestStatusCommand_CleansRedundantStorePathSegments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "store path with redundant traversal segments"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base := t.TempDir()
			sep := string(filepath.Separator)
			// Built by string concatenation (not filepath.Join, which would
			// clean the path itself) so the redundant segments genuinely
			// reach the command as flag input.
			messyPath := base + sep + "sub" + sep + ".." + sep + "sub" + sep + "." + sep + "registry.db"
			cleanedPath := filepath.Join(base, "sub", "registry.db")

			root := cmd.NewRootCommand()

			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{"status", "--store-path", messyPath})

			err := root.Execute()
			require.NoError(t, err)

			require.Contains(t, stdout.String(), cleanedPath)
			require.NotContains(t, stdout.String(), "..")
		})
	}
}
