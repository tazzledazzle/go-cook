package kquick

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// OutputFormat is a supported kquick output encoding.
type OutputFormat string

const (
	OutputTable OutputFormat = "table"
	OutputJSON  OutputFormat = "json"
	OutputYAML  OutputFormat = "yaml"
)

// ParseOutputFormat validates and normalizes an -o/--output value.
func ParseOutputFormat(value string) (OutputFormat, error) {
	switch OutputFormat(strings.ToLower(value)) {
	case OutputTable, "":
		return OutputTable, nil
	case OutputJSON:
		return OutputJSON, nil
	case OutputYAML:
		return OutputYAML, nil
	default:
		return "", fmt.Errorf("unsupported output format %q", value)
	}
}

// WriteJSON writes v as indented JSON to w.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

// WriteYAML writes v as YAML to w.
func WriteYAML(w io.Writer, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode yaml: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write yaml: %w", err)
	}
	return nil
}

// FormatAge renders a human-readable age relative to now.
func FormatAge(created, now time.Time) string {
	if created.IsZero() {
		return "<unknown>"
	}
	d := now.Sub(created)
	if d < 0 {
		d = 0
	}
	return formatDuration(d)
}

func formatDuration(d time.Duration) string {
	seconds := int(d / time.Second)
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	if days < 365 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dy", days/365)
}

// ReadyCount returns ready/total containers for a Pod.
func ReadyCount(pod corev1.Pod) (ready, total int) {
	total = len(pod.Status.ContainerStatuses)
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
	}
	return ready, total
}

// RestartCount returns the sum of container restart counts.
func RestartCount(pod corev1.Pod) int32 {
	var total int32
	for _, cs := range pod.Status.ContainerStatuses {
		total += cs.RestartCount
	}
	return total
}

// WritePodTable writes a kubectl-style Pod table sorted by name.
func WritePodTable(w io.Writer, list *corev1.PodList, now time.Time) error {
	pods := append([]corev1.Pod(nil), list.Items...)
	sort.Slice(pods, func(i, j int) bool {
		return pods[i].Name < pods[j].Name
	})

	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE\tNODE"); err != nil {
		return err
	}
	for _, pod := range pods {
		ready, total := ReadyCount(pod)
		if _, err := fmt.Fprintf(tw, "%s\t%d/%d\t%s\t%d\t%s\t%s\n",
			pod.Name,
			ready,
			total,
			pod.Status.Phase,
			RestartCount(pod),
			FormatAge(pod.CreationTimestamp.Time, now),
			pod.Spec.NodeName,
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}
