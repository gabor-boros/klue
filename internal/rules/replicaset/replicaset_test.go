package replicaset_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/replicaset"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(rs *appsv1.ReplicaSet) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindReplicaSet, "apps/v1", rs.Namespace, rs.Name, string(rs.UID)),
		Object: rs,
	}
}

func replicas(n int32) *int32 { return &n }

func TestUnavailableRule(t *testing.T) {
	t.Parallel()

	degraded := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-rs"},
		Spec:       appsv1.ReplicaSetSpec{Replicas: replicas(2)},
		Status:     appsv1.ReplicaSetStatus{ReadyReplicas: 0},
	}
	if got := (replicaset.UnavailableRule{}).Evaluate(diagnose.RuleContext{}, node(degraded)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(got))
	}

	healthy := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-rs"},
		Spec:       appsv1.ReplicaSetSpec{Replicas: replicas(2)},
		Status:     appsv1.ReplicaSetStatus{ReadyReplicas: 2},
	}
	if got := (replicaset.UnavailableRule{}).Evaluate(diagnose.RuleContext{}, node(healthy)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 when ready", len(got))
	}
}

func TestReplicaFailureRule(t *testing.T) {
	t.Parallel()

	failing := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-rs"},
		Status: appsv1.ReplicaSetStatus{
			Conditions: []appsv1.ReplicaSetCondition{
				{Type: appsv1.ReplicaSetReplicaFailure, Status: "True", Reason: "FailedCreate", Message: "exceeded quota"},
			},
		},
	}
	if got := (replicaset.ReplicaFailureRule{}).Evaluate(diagnose.RuleContext{}, node(failing)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for replica failure", len(got))
	}

	ok := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-rs"}}
	if got := (replicaset.ReplicaFailureRule{}).Evaluate(diagnose.RuleContext{}, node(ok)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 without ReplicaFailure", len(got))
	}
}
