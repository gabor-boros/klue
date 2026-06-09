package kube_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

func uid(value string) types.UID {
	return types.UID(value)
}

func TestNodeStatusRecognition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		node   corev1.Node
		expect resource.Status
	}{
		{
			name: "ready node",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "n-ready", UID: uid("n-ready")},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
				},
			},
			expect: resource.StatusReady,
		},
		{
			name: "not ready node",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "n-notready", UID: uid("n-notready")},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
					},
				},
			},
			expect: resource.StatusNotReady,
		},
		{
			name: "pressure node",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "n-pressure", UID: uid("n-pressure")},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
						{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue},
					},
				},
			},
			expect: resource.StatusDegraded,
		},
		{
			name: "unschedulable node",
			node: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "n-unsched", UID: uid("n-unsched")},
				Spec:       corev1.NodeSpec{Unschedulable: true},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
				},
			},
			expect: resource.StatusSuspended,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			snapshot := &kube.ResourceSnapshot{Nodes: []corev1.Node{tt.node}}
			g := snapshot.BuildGraph()

			ref := resource.NewReference(resource.ReferenceKindNode, "v1", "", tt.node.Name, "")
			node, found := g.FindByRef(ref)
			if !found {
				t.Fatalf("FindByRef() could not find node %q", tt.node.Name)
			}
			if node.Status != tt.expect {
				t.Fatalf("status = %s, want %s", node.Status, tt.expect)
			}
		})
	}
}

func TestPodAndPVCStatusRecognition(t *testing.T) {
	t.Parallel()

	readyPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ready-pod", UID: uid("p-ready")},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}

	terminatingPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "term-pod",
			UID:               uid("p-term"),
			DeletionTimestamp: &metav1.Time{Time: metav1.Now().Time},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	boundPVC := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "data", UID: uid("pvc-1")},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web", UID: uid("dep-1")},
		Spec:       appsv1.DeploymentSpec{Replicas: int32Ptr(3)},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 3,
			UpdatedReplicas:   3,
		},
	}

	snapshot := &kube.ResourceSnapshot{
		Pods:                   []corev1.Pod{readyPod, terminatingPod},
		PersistentVolumeClaims: []corev1.PersistentVolumeClaim{boundPVC},
		Deployments:            []appsv1.Deployment{deployment},
	}
	g := snapshot.BuildGraph()

	check := func(kind resource.Kind, apiVersion, namespace, name string, expect resource.Status) {
		t.Helper()
		ref := resource.NewReference(kind, apiVersion, namespace, name, "")
		node, found := g.FindByRef(ref)
		if !found {
			t.Fatalf("FindByRef() could not find %s/%s", kind, name)
		}
		if node.Status != expect {
			t.Fatalf("%s/%s status = %s, want %s", kind, name, node.Status, expect)
		}
	}

	check(resource.ReferenceKindPod, "v1", "default", "ready-pod", resource.StatusReady)
	check(resource.ReferenceKindPod, "v1", "default", "term-pod", resource.StatusTerminating)
	check(resource.ReferenceKindPersistentVolumeClaim, "v1", "default", "data", resource.StatusReady)
	check(resource.ReferenceKindDeployment, "apps/v1", "default", "web", resource.StatusReady)
}

func int32Ptr(value int32) *int32 {
	return &value
}
