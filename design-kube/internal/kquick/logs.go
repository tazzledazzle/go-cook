package kquick

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

func newLogsCommand(app *App) *cobra.Command {
	var (
		container string
		follow    bool
		tail      int64
	)

	cmd := &cobra.Command{
		Use:  "logs RESOURCE NAME",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if tail < -1 {
				return fmt.Errorf("--tail must be >= -1")
			}

			resource := strings.ToLower(args[0])
			podName := args[1]
			switch resource {
			case "pod", "pods":
			default:
				return fmt.Errorf("unsupported resource %q", args[0])
			}

			rt, namespace, err := app.Factory.New(cmd.Context(), app.Opts)
			if err != nil {
				return fmt.Errorf("logs: %w", err)
			}

			pod, err := rt.GetPod(cmd.Context(), namespace, podName)
			if err != nil {
				return fmt.Errorf("logs pod %q in namespace %q: %w", podName, namespace, err)
			}

			selected, err := selectContainer(pod, container)
			if err != nil {
				return err
			}

			opts := &corev1.PodLogOptions{
				Container: selected,
				Follow:    follow,
			}
			if tail != -1 {
				t := tail
				opts.TailLines = &t
			}

			stream, err := rt.StreamPodLogs(cmd.Context(), namespace, podName, opts)
			if err != nil {
				return fmt.Errorf("logs pod %q in namespace %q: %w", podName, namespace, err)
			}

			_, copyErr := io.Copy(app.Out, stream)
			closeErr := stream.Close()
			if copyErr != nil {
				if closeErr != nil {
					return fmt.Errorf("%w (also failed to close log stream: %v)", copyErr, closeErr)
				}
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&container, "container", "c", "", "container name; required when the Pod has more than one container")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "stream logs continuously")
	cmd.Flags().Int64Var(&tail, "tail", -1, "number of lines from the end of the logs to show; -1 shows all")
	return cmd
}

// regularContainerNames returns Spec.Containers names in declaration order.
// Init and ephemeral containers are intentionally excluded in version one.
func regularContainerNames(pod *corev1.Pod) []string {
	if pod == nil {
		return nil
	}
	names := make([]string, 0, len(pod.Spec.Containers))
	for _, c := range pod.Spec.Containers {
		names = append(names, c.Name)
	}
	return names
}

func selectContainer(pod *corev1.Pod, requested string) (string, error) {
	if pod == nil {
		return "", fmt.Errorf("pod is nil")
	}
	names := regularContainerNames(pod)
	if len(names) == 0 {
		return "", fmt.Errorf("pod %q has no containers", pod.Name)
	}

	if requested == "" {
		if len(names) == 1 {
			return names[0], nil
		}
		return "", fmt.Errorf("a container name must be specified for pod %q, choose one of: [%s]",
			pod.Name, strings.Join(names, " "))
	}

	for _, name := range names {
		if name == requested {
			return requested, nil
		}
	}
	return "", fmt.Errorf("container %q is not valid for pod %q; valid names: [%s]",
		requested, pod.Name, strings.Join(names, " "))
}
