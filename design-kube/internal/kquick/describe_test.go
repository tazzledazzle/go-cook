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

type fakeDescribeRuntime struct {
	getNamespace    string
	getName         string
	getCalls        int
	eventsNamespace string
	eventsName      string
	eventsCalls     int
	pod             *corev1.Pod
	events          *corev1.EventList
	getErr          error
	eventsErr       error
}

func (f *fakeDescribeRuntime) ListPods(context.Context, string) (*corev1.PodList, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeDescribeRuntime) GetPod(_ context.Context, namespace, name string) (*corev1.Pod, error) {
	f.getCalls++
	f.getNamespace = namespace
	f.getName = name
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.pod, nil
}

func (f *fakeDescribeRuntime) ListPodEvents(_ context.Context, namespace, podName string) (*corev1.EventList, error) {
	f.eventsCalls++
	f.eventsNamespace = namespace
	f.eventsName = podName
	if f.eventsErr != nil {
		return nil, f.eventsErr
	}
	if f.events == nil {
		return &corev1.EventList{}, nil
	}
	return f.events, nil
}

func (f *fakeDescribeRuntime) StreamPodLogs(context.Context, string, string, *corev1.PodLogOptions) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

type fakeDescribeFactory struct {
	rt        *fakeDescribeRuntime
	defaultNS string
	calls     int
	err       error
}

func (f *fakeDescribeFactory) New(_ context.Context, opts KubeOptions) (Runtime, string, error) {
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

func TestDescribePod(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	earlier := metav1.NewTime(ts.Add(-10 * time.Minute))
	later := metav1.NewTime(ts.Add(-2 * time.Minute))
	eventTime := metav1.NewMicroTime(ts.Add(-5 * time.Minute))

	fullPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-0",
			Namespace: "demo",
			Labels: map[string]string{
				"tier": "frontend",
				"app":  "api",
			},
		},
		Spec: corev1.PodSpec{NodeName: "worker-1"},
		Status: corev1.PodStatus{
			Phase:  corev1.PodRunning,
			PodIP:  "10.244.0.5",
			HostIP: "192.168.1.10",
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "api",
					Image:        "nginx:1.25",
					Ready:        true,
					RestartCount: 2,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.NewTime(ts.Add(-time.Hour)),
						},
					},
				},
			},
			Conditions: []corev1.PodCondition{
				{
					Type:    corev1.PodReady,
					Status:  corev1.ConditionTrue,
					Reason:  "ContainersReady",
					Message: "all containers ready",
				},
				{
					Type:   corev1.PodScheduled,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	tests := []struct {
		name         string
		args         []string
		pod          *corev1.Pod
		events       []corev1.Event
		defaultNS    string
		getErr       error
		eventsErr    error
		wantErr      bool
		errContain   string
		wantNS       string
		wantName     string
		wantGetCalls int
		wantEvtCalls int
		wantFactory  int
		checkOut     func(t *testing.T, out string)
	}{
		{
			name:         "describe pod NAME",
			args:         []string{"describe", "pod", "api-0"},
			pod:          fullPod,
			defaultNS:    "demo",
			wantNS:       "demo",
			wantName:     "api-0",
			wantGetCalls: 1,
			wantEvtCalls: 1,
			wantFactory:  1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assertContainsLines(t, out,
					"Name:", "api-0",
					"Namespace:", "demo",
					"Status:", "Running",
					"Pod IP:", "10.244.0.5",
					"Node:", "worker-1",
				)
				assertLineHas(t, out, "Labels:", "app=api")
				if !strings.Contains(out, "tier=frontend") {
					t.Fatalf("output missing tier=frontend:\n%s", out)
				}
				// Labels must be sorted by key: app before tier.
				appIdx := strings.Index(out, "app=api")
				tierIdx := strings.Index(out, "tier=frontend")
				if appIdx < 0 || tierIdx < 0 || appIdx > tierIdx {
					t.Fatalf("expected app=api before tier=frontend:\n%s", out)
				}
			},
		},
		{
			name:         "describe pods plural alias",
			args:         []string{"describe", "pods", "api-0"},
			pod:          fullPod,
			defaultNS:    "demo",
			wantNS:       "demo",
			wantName:     "api-0",
			wantGetCalls: 1,
			wantEvtCalls: 1,
			wantFactory:  1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assertContainsLines(t, out, "Name:", "api-0")
			},
		},
		{
			name:        "missing arguments fail",
			args:        []string{"describe", "pod"},
			wantErr:     true,
			errContain:  "arg",
			wantFactory: 0,
		},
		{
			name:        "excess arguments fail",
			args:        []string{"describe", "pod", "api-0", "extra"},
			wantErr:     true,
			errContain:  "arg",
			wantFactory: 0,
		},
		{
			name:        "unsupported resource rejected before API calls",
			args:        []string{"describe", "deployments", "api-0"},
			wantErr:     true,
			errContain:  "resource",
			wantFactory: 0,
		},
		{
			name:         "explicit namespace is propagated",
			args:         []string{"-n", "apps", "describe", "pod", "api-0"},
			pod:          clonePodWithNS(fullPod, "apps"),
			defaultNS:    "demo",
			wantNS:       "apps",
			wantName:     "api-0",
			wantGetCalls: 1,
			wantEvtCalls: 1,
			wantFactory:  1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assertLineHas(t, out, "Namespace:", "apps")
			},
		},
		{
			name:         "container fields are rendered",
			args:         []string{"describe", "pod", "api-0"},
			pod:          fullPod,
			defaultNS:    "demo",
			wantNS:       "demo",
			wantName:     "api-0",
			wantGetCalls: 1,
			wantEvtCalls: 1,
			wantFactory:  1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assertContainsLines(t, out,
					"Containers:",
					"api",
					"Image:", "nginx:1.25",
					"Ready:", "True",
					"Restarts:", "2",
					"State:", "Running",
				)
			},
		},
		{
			name:         "conditions are rendered",
			args:         []string{"describe", "pod", "api-0"},
			pod:          fullPod,
			defaultNS:    "demo",
			wantNS:       "demo",
			wantName:     "api-0",
			wantGetCalls: 1,
			wantEvtCalls: 1,
			wantFactory:  1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assertContainsLines(t, out,
					"Conditions:",
					"Ready", "True", "ContainersReady", "all containers ready",
					"PodScheduled", "True",
				)
			},
		},
		{
			name: "events are sorted chronologically",
			args: []string{"describe", "pod", "api-0"},
			pod:  fullPod,
			events: []corev1.Event{
				{
					ObjectMeta:    metav1.ObjectMeta{Name: "e-late"},
					Reason:        "Pulled",
					Message:       "pulled image",
					Type:          corev1.EventTypeNormal,
					LastTimestamp: later,
				},
				{
					ObjectMeta:    metav1.ObjectMeta{Name: "e-early"},
					Reason:        "Scheduled",
					Message:       "assigned to node",
					Type:          corev1.EventTypeNormal,
					LastTimestamp: earlier,
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "e-mid"},
					Reason:     "Started",
					Message:    "started container",
					Type:       corev1.EventTypeNormal,
					EventTime:  eventTime,
				},
			},
			defaultNS:    "demo",
			wantNS:       "demo",
			wantName:     "api-0",
			wantGetCalls: 1,
			wantEvtCalls: 1,
			wantFactory:  1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				scheduled := strings.Index(out, "Scheduled")
				started := strings.Index(out, "Started")
				pulled := strings.Index(out, "Pulled")
				if scheduled < 0 || started < 0 || pulled < 0 {
					t.Fatalf("missing event reasons:\n%s", out)
				}
				if !(scheduled < started && started < pulled) {
					t.Fatalf("events not chronological (Scheduled, Started, Pulled):\n%s", out)
				}
			},
		},
		{
			name: "no labels conditions or events show none",
			args: []string{"describe", "pod", "bare"},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "bare", Namespace: "demo"},
				Status:     corev1.PodStatus{Phase: corev1.PodPending},
			},
			defaultNS:    "demo",
			wantNS:       "demo",
			wantName:     "bare",
			wantGetCalls: 1,
			wantEvtCalls: 1,
			wantFactory:  1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assertLineHas(t, out, "Labels:", "<none>")
				assertLineHas(t, out, "Pod IP:", "<none>")
				assertLineHas(t, out, "Node:", "<none>")
				assertLineHas(t, out, "Containers:", "<none>")
				assertLineHas(t, out, "Conditions:", "<none>")
				assertLineHas(t, out, "Events:", "<none>")
			},
		},
		{
			name: "waiting container state",
			args: []string{"describe", "pod", "wait-0"},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "wait-0", Namespace: "demo"},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:  "initish",
							Image: "busybox:1.36",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{Reason: "ContainerCreating"},
								// Running/Terminated also set to verify waiting wins.
								Running:    &corev1.ContainerStateRunning{},
								Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Reason: "Error"},
							},
						},
					},
				},
			},
			defaultNS:    "demo",
			wantNS:       "demo",
			wantName:     "wait-0",
			wantGetCalls: 1,
			wantEvtCalls: 1,
			wantFactory:  1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assertContainsLines(t, out, "State:", "Waiting", "ContainerCreating")
			},
		},
		{
			name: "terminated container state preferred over empty",
			args: []string{"describe", "pod", "term-0"},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "term-0", Namespace: "demo"},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:  "job",
							Image: "busybox:1.36",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 137,
									Reason:   "OOMKilled",
								},
							},
						},
					},
				},
			},
			defaultNS:    "demo",
			wantNS:       "demo",
			wantName:     "term-0",
			wantGetCalls: 1,
			wantEvtCalls: 1,
			wantFactory:  1,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				assertContainsLines(t, out, "State:", "Terminated", "OOMKilled")
			},
		},
		{
			name:         "pod get failure identifies the pod",
			args:         []string{"-n", "broken", "describe", "pod", "api-0"},
			getErr:       errors.New("not found"),
			defaultNS:    "demo",
			wantErr:      true,
			errContain:   `describe pod "api-0"`,
			wantNS:       "broken",
			wantName:     "api-0",
			wantGetCalls: 1,
			wantEvtCalls: 0,
			wantFactory:  1,
		},
		{
			name:         "event list failure identifies the pod",
			args:         []string{"describe", "pod", "api-0"},
			pod:          fullPod,
			eventsErr:    errors.New("events unavailable"),
			defaultNS:    "demo",
			wantErr:      true,
			errContain:   `describe pod "api-0"`,
			wantNS:       "demo",
			wantName:     "api-0",
			wantGetCalls: 1,
			wantEvtCalls: 1,
			wantFactory:  1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rt := &fakeDescribeRuntime{
				pod:       tt.pod,
				events:    &corev1.EventList{Items: tt.events},
				getErr:    tt.getErr,
				eventsErr: tt.eventsErr,
			}
			factory := &fakeDescribeFactory{rt: rt, defaultNS: tt.defaultNS}

			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			cmd := newRootCommand(&App{
				Factory: factory,
				Out:     out,
				ErrOut:  errOut,
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
			if factory.calls != tt.wantFactory {
				t.Fatalf("factory calls = %d, want %d", factory.calls, tt.wantFactory)
			}
			if rt.getCalls != tt.wantGetCalls {
				t.Fatalf("GetPod calls = %d, want %d", rt.getCalls, tt.wantGetCalls)
			}
			if rt.eventsCalls != tt.wantEvtCalls {
				t.Fatalf("ListPodEvents calls = %d, want %d", rt.eventsCalls, tt.wantEvtCalls)
			}
			if tt.wantGetCalls > 0 {
				if tt.wantNS != "" && rt.getNamespace != tt.wantNS {
					t.Fatalf("GetPod namespace = %q, want %q", rt.getNamespace, tt.wantNS)
				}
				if tt.wantName != "" && rt.getName != tt.wantName {
					t.Fatalf("GetPod name = %q, want %q", rt.getName, tt.wantName)
				}
			}
			if tt.wantEvtCalls > 0 {
				if tt.wantNS != "" && rt.eventsNamespace != tt.wantNS {
					t.Fatalf("ListPodEvents namespace = %q, want %q", rt.eventsNamespace, tt.wantNS)
				}
				if tt.wantName != "" && rt.eventsName != tt.wantName {
					t.Fatalf("ListPodEvents name = %q, want %q", rt.eventsName, tt.wantName)
				}
			}
			if tt.checkOut != nil {
				tt.checkOut(t, out.String())
			}
		})
	}
}

func clonePodWithNS(pod *corev1.Pod, ns string) *corev1.Pod {
	cp := pod.DeepCopy()
	cp.Namespace = ns
	return cp
}

func assertContainsLines(t *testing.T, out string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(out, part) {
			t.Fatalf("output missing %q:\n%s", part, out)
		}
	}
}

func assertLineHas(t *testing.T, out, key, value string) {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, key) && strings.Contains(line, value) {
			return
		}
	}
	t.Fatalf("no line contains both %q and %q:\n%s", key, value, out)
}
