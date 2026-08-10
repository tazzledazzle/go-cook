package kquick

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type fakeGetRuntime struct {
	listNamespace string
	listCalls     int
	pods          *corev1.PodList
	err           error
}

func (f *fakeGetRuntime) ListPods(_ context.Context, namespace string) (*corev1.PodList, error) {
	f.listCalls++
	f.listNamespace = namespace
	if f.err != nil {
		return nil, f.err
	}
	if f.pods == nil {
		return &corev1.PodList{}, nil
	}
	return f.pods, nil
}

func (f *fakeGetRuntime) GetPod(context.Context, string, string) (*corev1.Pod, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeGetRuntime) ListPodEvents(context.Context, string, string) (*corev1.EventList, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeGetRuntime) StreamPodLogs(context.Context, string, string, *corev1.PodLogOptions) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

type fakeGetFactory struct {
	rt        *fakeGetRuntime
	defaultNS string
	calls     int
	err       error
}

func (f *fakeGetFactory) New(_ context.Context, opts KubeOptions) (Runtime, string, error) {
	f.calls++
	if f.err != nil {
		return nil, "", f.err
	}
	ns := f.defaultNS
	if ns == "" {
		ns = "default"
	}
	if opts.Namespace != "" {
		ns = opts.Namespace
	}
	return f.rt, ns, nil
}

func TestGetPods(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)

	fixturePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "api-0",
			Namespace:         "demo",
			CreationTimestamp: metav1.NewTime(now.Add(-90 * time.Minute)),
		},
		Spec: corev1.PodSpec{NodeName: "worker-1"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "api", Ready: true, RestartCount: 2},
				{Name: "sidecar", Ready: false, RestartCount: 1},
			},
		},
	}

	pendingPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pending-0",
			Namespace:         "demo",
			CreationTimestamp: metav1.NewTime(now.Add(-5 * time.Minute)),
		},
		Status: corev1.PodStatus{Phase: corev1.PodPending},
	}

	tests := []struct {
		name       string
		args       []string
		pods       []corev1.Pod
		defaultNS  string
		rtErr      error
		wantErr    bool
		errContain string
		wantNS     string
		checkOut   func(t *testing.T, out string)
		wantCalls  int
	}{
		{
			name:      "get pods alias lists default namespace",
			args:      []string{"get", "pods"},
			pods:      []corev1.Pod{fixturePod},
			defaultNS: "demo",
			wantNS:    "demo",
			wantCalls: 1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assertTableContains(t, out, "NAME", "READY", "STATUS", "RESTARTS", "AGE", "NODE")
				assertTableContains(t, out, "api-0", "1/2", "Running", "3", "1h", "worker-1")
			},
		},
		{
			name:      "get pod alias works",
			args:      []string{"get", "pod"},
			pods:      []corev1.Pod{fixturePod},
			defaultNS: "demo",
			wantNS:    "demo",
			wantCalls: 1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				if !strings.Contains(out, "api-0") {
					t.Fatalf("output missing pod name: %q", out)
				}
			},
		},
		{
			name:      "explicit namespace is passed to ListPods",
			args:      []string{"-n", "apps", "get", "pods"},
			pods:      []corev1.Pod{},
			defaultNS: "demo",
			wantNS:    "apps",
			wantCalls: 1,
		},
		{
			name:      "pending phase and empty node name",
			args:      []string{"get", "pods"},
			pods:      []corev1.Pod{pendingPod},
			defaultNS: "demo",
			wantNS:    "demo",
			wantCalls: 1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assertTableContains(t, out, "pending-0", "0/0", "Pending", "0", "5m")
				lines := strings.Split(strings.TrimSpace(out), "\n")
				if len(lines) < 2 {
					t.Fatalf("expected header and row, got %q", out)
				}
				fields := strings.Fields(lines[1])
				// NAME READY STATUS RESTARTS AGE [NODE optional/empty]
				if len(fields) != 5 {
					t.Fatalf("expected 5 fields with empty node, got %d: %#v", len(fields), fields)
				}
			},
		},
		{
			name:      "json output is valid PodList JSON",
			args:      []string{"get", "pods", "-o", "json"},
			pods:      []corev1.Pod{fixturePod},
			defaultNS: "demo",
			wantNS:    "demo",
			wantCalls: 1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				var list corev1.PodList
				if err := json.Unmarshal([]byte(out), &list); err != nil {
					t.Fatalf("json.Unmarshal: %v\noutput: %s", err, out)
				}
				if len(list.Items) != 1 || list.Items[0].Name != "api-0" {
					t.Fatalf("decoded PodList = %+v", list.Items)
				}
			},
		},
		{
			name:      "yaml output is valid PodList YAML",
			args:      []string{"get", "pods", "-o", "yaml"},
			pods:      []corev1.Pod{fixturePod},
			defaultNS: "demo",
			wantNS:    "demo",
			wantCalls: 1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				var list corev1.PodList
				if err := yaml.Unmarshal([]byte(out), &list); err != nil {
					t.Fatalf("yaml.Unmarshal: %v\noutput: %s", err, out)
				}
				if len(list.Items) != 1 || list.Items[0].Name != "api-0" {
					t.Fatalf("decoded PodList = %+v", list.Items)
				}
			},
		},
		{
			name:       "unsupported output format fails before API call",
			args:       []string{"get", "pods", "-o", "xml"},
			defaultNS:  "demo",
			wantErr:    true,
			errContain: "output",
			wantCalls:  0,
		},
		{
			name:       "unsupported resource fails before API call",
			args:       []string{"get", "deployments"},
			defaultNS:  "demo",
			wantErr:    true,
			errContain: "resource",
			wantCalls:  0,
		},
		{
			name:       "runtime errors retain operation and namespace",
			args:       []string{"-n", "broken", "get", "pods"},
			rtErr:      errors.New("api unavailable"),
			defaultNS:  "demo",
			wantErr:    true,
			errContain: `get pods in namespace "broken"`,
			wantNS:     "broken",
			wantCalls:  1,
		},
		{
			name:      "table rows are sorted by pod name",
			args:      []string{"get", "pods"},
			defaultNS: "demo",
			wantNS:    "demo",
			wantCalls: 1,
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "zeta",
						Namespace:         "demo",
						CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
					},
					Status: corev1.PodStatus{Phase: corev1.PodRunning},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "alpha",
						Namespace:         "demo",
						CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
					},
					Status: corev1.PodStatus{Phase: corev1.PodRunning},
				},
			},
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				alpha := strings.Index(out, "alpha")
				zeta := strings.Index(out, "zeta")
				if alpha < 0 || zeta < 0 || alpha > zeta {
					t.Fatalf("expected alpha before zeta in table output: %q", out)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := &fakeGetRuntime{
				pods: &corev1.PodList{Items: tt.pods},
				err:  tt.rtErr,
			}
			factory := &fakeGetFactory{rt: rt, defaultNS: tt.defaultNS}

			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			cmd := newRootCommand(&App{
				Factory: factory,
				Out:     out,
				ErrOut:  errOut,
				Now:     now,
			})
			cmd.SetArgs(tt.args)
			err := cmd.ExecuteContext(context.Background())

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errContain != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContain) {
					t.Fatalf("error = %v, want substring %q", err, tt.errContain)
				}
			}
			if factory.calls != tt.wantCalls {
				t.Fatalf("factory calls = %d, want %d", factory.calls, tt.wantCalls)
			}
			if tt.wantCalls > 0 && tt.wantNS != "" && rt.listNamespace != tt.wantNS {
				t.Fatalf("ListPods namespace = %q, want %q", rt.listNamespace, tt.wantNS)
			}
			if tt.checkOut != nil {
				tt.checkOut(t, out.String())
			}
		})
	}
}

func assertTableContains(t *testing.T, out string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(out, part) {
			t.Fatalf("output missing %q:\n%s", part, out)
		}
	}
}
