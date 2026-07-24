package kquick

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

func newDescribeCommand(app *App) *cobra.Command {
	return &cobra.Command{
		Use:  "describe RESOURCE NAME",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := strings.ToLower(args[0])
			name := args[1]
			switch resource {
			case "pod", "pods":
			default:
				return fmt.Errorf("unsupported resource %q", args[0])
			}

			rt, namespace, err := app.Factory.New(cmd.Context(), app.Opts)
			if err != nil {
				return fmt.Errorf("describe: %w", err)
			}

			pod, err := rt.GetPod(cmd.Context(), namespace, name)
			if err != nil {
				return fmt.Errorf("describe pod %q in namespace %q: %w", name, namespace, err)
			}

			events, err := rt.ListPodEvents(cmd.Context(), namespace, name)
			if err != nil {
				return fmt.Errorf("describe pod %q in namespace %q: %w", name, namespace, err)
			}

			return WritePodDescribe(app.Out, pod, events)
		},
	}
}

// WritePodDescribe writes a concise operational Pod summary.
func WritePodDescribe(w io.Writer, pod *corev1.Pod, events *corev1.EventList) error {
	if pod == nil {
		return fmt.Errorf("pod is nil")
	}
	if events == nil {
		events = &corev1.EventList{}
	}

	tw := tabwriter.NewWriter(w, 0, 8, 1, '\t', 0)

	if _, err := fmt.Fprintf(tw, "Name:\t%s\n", noneIfEmpty(pod.Name)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "Namespace:\t%s\n", noneIfEmpty(pod.Namespace)); err != nil {
		return err
	}
	if err := writeLabels(tw, pod.Labels); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "Status:\t%s\n", noneIfEmpty(string(pod.Status.Phase))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "Pod IP:\t%s\n", noneIfEmpty(pod.Status.PodIP)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(tw, "Node:\t%s\n", noneIfEmpty(pod.Spec.NodeName)); err != nil {
		return err
	}
	if err := writeContainers(tw, pod.Status.ContainerStatuses); err != nil {
		return err
	}
	if err := writeConditions(tw, pod.Status.Conditions); err != nil {
		return err
	}
	if err := writeEvents(tw, events.Items); err != nil {
		return err
	}
	return tw.Flush()
}

func noneIfEmpty(s string) string {
	if s == "" {
		return "<none>"
	}
	return s
}

func writeLabels(w io.Writer, labels map[string]string) error {
	if len(labels) == 0 {
		_, err := fmt.Fprintf(w, "Labels:\t%s\n", "<none>")
		return err
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	_, err := fmt.Fprintf(w, "Labels:\t%s=%s\n", keys[0], labels[keys[0]])
	if err != nil {
		return err
	}
	for _, k := range keys[1:] {
		if _, err := fmt.Fprintf(w, "\t%s=%s\n", k, labels[k]); err != nil {
			return err
		}
	}
	return nil
}

func writeContainers(w io.Writer, statuses []corev1.ContainerStatus) error {
	if len(statuses) == 0 {
		_, err := fmt.Fprintf(w, "Containers:\t%s\n", "<none>")
		return err
	}
	if _, err := fmt.Fprintln(w, "Containers:"); err != nil {
		return err
	}
	for _, cs := range statuses {
		ready := "False"
		if cs.Ready {
			ready = "True"
		}
		if _, err := fmt.Fprintf(w, "  %s:\n", cs.Name); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "    Image:\t%s\n", noneIfEmpty(cs.Image)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "    Ready:\t%s\n", ready); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "    Restarts:\t%d\n", cs.RestartCount); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "    State:\t%s\n", formatContainerState(cs.State)); err != nil {
			return err
		}
	}
	return nil
}

func formatContainerState(state corev1.ContainerState) string {
	switch {
	case state.Waiting != nil:
		if state.Waiting.Reason != "" {
			return fmt.Sprintf("Waiting (%s)", state.Waiting.Reason)
		}
		return "Waiting"
	case state.Running != nil:
		return "Running"
	case state.Terminated != nil:
		if state.Terminated.Reason != "" {
			return fmt.Sprintf("Terminated (%s, exit %d)", state.Terminated.Reason, state.Terminated.ExitCode)
		}
		return fmt.Sprintf("Terminated (exit %d)", state.Terminated.ExitCode)
	default:
		return "<none>"
	}
}

func writeConditions(w io.Writer, conditions []corev1.PodCondition) error {
	if len(conditions) == 0 {
		_, err := fmt.Fprintf(w, "Conditions:\t%s\n", "<none>")
		return err
	}
	if _, err := fmt.Fprintln(w, "Conditions:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  Type\tStatus\tReason\tMessage"); err != nil {
		return err
	}
	for _, c := range conditions {
		if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
			c.Type, c.Status, c.Reason, c.Message); err != nil {
			return err
		}
	}
	return nil
}

func writeEvents(w io.Writer, events []corev1.Event) error {
	if len(events) == 0 {
		_, err := fmt.Fprintf(w, "Events:\t%s\n", "<none>")
		return err
	}

	sorted := append([]corev1.Event(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool {
		ti := eventTimestamp(sorted[i])
		tj := eventTimestamp(sorted[j])
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		if sorted[i].Reason != sorted[j].Reason {
			return sorted[i].Reason < sorted[j].Reason
		}
		return sorted[i].Message < sorted[j].Message
	})

	if _, err := fmt.Fprintln(w, "Events:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  Time\tType\tReason\tMessage"); err != nil {
		return err
	}
	for _, ev := range sorted {
		ts := eventTimestamp(ev)
		tsText := "<unknown>"
		if !ts.IsZero() {
			tsText = ts.UTC().Format(time.RFC3339)
		}
		if _, err := fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
			tsText, ev.Type, ev.Reason, ev.Message); err != nil {
			return err
		}
	}
	return nil
}

func eventTimestamp(ev corev1.Event) time.Time {
	if !ev.EventTime.IsZero() {
		return ev.EventTime.Time
	}
	if !ev.LastTimestamp.IsZero() {
		return ev.LastTimestamp.Time
	}
	if !ev.FirstTimestamp.IsZero() {
		return ev.FirstTimestamp.Time
	}
	return time.Time{}
}
