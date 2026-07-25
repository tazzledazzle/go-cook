package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tazzledazzle/go-cook/nerv-ecosystem/internal/store"
)

// newStatusCommand builds the `modular status` subcommand, which opens
// (creating and migrating if needed) the local store and reports its
// path, journal mode, and FTS5 readiness. storePath is a pointer to the
// root command's persistent --store-path flag value.
func newStatusCommand(storePath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the local store's path and health",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := *storePath
			if path == "" {
				path = store.DefaultPath()
			}

			st, err := store.Open(path)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer func() { _ = st.Close() }()

			mode, err := st.JournalMode(cmd.Context())
			if err != nil {
				return fmt.Errorf("read journal mode: %w", err)
			}

			hasFTS, err := st.HasTable(cmd.Context(), "projects_fts")
			if err != nil {
				return fmt.Errorf("check fts5 table: %w", err)
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(),
				"store:        %s\njournal_mode: %s\nfts5 ready:   %v\n",
				path, mode, hasFTS); err != nil {
				return fmt.Errorf("write status output: %w", err)
			}
			return nil
		},
	}
}
