package pdb_test

import (
	"testing"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/pdb"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(budget *policyv1.PodDisruptionBudget) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindPodDisruptionBudget, "policy/v1", budget.Namespace, budget.Name, string(budget.UID)),
		Object: budget,
	}
}

func TestDisruptionsBlockedRule(t *testing.T) {
	t.Parallel()

	blocked := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Status: policyv1.PodDisruptionBudgetStatus{
			ObservedGeneration: 1,
			ExpectedPods:       3,
			CurrentHealthy:     1,
			DesiredHealthy:     2,
			DisruptionsAllowed: 0,
		},
	}
	if got := (pdb.DisruptionsBlockedRule{}).Evaluate(diagnose.RuleContext{}, node(blocked)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 when no disruptions allowed", len(got))
	}

	healthy := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Status: policyv1.PodDisruptionBudgetStatus{
			ObservedGeneration: 1,
			ExpectedPods:       3,
			DisruptionsAllowed: 1,
		},
	}
	if got := (pdb.DisruptionsBlockedRule{}).Evaluate(diagnose.RuleContext{}, node(healthy)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 when disruptions allowed", len(got))
	}
}

func TestNoMatchingPodsRule(t *testing.T) {
	t.Parallel()

	budget := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec:       policyv1.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}}},
	}

	builder := graph.NewBuilder()
	builder.AddNode(*node(budget))
	g := builder.Build()
	if got := (pdb.NoMatchingPodsRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node(budget)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 when selector matches no pods", len(got))
	}

	// With a protect edge present (pod selected), there is no finding.
	builder = graph.NewBuilder()
	n := node(budget)
	pod := graph.Node{Ref: resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web-pod", "p1")}
	builder.AddNode(*n)
	builder.AddNode(pod)
	builder.AddEdge(graph.Edge{Kind: graph.EdgeProtects, From: *n, To: pod})
	g = builder.Build()
	if got := (pdb.NoMatchingPodsRule{}).Evaluate(diagnose.RuleContext{Graph: g}, n); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 when pods are selected", len(got))
	}
}
