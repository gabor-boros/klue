package kube_test

import (
	"context"
	"errors"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/gabor-boros/klue/internal/kube"
)

func meta(namespace, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Namespace: namespace, Name: name}
}

func TestClientFetch(t *testing.T) {
	t.Parallel()

	clientset := fake.NewClientset(
		&corev1.Pod{ObjectMeta: meta("default", "pod-a")},
		&corev1.Pod{ObjectMeta: meta("default", "pod-b")},
		&corev1.Pod{ObjectMeta: meta("other", "pod-c")},
		&corev1.Service{ObjectMeta: meta("default", "svc-a")},
		&appsv1.Deployment{ObjectMeta: meta("default", "deploy-a")},
		&corev1.Node{ObjectMeta: meta("", "node-1")},
		&corev1.PersistentVolume{ObjectMeta: meta("", "pv-1")},
		&storagev1.StorageClass{ObjectMeta: meta("", "standard")},
	)

	client := kube.NewClientForInterface(clientset)

	snapshot, err := client.Fetch(context.Background(), "default")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if got := len(snapshot.Pods); got != 2 {
		t.Errorf("Pods = %d, want 2 (namespace scoping failed)", got)
	}
	if got := len(snapshot.Services); got != 1 {
		t.Errorf("Services = %d, want 1", got)
	}
	if got := len(snapshot.Deployments); got != 1 {
		t.Errorf("Deployments = %d, want 1", got)
	}
	if got := len(snapshot.Nodes); got != 1 {
		t.Errorf("Nodes = %d, want 1", got)
	}
	if got := len(snapshot.PersistentVolumes); got != 1 {
		t.Errorf("PersistentVolumes = %d, want 1", got)
	}
	if got := len(snapshot.StorageClasses); got != 1 {
		t.Errorf("StorageClasses = %d, want 1", got)
	}
	if snapshot.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", snapshot.Namespace, "default")
	}
}

func TestClientFetchFailsOnCoreListError(t *testing.T) {
	t.Parallel()

	clientset := fake.NewClientset()
	clientset.PrependReactor("list", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("pods list failed")
	})

	client := kube.NewClientForInterface(clientset)

	if _, err := client.Fetch(context.Background(), "default"); err == nil {
		t.Fatal("Fetch() error = nil, want core list failure")
	}
}
