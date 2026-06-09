package kube_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestBuildGraphExtendedEdges(t *testing.T) {
	t.Parallel()

	snapshot := &kube.ResourceSnapshot{
		Namespace: "default",
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "web-pod",
				UID:       typesUID("p1"),
				Labels:    map[string]string{"app": "web"},
			}},
		},
		Deployments: []appsv1.Deployment{
			{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web", UID: typesUID("d1")}},
		},
		HorizontalPodAutoscalers: []autoscalingv2.HorizontalPodAutoscaler{
			{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-hpa", UID: typesUID("h1")},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "web",
					},
				},
			},
		},
		PodDisruptionBudgets: []policyv1.PodDisruptionBudget{
			{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-pdb", UID: typesUID("b1")},
				Spec: policyv1.PodDisruptionBudgetSpec{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
				},
			},
		},
		Roles: []rbacv1.Role{
			{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "reader", UID: typesUID("ro1")}},
		},
		RoleBindings: []rbacv1.RoleBinding{
			{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "read-binding", UID: typesUID("rb1")},
				RoleRef:    rbacv1.RoleRef{Kind: "Role", Name: "reader"},
			},
		},
	}

	g := snapshot.BuildGraph()
	edges := g.GetEdges()

	cases := []struct {
		name     string
		kind     graph.EdgeKind
		from, to string
	}{
		{"hpa scales deployment", graph.EdgeScaleTarget, "web-hpa", "web"},
		{"pdb protects pod", graph.EdgeProtects, "web-pdb", "web-pod"},
		{"rolebinding references role", graph.EdgeRoleRef, "read-binding", "reader"},
	}

	for _, tc := range cases {
		if !edgeExists(edges, tc.kind, tc.from, tc.to) {
			t.Errorf("expected edge %q: %s %s -> %s", tc.name, tc.kind, tc.from, tc.to)
		}
	}
}

func TestBuildGraphCreatesPlaceholderForMissingScaleTarget(t *testing.T) {
	t.Parallel()

	snapshot := &kube.ResourceSnapshot{
		Namespace: "default",
		HorizontalPodAutoscalers: []autoscalingv2.HorizontalPodAutoscaler{
			{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-hpa", UID: typesUID("h1")},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "missing-deploy",
					},
				},
			},
		},
	}

	g := snapshot.BuildGraph()
	edges := g.GetEdges()
	if !edgeExists(edges, graph.EdgeScaleTarget, "web-hpa", "missing-deploy") {
		t.Fatal("expected HPA to keep a scaleTarget edge to a placeholder when target is missing")
	}

	ref := resource.NewReference(resource.ReferenceKindDeployment, "apps/v1", "default", "missing-deploy", "")
	node, found := g.FindByRef(ref)
	if !found {
		t.Fatal("expected missing deployment placeholder node")
	}
	if node.Status != resource.StatusMissing {
		t.Fatalf("placeholder status = %s, want %s", node.Status, resource.StatusMissing)
	}
	if !node.IsPlaceholder() {
		t.Fatal("expected missing deployment node to be marked as placeholder")
	}
}
