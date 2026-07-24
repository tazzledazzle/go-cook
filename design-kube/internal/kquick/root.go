package kquick

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// App holds shared dependencies for kquick commands.
type App struct {
	Factory RuntimeFactory
	Out     io.Writer
	ErrOut  io.Writer
	Opts    KubeOptions
}

// NewRootCommand builds the kquick root command with injected dependencies.
func NewRootCommand(factory RuntimeFactory, out, errOut io.Writer) *cobra.Command {
	app := &App{
		Factory: factory,
		Out:     out,
		ErrOut:  errOut,
	}

	root := &cobra.Command{
		Use:           "kquick",
		Short:         "Quick read-only Kubernetes Pod inspection",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetOut(out)
	root.SetErr(errOut)

	root.PersistentFlags().StringVar(&app.Opts.Kubeconfig, "kubeconfig", "", "path to the kubeconfig file")
	root.PersistentFlags().StringVar(&app.Opts.Context, "context", "", "kubeconfig context to use")
	root.PersistentFlags().StringVarP(&app.Opts.Namespace, "namespace", "n", "", "namespace to use")

	root.AddCommand(newPlaceholderCommand(app, "get", "RESOURCE"))
	root.AddCommand(newPlaceholderCommand(app, "describe", "RESOURCE NAME"))
	root.AddCommand(newPlaceholderCommand(app, "logs", "RESOURCE NAME"))

	return root
}

func newPlaceholderCommand(app *App, use, argsUse string) *cobra.Command {
	return &cobra.Command{
		Use:  use + " " + argsUse,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _, err := app.Factory.New(cmd.Context(), app.Opts)
			if err != nil {
				return fmt.Errorf("%s: %w", use, err)
			}
			return nil
		},
	}
}
