package kquick

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// KubeOptions holds kubeconfig selection flags for a command invocation.
type KubeOptions struct {
	Kubeconfig string
	Context    string
	Namespace  string
}

// Runtime is the cluster access boundary used by version-one commands.
type Runtime interface {
	ListPods(context.Context, string) (*corev1.PodList, error)
	GetPod(context.Context, string, string) (*corev1.Pod, error)
	ListPodEvents(context.Context, string, string) (*corev1.EventList, error)
	StreamPodLogs(context.Context, string, string, *corev1.PodLogOptions) (io.ReadCloser, error)
}

// RuntimeFactory creates a Runtime from kubeconfig options.
type RuntimeFactory interface {
	New(context.Context, KubeOptions) (Runtime, string, error)
}

// ClientGoFactory builds a typed client-go Runtime from kubeconfig.
type ClientGoFactory struct{}

// New loads kubeconfig, resolves the active namespace, and returns a Runtime.
func (ClientGoFactory) New(_ context.Context, opts KubeOptions) (Runtime, string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.Kubeconfig != "" {
		loadingRules.ExplicitPath = opts.Kubeconfig
	}

	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: opts.Context,
		Context: clientcmdapi.Context{
			Namespace: opts.Namespace,
		},
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, "", fmt.Errorf("load kubeconfig: %w", err)
	}

	ns, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, "", fmt.Errorf("resolve namespace: %w", err)
	}
	if ns == "" {
		ns = "default"
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, "", fmt.Errorf("create kubernetes client: %w", err)
	}

	return &clientGoRuntime{client: clientset}, ns, nil
}

type clientGoRuntime struct {
	client kubernetes.Interface
}

func (r *clientGoRuntime) ListPods(ctx context.Context, namespace string) (*corev1.PodList, error) {
	list, err := r.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods in namespace %q: %w", namespace, err)
	}
	return list, nil
}

func (r *clientGoRuntime) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	pod, err := r.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get pod %q in namespace %q: %w", name, namespace, err)
	}
	return pod, nil
}

func (r *clientGoRuntime) ListPodEvents(ctx context.Context, namespace, podName string) (*corev1.EventList, error) {
	list, err := r.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list events for pod %q in namespace %q: %w", podName, namespace, err)
	}

	filtered := &corev1.EventList{
		TypeMeta: list.TypeMeta,
		ListMeta: list.ListMeta,
		Items:    make([]corev1.Event, 0, len(list.Items)),
	}
	for _, event := range list.Items {
		ref := event.InvolvedObject
		if ref.Kind == "Pod" && ref.Name == podName && ref.Namespace == namespace {
			filtered.Items = append(filtered.Items, event)
		}
	}
	return filtered, nil
}

func (r *clientGoRuntime) StreamPodLogs(ctx context.Context, namespace, name string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
	stream, err := r.client.CoreV1().Pods(namespace).GetLogs(name, opts).Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("stream logs for pod %q in namespace %q: %w", name, namespace, err)
	}
	return stream, nil
}
