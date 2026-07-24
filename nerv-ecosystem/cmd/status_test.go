package cmd_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tazzledazzle/go-cook/nerv-ecosystem/cmd"
)

func TestStatusCommand_ReportsStoreHealth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "fresh store path via --store-path flag"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			storePath := filepath.Join(t.TempDir(), "registry.db")

			root := cmd.NewRootCommand()

			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{"status", "--store-path", storePath})

			err := root.Execute()
			require.NoError(t, err)

			output := stdout.String()
			require.Contains(t, output, storePath)
			require.Contains(t, output, "journal_mode: wal")
			require.Contains(t, output, "fts5 ready")
			require.Contains(t, output, "true")
		})
	}
}
