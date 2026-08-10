package kquick

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// flowRuntime is a single fake Runtime covering get, describe, and logs flows.
type flowRuntime struct {
	pod    *corev1.Pod
	pods   *corev1.PodList
	events *corev1.EventList
	logs   string

	listCalls   int
	listNS      string
	getCalls    int
	getNS       string
	getName     string
	eventsCalls int
	eventsNS    string
	eventsName  string
	streamCalls int
	streamNS    string
	streamName  string
	streamOpts  *corev1.PodLogOptions
}

func (f *flowRuntime) ListPods(_ context.Context, namespace string) (*corev1.PodList, error) {
	f.listCalls++
	f.listNS = namespace
	if f.pods == nil {
		return &corev1.PodList{}, nil
	}
	return f.pods, nil
}

func (f *flowRuntime) GetPod(_ context.Context, namespace, name string) (*corev1.Pod, error) {
	f.getCalls++
	f.getNS = namespace
	f.getName = name
	if f.pod == nil {
		return nil, errors.New("pod not configured")
	}
	return f.pod, nil
}

func (f *flowRuntime) ListPodEvents(_ context.Context, namespace, podName string) (*corev1.EventList, error) {
	f.eventsCalls++
	f.eventsNS = namespace
	f.eventsName = podName
	if f.events == nil {
		return &corev1.EventList{}, nil
	}
	return f.events, nil
}

func (f *flowRuntime) StreamPodLogs(_ context.Context, namespace, name string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	f.streamCalls++
	f.streamNS = namespace
	f.streamName = name
	if opts != nil {
		cp := *opts
		if opts.TailLines != nil {
			v := *opts.TailLines
			cp.TailLines = &v
		}
		f.streamOpts = &cp
	}
	return io.NopCloser(strings.NewReader(f.logs)), nil
}

type flowFactory struct {
	rt    *flowRuntime
	calls []KubeOptions
}

func (f *flowFactory) New(_ context.Context, opts KubeOptions) (Runtime, string, error) {
	f.calls = append(f.calls, opts)
	ns := "default"
	if opts.Namespace != "" {
		ns = opts.Namespace
	}
	return f.rt, ns, nil
}

// newFlowFixture returns an isolated runtime/factory pair for one subtest.
func newFlowFixture(now time.Time) (*flowRuntime, *flowFactory) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "api-0",
			Namespace:         "demo",
			CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
			Labels:            map[string]string{"app": "api"},
		},
		Spec: corev1.PodSpec{
			NodeName:   "worker-1",
			Containers: []corev1.Container{{Name: "api", Image: "api:1"}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "api", Ready: true, RestartCount: 1, Image: "api:1", State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				}},
			},
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}

	rt := &flowRuntime{
		pod: pod,
		pods: &corev1.PodList{
			Items: []corev1.Pod{*pod},
		},
		events: &corev1.EventList{
			Items: []corev1.Event{{
				ObjectMeta:     metav1.ObjectMeta{Name: "api-0.1"},
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "api-0", Namespace: "demo"},
				Type:           corev1.EventTypeNormal,
				Reason:         "Started",
				Message:        "Started container api",
				LastTimestamp:  metav1.NewTime(now.Add(-time.Hour)),
			}},
		},
		logs: "ready to serve\n",
	}
	return rt, &flowFactory{rt: rt}
}

func TestCommandFlows(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)

	flows := []struct {
		name   string
		args   []string
		assert func(t *testing.T, out, errOut string, rt *flowRuntime, factory *flowFactory)
	}{
		{
			name: "get pods -n demo",
			args: []string{"get", "pods", "-n", "demo"},
			assert: func(t *testing.T, out, errOut string, rt *flowRuntime, factory *flowFactory) {
				t.Helper()
				if errOut != "" {
					t.Fatalf("stderr = %q, want empty", errOut)
				}
				// Pinned App.Now + fixture CreationTimestamp (-2h) → AGE 2h;
				// ReadyCount 1/1, RestartCount 1, NodeName worker-1.
				for _, want := range []string{
					"NAME", "READY", "STATUS", "RESTARTS", "AGE", "NODE",
					"api-0", "1/1", "Running", "2h", "worker-1",
				} {
					if !strings.Contains(out, want) {
						t.Fatalf("stdout missing %q: %q", want, out)
					}
				}
				// RESTARTS column value "1" — match the data row fields in order.
				lines := strings.Split(strings.TrimSpace(out), "\n")
				if len(lines) < 2 {
					t.Fatalf("stdout want header + data row, got %q", out)
				}
				fields := strings.Fields(lines[1])
				if len(fields) != 6 {
					t.Fatalf("data row fields = %v, want 6 columns", fields)
				}
				wantFields := []string{"api-0", "1/1", "Running", "1", "2h", "worker-1"}
				for i, want := range wantFields {
					if fields[i] != want {
						t.Fatalf("data row[%d] = %q, want %q (row=%v)", i, fields[i], want, fields)
					}
				}
				if rt.listCalls != 1 {
					t.Fatalf("ListPods calls = %d, want 1", rt.listCalls)
				}
				if rt.listNS != "demo" {
					t.Fatalf("ListPods namespace = %q, want %q", rt.listNS, "demo")
				}
				if len(factory.calls) != 1 || factory.calls[0].Namespace != "demo" {
					t.Fatalf("factory calls = %+v, want namespace demo", factory.calls)
				}
			},
		},
		{
			name: "describe pod api-0 -n demo",
			args: []string{"describe", "pod", "api-0", "-n", "demo"},
			assert: func(t *testing.T, out, errOut string, rt *flowRuntime, factory *flowFactory) {
				t.Helper()
				if errOut != "" {
					t.Fatalf("stderr = %q, want empty", errOut)
				}
				for _, want := range []string{"Name:", "api-0", "Namespace:", "demo", "Status:", "Running", "Events:", "Started"} {
					if !strings.Contains(out, want) {
						t.Fatalf("stdout missing %q: %q", want, out)
					}
				}
				if rt.getCalls != 1 || rt.getNS != "demo" || rt.getName != "api-0" {
					t.Fatalf("GetPod calls=%d ns=%q name=%q, want 1/demo/api-0", rt.getCalls, rt.getNS, rt.getName)
				}
				if rt.eventsCalls != 1 || rt.eventsNS != "demo" || rt.eventsName != "api-0" {
					t.Fatalf("ListPodEvents calls=%d ns=%q name=%q, want 1/demo/api-0", rt.eventsCalls, rt.eventsNS, rt.eventsName)
				}
				if len(factory.calls) != 1 || factory.calls[0].Namespace != "demo" {
					t.Fatalf("factory calls = %+v, want namespace demo", factory.calls)
				}
			},
		},
		{
			name: "logs pod api-0 -n demo --tail 20",
			args: []string{"logs", "pod", "api-0", "-n", "demo", "--tail", "20"},
			assert: func(t *testing.T, out, errOut string, rt *flowRuntime, factory *flowFactory) {
				t.Helper()
				if errOut != "" {
					t.Fatalf("stderr = %q, want empty", errOut)
				}
				if out != "ready to serve\n" {
					t.Fatalf("stdout = %q, want %q", out, "ready to serve\n")
				}
				if rt.getCalls != 1 || rt.getNS != "demo" || rt.getName != "api-0" {
					t.Fatalf("GetPod calls=%d ns=%q name=%q, want 1/demo/api-0", rt.getCalls, rt.getNS, rt.getName)
				}
				if rt.streamCalls != 1 || rt.streamNS != "demo" || rt.streamName != "api-0" {
					t.Fatalf("StreamPodLogs calls=%d ns=%q name=%q, want 1/demo/api-0",
						rt.streamCalls, rt.streamNS, rt.streamName)
				}
				if rt.streamOpts == nil || rt.streamOpts.TailLines == nil || *rt.streamOpts.TailLines != 20 {
					t.Fatalf("stream TailLines = %v, want 20", rt.streamOpts)
				}
				if rt.streamOpts.Container != "api" {
					t.Fatalf("stream Container = %q, want %q", rt.streamOpts.Container, "api")
				}
				if len(factory.calls) != 1 || factory.calls[0].Namespace != "demo" {
					t.Fatalf("factory calls = %+v, want namespace demo", factory.calls)
				}
			},
		},
	}

	for _, flow := range flows {
		flow := flow
		t.Run(flow.name, func(t *testing.T) {
			t.Parallel()

			rt, factory := newFlowFixture(now)

			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			cmd := newRootCommand(&App{
				Factory: factory,
				Out:     out,
				ErrOut:  errOut,
				Now:     now,
			})
			cmd.SetArgs(flow.args)

			if err := cmd.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("ExecuteContext(%v): %v\nstdout=%q\nstderr=%q", flow.args, err, out.String(), errOut.String())
			}
			flow.assert(t, out.String(), errOut.String(), rt, factory)
		})
	}
}
