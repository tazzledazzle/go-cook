package kquick

import (
	"context"
	"io"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestClientGoRuntimeListPods(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "demo"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: "demo"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "other"}},
	)
	rt := &clientGoRuntime{client: client}

	list, err := rt.ListPods(context.Background(), "demo")
	if err != nil {
		t.Fatalf("ListPods: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("ListPods items = %d, want 2", len(list.Items))
	}
}

func TestClientGoRuntimeListPodsPaginates(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	var calls []string
	client.PrependReactor("list", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		listAction, ok := action.(clienttesting.ListActionImpl)
		if !ok {
			t.Fatalf("expected ListActionImpl, got %T", action)
		}
		lo := listAction.GetListOptions()
		calls = append(calls, lo.Continue)

		switch lo.Continue {
		case "":
			return true, &corev1.PodList{
				ListMeta: metav1.ListMeta{Continue: "token-2"},
				Items: []corev1.Pod{
					{ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "ns"}},
				},
			}, nil
		case "token-2":
			return true, &corev1.PodList{
				Items: []corev1.Pod{
					{ObjectMeta: metav1.ObjectMeta{Name: "p2", Namespace: "ns"}},
				},
			}, nil
		default:
			t.Fatalf("unexpected continue token %q", lo.Continue)
			return true, nil, nil
		}
	})

	rt := &clientGoRuntime{client: client}
	list, err := rt.ListPods(context.Background(), "ns")
	if err != nil {
		t.Fatalf("ListPods: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("list calls = %d, want 2 (pagination)", len(calls))
	}
	if calls[0] != "" || calls[1] != "token-2" {
		t.Fatalf("continue tokens = %#v, want [\"\", \"token-2\"]", calls)
	}
	if len(list.Items) != 2 {
		t.Fatalf("paginated items = %d, want 2", len(list.Items))
	}
	names := []string{list.Items[0].Name, list.Items[1].Name}
	if names[0] != "p1" || names[1] != "p2" {
		t.Fatalf("pod names = %v, want [p1 p2]", names)
	}
}

func TestClientGoRuntimeGetPod(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "apps"}},
	)
	rt := &clientGoRuntime{client: client}

	pod, err := rt.GetPod(context.Background(), "apps", "web")
	if err != nil {
		t.Fatalf("GetPod: %v", err)
	}
	if pod.Name != "web" {
		t.Fatalf("pod.Name = %q, want web", pod.Name)
	}
}

func TestClientGoRuntimeListPodEventsUsesFieldSelector(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	var gotSelector string
	client.PrependReactor("list", "events", func(action clienttesting.Action) (bool, runtime.Object, error) {
		listAction, ok := action.(clienttesting.ListActionImpl)
		if !ok {
			t.Fatalf("expected ListActionImpl, got %T", action)
		}
		gotSelector = listAction.GetListOptions().FieldSelector
		return true, &corev1.EventList{
			Items: []corev1.Event{
				{
					ObjectMeta:     metav1.ObjectMeta{Name: "e1", Namespace: "apps"},
					InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "web", Namespace: "apps"},
					Message:        "Scheduled",
				},
			},
		}, nil
	})

	rt := &clientGoRuntime{client: client}
	list, err := rt.ListPodEvents(context.Background(), "apps", "web")
	if err != nil {
		t.Fatalf("ListPodEvents: %v", err)
	}
	if !strings.Contains(gotSelector, "involvedObject.name=web") {
		t.Fatalf("FieldSelector = %q, want involvedObject.name=web", gotSelector)
	}
	if !strings.Contains(gotSelector, "involvedObject.kind=Pod") {
		t.Fatalf("FieldSelector = %q, want involvedObject.kind=Pod", gotSelector)
	}
	if len(list.Items) != 1 || list.Items[0].Message != "Scheduled" {
		t.Fatalf("events = %+v, want one Scheduled event", list.Items)
	}
}

func TestClientGoRuntimeListPodEventsPaginates(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	var calls int
	client.PrependReactor("list", "events", func(action clienttesting.Action) (bool, runtime.Object, error) {
		listAction, ok := action.(clienttesting.ListActionImpl)
		if !ok {
			t.Fatalf("expected ListActionImpl, got %T", action)
		}
		lo := listAction.GetListOptions()
		calls++
		switch lo.Continue {
		case "":
			return true, &corev1.EventList{
				ListMeta: metav1.ListMeta{Continue: "evt-2"},
				Items: []corev1.Event{
					{ObjectMeta: metav1.ObjectMeta{Name: "e1"}, Message: "first"},
				},
			}, nil
		case "evt-2":
			return true, &corev1.EventList{
				Items: []corev1.Event{
					{ObjectMeta: metav1.ObjectMeta{Name: "e2"}, Message: "second"},
				},
			}, nil
		default:
			t.Fatalf("unexpected continue %q", lo.Continue)
			return true, nil, nil
		}
	})

	rt := &clientGoRuntime{client: client}
	list, err := rt.ListPodEvents(context.Background(), "apps", "web")
	if err != nil {
		t.Fatalf("ListPodEvents: %v", err)
	}
	if calls != 2 {
		t.Fatalf("event list calls = %d, want 2", calls)
	}
	if len(list.Items) != 2 {
		t.Fatalf("paginated events = %d, want 2", len(list.Items))
	}
}

func TestClientGoRuntimeGetPodErrorIsContextual(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	rt := &clientGoRuntime{client: client}

	_, err := rt.GetPod(context.Background(), "apps", "missing")
	if err == nil {
		t.Fatal("expected GetPod error")
	}
	if !strings.Contains(err.Error(), `get pod "missing" in namespace "apps"`) {
		t.Fatalf("error = %q, want contextual wrap", err.Error())
	}
}

func TestClientGoRuntimeListPodEventsErrorIsContextual(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "events", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, io.ErrUnexpectedEOF
	})
	rt := &clientGoRuntime{client: client}

	_, err := rt.ListPodEvents(context.Background(), "apps", "web")
	if err == nil {
		t.Fatal("expected ListPodEvents error")
	}
	if !strings.Contains(err.Error(), `list events for pod "web" in namespace "apps"`) {
		t.Fatalf("error = %q, want contextual wrap", err.Error())
	}
}

func TestClientGoRuntimeListPodsErrorIsContextual(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, io.ErrUnexpectedEOF
	})
	rt := &clientGoRuntime{client: client}

	_, err := rt.ListPods(context.Background(), "broken")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `list pods in namespace "broken"`) {
		t.Fatalf("error = %q, want contextual wrap", err.Error())
	}
}
