package kquick

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newGetCommand(app *App) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:  "get RESOURCE",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := strings.ToLower(args[0])
			switch resource {
			case "pod", "pods":
			default:
				return fmt.Errorf("unsupported resource %q", args[0])
			}

			format, err := ParseOutputFormat(output)
			if err != nil {
				return err
			}

			rt, namespace, err := app.Factory.New(cmd.Context(), app.Opts)
			if err != nil {
				return fmt.Errorf("get: %w", err)
			}

			list, err := rt.ListPods(cmd.Context(), namespace)
			if err != nil {
				return fmt.Errorf("get pods in namespace %q: %w", namespace, err)
			}

			now := app.Now
			if now.IsZero() {
				now = time.Now()
			}

			switch format {
			case OutputJSON:
				return WriteJSON(app.Out, list)
			case OutputYAML:
				return WriteYAML(app.Out, list)
			default:
				return WritePodTable(app.Out, list, now)
			}
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", string(OutputTable), "output format: table|json|yaml")
	return cmd
}
