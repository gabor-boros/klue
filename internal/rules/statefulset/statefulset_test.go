package statefulset_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/statefulset"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(sts *appsv1.StatefulSet) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindStatefulSet, "apps/v1", sts.Namespace, sts.Name, string(sts.UID)),
		Object: sts,
	}
}

func replicas(n int32) *int32 { return &n }

func TestUnavailableRule(t *testing.T) {
	t.Parallel()

	degraded := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "db"},
		Spec:       appsv1.StatefulSetSpec{Replicas: replicas(3)},
		Status:     appsv1.StatefulSetStatus{ReadyReplicas: 1},
	}
	if got := (statefulset.UnavailableRule{}).Evaluate(diagnose.RuleContext{}, node(degraded)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(got))
	}

	healthy := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "db"},
		Spec:       appsv1.StatefulSetSpec{Replicas: replicas(3)},
		Status:     appsv1.StatefulSetStatus{ReadyReplicas: 3},
	}
	if got := (statefulset.UnavailableRule{}).Evaluate(diagnose.RuleContext{}, node(healthy)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 when ready", len(got))
	}
}

func TestRolloutStuckRule(t *testing.T) {
	t.Parallel()

	stuck := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "db"},
		Spec:       appsv1.StatefulSetSpec{Replicas: replicas(3)},
		Status: appsv1.StatefulSetStatus{
			UpdateRevision:  "rev-2",
			CurrentRevision: "rev-1",
			UpdatedReplicas: 1,
		},
	}
	if got := (statefulset.RolloutStuckRule{}).Evaluate(diagnose.RuleContext{}, node(stuck)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for partial rollout", len(got))
	}

	complete := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "db"},
		Spec:       appsv1.StatefulSetSpec{Replicas: replicas(3)},
		Status: appsv1.StatefulSetStatus{
			UpdateRevision:  "rev-2",
			CurrentRevision: "rev-2",
			UpdatedReplicas: 3,
		},
	}
	if got := (statefulset.RolloutStuckRule{}).Evaluate(diagnose.RuleContext{}, node(complete)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 when rollout complete", len(got))
	}
}
