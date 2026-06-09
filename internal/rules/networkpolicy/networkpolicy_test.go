package networkpolicy_test

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/networkpolicy"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(np *networkingv1.NetworkPolicy) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindNetworkPolicy, "networking.k8s.io/v1", np.Namespace, np.Name, string(np.UID)),
		Object: np,
	}
}

func TestNoMatchingPodsRule(t *testing.T) {
	t.Parallel()

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "deny"},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
		},
	}

	builder := graph.NewBuilder()
	builder.AddNode(*node(np))
	g := builder.Build()
	if got := (networkpolicy.NoMatchingPodsRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node(np)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 when selector matches no pods", len(got))
	}

	// An empty selector applies to all pods and is not flagged.
	all := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "default-deny"},
	}
	builder = graph.NewBuilder()
	builder.AddNode(*node(all))
	g = builder.Build()
	if got := (networkpolicy.NoMatchingPodsRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node(all)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for an empty (all pods) selector", len(got))
	}
}
