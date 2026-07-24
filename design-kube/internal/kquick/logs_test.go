package kquick

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type recordingReadCloser struct {
	io.Reader
	mu       sync.Mutex
	closed   int
	closeErr error
}

func (r *recordingReadCloser) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed++
	return r.closeErr
}

func (r *recordingReadCloser) closeCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

type contextBlockingReader struct {
	ctx    context.Context
	closed int
	mu     sync.Mutex
}

func (r *contextBlockingReader) Read(p []byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

func (r *contextBlockingReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed++
	return nil
}

func (r *contextBlockingReader) closeCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed
}

type errAfterReader struct {
	data []byte
	err  error
	n    int
}

func (r *errAfterReader) Read(p []byte) (int, error) {
	if r.n < len(r.data) {
		n := copy(p, r.data[r.n:])
		r.n += n
		return n, nil
	}
	return 0, r.err
}

type fakeLogsRuntime struct {
	getNamespace    string
	getName         string
	getCalls        int
	streamNamespace string
	streamName      string
	streamCalls     int
	streamOpts      *corev1.PodLogOptions
	streamCtx       context.Context
	pod             *corev1.Pod
	stream          io.ReadCloser
	getErr          error
	streamErr       error
}

func (f *fakeLogsRuntime) ListPods(context.Context, string) (*corev1.PodList, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeLogsRuntime) GetPod(_ context.Context, namespace, name string) (*corev1.Pod, error) {
	f.getCalls++
	f.getNamespace = namespace
	f.getName = name
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.pod, nil
}

func (f *fakeLogsRuntime) ListPodEvents(context.Context, string, string) (*corev1.EventList, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeLogsRuntime) StreamPodLogs(ctx context.Context, namespace, name string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	f.streamCalls++
	f.streamNamespace = namespace
	f.streamName = name
	f.streamCtx = ctx
	if opts != nil {
		cp := *opts
		if opts.TailLines != nil {
			v := *opts.TailLines
			cp.TailLines = &v
		}
		f.streamOpts = &cp
	}
	if f.streamErr != nil {
		return nil, f.streamErr
	}
	return f.stream, nil
}

type fakeLogsFactory struct {
	rt        *fakeLogsRuntime
	defaultNS string
	calls     int
	err       error
}

func (f *fakeLogsFactory) New(_ context.Context, opts KubeOptions) (Runtime, string, error) {
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

func TestPodLogs(t *testing.T) {
	t.Parallel()

	singlePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api-0", Namespace: "demo"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "api"}},
			InitContainers: []corev1.Container{
				{Name: "init-db"},
			},
		},
	}

	multiPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api-0", Namespace: "demo"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "api"},
				{Name: "sidecar"},
			},
			InitContainers: []corev1.Container{
				{Name: "init-db"},
			},
		},
	}

	tests := []struct {
		name           string
		args           []string
		pod            *corev1.Pod
		stream         io.ReadCloser
		defaultNS      string
		getErr         error
		streamErr      error
		wantErr        bool
		errContain     string
		errContainsAll []string
		wantNS         string
		wantName       string
		wantGetCalls   int
		wantStream     int
		wantFactory    int
		wantContainer  string
		wantFollow     bool
		wantTail       *int64
		wantClosed     bool
		checkOut       func(t *testing.T, out string)
		checkStream    func(t *testing.T, rt *fakeLogsRuntime)
	}{
		{
			name:          "logs pod NAME streams single container",
			args:          []string{"logs", "pod", "api-0"},
			pod:           singlePod,
			stream:        io.NopCloser(strings.NewReader("line-one\n")),
			defaultNS:     "demo",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantFollow:    false,
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				if out != "line-one\n" {
					t.Fatalf("stdout = %q, want %q", out, "line-one\n")
				}
			},
		},
		{
			name:          "logs pods plural alias",
			args:          []string{"logs", "pods", "api-0"},
			pod:           singlePod,
			stream:        io.NopCloser(strings.NewReader("ok")),
			defaultNS:     "demo",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				if out != "ok" {
					t.Fatalf("stdout = %q, want %q", out, "ok")
				}
			},
		},
		{
			name:        "missing arguments fail",
			args:        []string{"logs", "pod"},
			wantErr:     true,
			errContain:  "arg",
			wantFactory: 0,
		},
		{
			name:        "excess arguments fail",
			args:        []string{"logs", "pod", "api-0", "extra"},
			wantErr:     true,
			errContain:  "arg",
			wantFactory: 0,
		},
		{
			name:        "unsupported resource fails before API call",
			args:        []string{"logs", "deployments", "api-0"},
			wantErr:     true,
			errContain:  "resource",
			wantFactory: 0,
		},
		{
			name:          "single container without --container auto-selects",
			args:          []string{"logs", "pod", "api-0"},
			pod:           singlePod,
			stream:        io.NopCloser(strings.NewReader("auto")),
			defaultNS:     "demo",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
		},
		{
			name:          "selected container with -c",
			args:          []string{"logs", "pod", "api-0", "-c", "sidecar"},
			pod:           multiPod,
			stream:        io.NopCloser(strings.NewReader("side")),
			defaultNS:     "demo",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "sidecar",
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
		},
		{
			name:          "selected container with --container",
			args:          []string{"logs", "pod", "api-0", "--container", "api"},
			pod:           multiPod,
			stream:        io.NopCloser(strings.NewReader("main")),
			defaultNS:     "demo",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
		},
		{
			name:           "multi-container without selection lists valid names",
			args:           []string{"logs", "pod", "api-0"},
			pod:            multiPod,
			defaultNS:      "demo",
			wantErr:        true,
			wantNS:         "demo",
			wantName:       "api-0",
			wantGetCalls:   1,
			wantStream:     0,
			wantFactory:    1,
			errContainsAll: []string{"container", "api", "sidecar"},
		},
		{
			name:           "unknown selected container lists valid names",
			args:           []string{"logs", "pod", "api-0", "-c", "missing"},
			pod:            multiPod,
			defaultNS:      "demo",
			wantErr:        true,
			wantNS:         "demo",
			wantName:       "api-0",
			wantGetCalls:   1,
			wantStream:     0,
			wantFactory:    1,
			errContainsAll: []string{"missing", "api", "sidecar"},
		},
		{
			name:          "follow reaches PodLogOptions.Follow",
			args:          []string{"logs", "pod", "api-0", "--follow"},
			pod:           singlePod,
			stream:        io.NopCloser(strings.NewReader("f")),
			defaultNS:     "demo",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantFollow:    true,
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
		},
		{
			name:          "follow short flag -f",
			args:          []string{"logs", "pod", "api-0", "-f"},
			pod:           singlePod,
			stream:        io.NopCloser(strings.NewReader("f")),
			defaultNS:     "demo",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantFollow:    true,
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
		},
		{
			name:          "tail 25 reaches PodLogOptions.TailLines",
			args:          []string{"logs", "pod", "api-0", "--tail", "25"},
			pod:           singlePod,
			stream:        io.NopCloser(strings.NewReader("t")),
			defaultNS:     "demo",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantTail:      int64Ptr(25),
			wantClosed:    true,
		},
		{
			name:         "tail below -1 rejected before API access",
			args:         []string{"logs", "pod", "api-0", "--tail", "-2"},
			pod:          singlePod,
			defaultNS:    "demo",
			wantErr:      true,
			errContain:   ">= -1",
			wantFactory:  0,
			wantGetCalls: 0,
			wantStream:   0,
		},
		{
			name:          "stream bytes copied exactly to stdout",
			args:          []string{"logs", "pod", "api-0"},
			pod:           singlePod,
			stream:        io.NopCloser(bytes.NewReader([]byte{0x00, 0x01, 0xff, 'a', '\n'})),
			defaultNS:     "demo",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				want := string([]byte{0x00, 0x01, 0xff, 'a', '\n'})
				if out != want {
					t.Fatalf("stdout = %q, want %q", out, want)
				}
			},
		},
		{
			name:         "get errors are surfaced",
			args:         []string{"-n", "broken", "logs", "pod", "api-0"},
			getErr:       errors.New("pod missing"),
			defaultNS:    "demo",
			wantErr:      true,
			errContain:   "api-0",
			wantNS:       "broken",
			wantName:     "api-0",
			wantGetCalls: 1,
			wantStream:   0,
			wantFactory:  1,
		},
		{
			name:          "stream-open errors are surfaced",
			args:          []string{"logs", "pod", "api-0"},
			pod:           singlePod,
			streamErr:     errors.New("stream unavailable"),
			defaultNS:     "demo",
			wantErr:       true,
			errContain:    "stream unavailable",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantTail:      int64Ptr(-1),
		},
		{
			name: "copy errors are surfaced",
			args: []string{"logs", "pod", "api-0"},
			pod:  singlePod,
			stream: &recordingReadCloser{
				Reader: &errAfterReader{data: []byte("partial"), err: errors.New("read boom")},
			},
			defaultNS:     "demo",
			wantErr:       true,
			errContain:    "read boom",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
		},
		{
			name: "close errors are surfaced",
			args: []string{"logs", "pod", "api-0"},
			pod:  singlePod,
			stream: &recordingReadCloser{
				Reader:   strings.NewReader("ok"),
				closeErr: errors.New("close boom"),
			},
			defaultNS:     "demo",
			wantErr:       true,
			errContain:    "close boom",
			wantNS:        "demo",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
		},
		{
			name: "copy and close errors preserve copy failure",
			args: []string{"logs", "pod", "api-0"},
			pod:  singlePod,
			stream: &recordingReadCloser{
				Reader:   &errAfterReader{err: errors.New("copy boom")},
				closeErr: errors.New("close boom"),
			},
			defaultNS:      "demo",
			wantErr:        true,
			wantNS:         "demo",
			wantName:       "api-0",
			wantGetCalls:   1,
			wantStream:     1,
			wantFactory:    1,
			wantContainer:  "api",
			wantTail:       int64Ptr(-1),
			wantClosed:     true,
			errContainsAll: []string{"copy boom", "close boom"},
			checkOut: func(t *testing.T, out string) {
				t.Helper()
				// Error message should lead with the copy failure.
			},
			checkStream: func(t *testing.T, _ *fakeLogsRuntime) {
				t.Helper()
			},
		},
		{
			name:          "explicit namespace reaches GetPod and StreamPodLogs",
			args:          []string{"-n", "apps", "logs", "pod", "api-0"},
			pod:           singlePod,
			stream:        io.NopCloser(strings.NewReader("ns")),
			defaultNS:     "demo",
			wantNS:        "apps",
			wantName:      "api-0",
			wantGetCalls:  1,
			wantStream:    1,
			wantFactory:   1,
			wantContainer: "api",
			wantTail:      int64Ptr(-1),
			wantClosed:    true,
		},
		{
			name:           "init containers are not selectable",
			args:           []string{"logs", "pod", "api-0", "-c", "init-db"},
			pod:            singlePod,
			defaultNS:      "demo",
			wantErr:        true,
			wantNS:         "demo",
			wantName:       "api-0",
			wantGetCalls:   1,
			wantStream:     0,
			wantFactory:    1,
			errContainsAll: []string{"init-db", "api"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var recorder *recordingReadCloser
			stream := tt.stream
			if stream != nil {
				if rc, ok := stream.(*recordingReadCloser); ok {
					recorder = rc
				} else {
					recorder = &recordingReadCloser{Reader: stream}
					stream = recorder
				}
			}

			rt := &fakeLogsRuntime{
				pod:       tt.pod,
				stream:    stream,
				getErr:    tt.getErr,
				streamErr: tt.streamErr,
			}
			factory := &fakeLogsFactory{rt: rt, defaultNS: tt.defaultNS}

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
			for _, part := range tt.errContainsAll {
				if err == nil || !strings.Contains(err.Error(), part) {
					t.Fatalf("error = %v, want substring %q", err, part)
				}
			}
			if tt.name == "copy and close errors preserve copy failure" {
				if err == nil {
					t.Fatal("expected error")
				}
				msg := err.Error()
				copyIdx := strings.Index(msg, "copy boom")
				closeIdx := strings.Index(msg, "close boom")
				if copyIdx < 0 || closeIdx < 0 || copyIdx > closeIdx {
					t.Fatalf("expected copy boom before close boom in %q", msg)
				}
			}
			if factory.calls != tt.wantFactory {
				t.Fatalf("factory calls = %d, want %d", factory.calls, tt.wantFactory)
			}
			if rt.getCalls != tt.wantGetCalls {
				t.Fatalf("GetPod calls = %d, want %d", rt.getCalls, tt.wantGetCalls)
			}
			if rt.streamCalls != tt.wantStream {
				t.Fatalf("StreamPodLogs calls = %d, want %d", rt.streamCalls, tt.wantStream)
			}
			if tt.wantGetCalls > 0 {
				if rt.getNamespace != tt.wantNS {
					t.Fatalf("GetPod namespace = %q, want %q", rt.getNamespace, tt.wantNS)
				}
				if rt.getName != tt.wantName {
					t.Fatalf("GetPod name = %q, want %q", rt.getName, tt.wantName)
				}
			}
			if tt.wantStream > 0 {
				if rt.streamNamespace != tt.wantNS {
					t.Fatalf("StreamPodLogs namespace = %q, want %q", rt.streamNamespace, tt.wantNS)
				}
				if rt.streamName != tt.wantName {
					t.Fatalf("StreamPodLogs name = %q, want %q", rt.streamName, tt.wantName)
				}
				if rt.streamOpts == nil {
					t.Fatal("StreamPodLogs options are nil")
				}
				if rt.streamOpts.Container != tt.wantContainer {
					t.Fatalf("Container = %q, want %q", rt.streamOpts.Container, tt.wantContainer)
				}
				if rt.streamOpts.Follow != tt.wantFollow {
					t.Fatalf("Follow = %v, want %v", rt.streamOpts.Follow, tt.wantFollow)
				}
				if tt.wantTail == nil {
					if rt.streamOpts.TailLines != nil {
						t.Fatalf("TailLines = %v, want nil", *rt.streamOpts.TailLines)
					}
				} else {
					if rt.streamOpts.TailLines == nil {
						t.Fatal("TailLines is nil")
					}
					if *rt.streamOpts.TailLines != *tt.wantTail {
						t.Fatalf("TailLines = %d, want %d", *rt.streamOpts.TailLines, *tt.wantTail)
					}
				}
			}
			if tt.wantClosed {
				if recorder == nil || recorder.closeCount() != 1 {
					got := 0
					if recorder != nil {
						got = recorder.closeCount()
					}
					t.Fatalf("stream Close calls = %d, want 1", got)
				}
			}
			if tt.checkOut != nil {
				tt.checkOut(t, out.String())
			}
			if tt.checkStream != nil {
				tt.checkStream(t, rt)
			}
		})
	}
}

func TestPodLogsCancellationTerminatesBlockedStream(t *testing.T) {
	t.Parallel()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "api-0", Namespace: "demo"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "api"}},
		},
	}

	started := make(chan struct{})
	blockingRT := &blockingLogsRuntime{
		inner:   &fakeLogsRuntime{pod: pod},
		started: started,
	}
	blockingFactory := &blockingLogsFactory{
		rt:        blockingRT,
		defaultNS: "demo",
	}

	out := &bytes.Buffer{}
	cmd := newRootCommand(&App{
		Factory: blockingFactory,
		Out:     out,
		ErrOut:  io.Discard,
	})
	cmd.SetArgs([]string{"logs", "pod", "api-0", "-f"})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.ExecuteContext(ctx)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for blocked stream to start")
	}

	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected cancellation error, got nil")
		}
		if !errors.Is(err, context.Canceled) && !strings.Contains(strings.ToLower(err.Error()), "cancel") {
			t.Fatalf("error = %v, want cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for command to return after cancel")
	}

	if blockingRT.blocker == nil || blockingRT.blocker.closeCount() != 1 {
		got := 0
		if blockingRT.blocker != nil {
			got = blockingRT.blocker.closeCount()
		}
		t.Fatalf("stream Close calls = %d, want 1", got)
	}
}

type blockingLogsRuntime struct {
	inner   *fakeLogsRuntime
	started chan struct{}
	blocker *contextBlockingReader
	once    sync.Once
}

func (f *blockingLogsRuntime) ListPods(context.Context, string) (*corev1.PodList, error) {
	return nil, errors.New("not implemented")
}

func (f *blockingLogsRuntime) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return f.inner.GetPod(ctx, namespace, name)
}

func (f *blockingLogsRuntime) ListPodEvents(context.Context, string, string) (*corev1.EventList, error) {
	return nil, errors.New("not implemented")
}

func (f *blockingLogsRuntime) StreamPodLogs(ctx context.Context, namespace, name string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	f.inner.streamCalls++
	f.inner.streamNamespace = namespace
	f.inner.streamName = name
	if opts != nil {
		cp := *opts
		if opts.TailLines != nil {
			v := *opts.TailLines
			cp.TailLines = &v
		}
		f.inner.streamOpts = &cp
	}
	f.blocker = &contextBlockingReader{ctx: ctx}
	f.once.Do(func() { close(f.started) })
	return f.blocker, nil
}

type blockingLogsFactory struct {
	rt        *blockingLogsRuntime
	defaultNS string
	calls     int
}

func (f *blockingLogsFactory) New(_ context.Context, opts KubeOptions) (Runtime, string, error) {
	f.calls++
	ns := f.defaultNS
	if opts.Namespace != "" {
		ns = opts.Namespace
	}
	return f.rt, ns, nil
}

func int64Ptr(v int64) *int64 { return &v }
