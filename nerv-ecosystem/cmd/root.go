// Package cmd holds Cobra command wiring only. No business logic lives
// here, and this package never imports the SQL standard library package or
// a SQLite driver directly — every store interaction goes through
// internal/store's exported methods.
package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

// NewRootCommand builds a fresh command tree (root plus every subcommand)
// so each caller — production main.go or a test — gets an isolated
// instance. Package-level init() registration is forbidden here because
// parallel subtests would otherwise share flag state.
func NewRootCommand() *cobra.Command {
	var storePath string

	root := &cobra.Command{
		Use:          "modular",
		Short:        "Nerv Ecosystem platform CLI",
		SilenceUsage: true,
	}
	root.PersistentFlags().StringVar(&storePath, "store-path", "",
		"override the default store location (default: resolved via store.DefaultPath())")

	root.AddCommand(newStatusCommand(&storePath))

	return root
}

// Execute builds a root command and runs it against the process context.
func Execute() error {
	return NewRootCommand().ExecuteContext(context.Background())
}
