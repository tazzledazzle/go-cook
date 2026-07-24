package kquick

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type stubRuntime struct{}

func (stubRuntime) ListPods(context.Context, string) (*corev1.PodList, error) {
	return &corev1.PodList{}, nil
}

func (stubRuntime) GetPod(_ context.Context, _, name string) (*corev1.Pod, error) {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main"}},
		},
	}, nil
}

func (stubRuntime) ListPodEvents(context.Context, string, string) (*corev1.EventList, error) {
	return &corev1.EventList{}, nil
}

func (stubRuntime) StreamPodLogs(context.Context, string, string, *corev1.PodLogOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

type recordingFactory struct {
	calls []KubeOptions
	ns    string
	err   error
}

func (f *recordingFactory) New(_ context.Context, opts KubeOptions) (Runtime, string, error) {
	f.calls = append(f.calls, opts)
	if f.err != nil {
		return nil, "", f.err
	}
	ns := f.ns
	if ns == "" {
		ns = "default"
	}
	if opts.Namespace != "" {
		ns = opts.Namespace
	}
	return stubRuntime{}, ns, nil
}

func TestRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		errContain string
		check      func(t *testing.T, out, errOut *bytes.Buffer, factory *recordingFactory, err error)
	}{
		{
			name: "root command name is kquick",
			args: []string{"--help"},
			check: func(t *testing.T, out, _ *bytes.Buffer, factory *recordingFactory, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				cmd := NewRootCommand(factory, out, io.Discard)
				if got := cmd.Name(); got != "kquick" {
					t.Fatalf("Name() = %q, want %q", got, "kquick")
				}
			},
		},
		{
			name: "help succeeds without loading kubeconfig",
			args: []string{"--help"},
			check: func(t *testing.T, out, _ *bytes.Buffer, factory *recordingFactory, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(factory.calls) != 0 {
					t.Fatalf("factory called %d times; help must not load kubeconfig", len(factory.calls))
				}
				if !strings.Contains(out.String(), "kquick") {
					t.Fatalf("help output missing command name: %q", out.String())
				}
			},
		},
		{
			name:       "unknown command fails",
			args:       []string{"unknown"},
			wantErr:    true,
			errContain: "unknown",
			check: func(t *testing.T, _, _ *bytes.Buffer, factory *recordingFactory, err error) {
				t.Helper()
				if err == nil {
					t.Fatal("expected error for unknown command")
				}
				if len(factory.calls) != 0 {
					t.Fatalf("factory should not be called for unknown command, got %d calls", len(factory.calls))
				}
			},
		},
		{
			name:       "missing get argument fails",
			args:       []string{"get"},
			wantErr:    true,
			errContain: "arg",
			check: func(t *testing.T, _, _ *bytes.Buffer, factory *recordingFactory, err error) {
				t.Helper()
				if err == nil {
					t.Fatal("expected error for missing get argument")
				}
				if len(factory.calls) != 0 {
					t.Fatalf("factory should not be called when args are missing, got %d calls", len(factory.calls))
				}
			},
		},
		{
			name:       "missing describe argument fails",
			args:       []string{"describe"},
			wantErr:    true,
			errContain: "arg",
			check: func(t *testing.T, _, _ *bytes.Buffer, factory *recordingFactory, err error) {
				t.Helper()
				if err == nil {
					t.Fatal("expected error for missing describe argument")
				}
			},
		},
		{
			name:       "missing logs argument fails",
			args:       []string{"logs"},
			wantErr:    true,
			errContain: "arg",
			check: func(t *testing.T, _, _ *bytes.Buffer, factory *recordingFactory, err error) {
				t.Helper()
				if err == nil {
					t.Fatal("expected error for missing logs argument")
				}
			},
		},
		{
			name:    "kube options reach the runtime factory",
			args:    []string{"--kubeconfig", "/tmp/test.kubeconfig", "--context", "demo-ctx", "--namespace", "demo-ns", "get", "pods"},
			wantErr: false,
			check: func(t *testing.T, _, _ *bytes.Buffer, factory *recordingFactory, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(factory.calls) != 1 {
					t.Fatalf("factory calls = %d, want 1", len(factory.calls))
				}
				got := factory.calls[0]
				want := KubeOptions{
					Kubeconfig: "/tmp/test.kubeconfig",
					Context:    "demo-ctx",
					Namespace:  "demo-ns",
				}
				if got != want {
					t.Fatalf("KubeOptions = %+v, want %+v", got, want)
				}
			},
		},
		{
			name:    "short namespace flag reaches the runtime factory",
			args:    []string{"-n", "short-ns", "get", "pods"},
			wantErr: false,
			check: func(t *testing.T, _, _ *bytes.Buffer, factory *recordingFactory, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(factory.calls) != 1 {
					t.Fatalf("factory calls = %d, want 1", len(factory.calls))
				}
				if got := factory.calls[0].Namespace; got != "short-ns" {
					t.Fatalf("Namespace = %q, want %q", got, "short-ns")
				}
			},
		},
		{
			name:       "command errors are returned instead of exiting",
			args:       []string{"get", "pods"},
			wantErr:    true,
			errContain: "factory boom",
			check: func(t *testing.T, _, errOut *bytes.Buffer, _ *recordingFactory, err error) {
				t.Helper()
				if err == nil {
					t.Fatal("expected factory error to be returned")
				}
				if !strings.Contains(err.Error(), "factory boom") {
					t.Fatalf("error = %q, want substring %q", err.Error(), "factory boom")
				}
				// SilenceUsage/SilenceErrors should keep usage off stderr for returned errors.
				if strings.Contains(errOut.String(), "Usage:") {
					t.Fatalf("usage printed to stderr despite SilenceUsage: %q", errOut.String())
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			factory := &recordingFactory{ns: "default"}
			if tt.name == "command errors are returned instead of exiting" {
				factory.err = errors.New("factory boom")
			}

			cmd := NewRootCommand(factory, out, errOut)
			cmd.SetArgs(tt.args)
			err := cmd.ExecuteContext(context.Background())

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil && tt.check == nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errContain != "" && err != nil && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContain)) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.errContain)
			}
			if tt.check != nil {
				tt.check(t, out, errOut, factory, err)
			}
		})
	}
}

func TestKubeOptionsReachFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want KubeOptions
	}{
		{
			name: "all kube flags",
			args: []string{
				"--kubeconfig", "/path/to/kubeconfig",
				"--context", "prod",
				"--namespace", "apps",
				"get", "pods",
			},
			want: KubeOptions{
				Kubeconfig: "/path/to/kubeconfig",
				Context:    "prod",
				Namespace:  "apps",
			},
		},
		{
			name: "empty options use defaults",
			args: []string{"get", "pods"},
			want: KubeOptions{},
		},
		{
			name: "short namespace flag",
			args: []string{"-n", "apps", "logs", "pod", "api-0"},
			want: KubeOptions{Namespace: "apps"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			factory := &recordingFactory{ns: "default"}
			cmd := NewRootCommand(factory, io.Discard, io.Discard)
			cmd.SetArgs(tt.args)
			if err := cmd.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("ExecuteContext: %v", err)
			}
			if len(factory.calls) != 1 {
				t.Fatalf("factory calls = %d, want 1", len(factory.calls))
			}
			if got := factory.calls[0]; got != tt.want {
				t.Fatalf("KubeOptions = %+v, want %+v", got, tt.want)
			}
		})
	}
}
