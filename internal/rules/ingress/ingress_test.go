package ingress_test

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/ingress"
	"github.com/gabor-boros/klue/pkg/resource"
)

func ingressNode(ing *networkingv1.Ingress) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindIngress, "networking.k8s.io/v1", ing.Namespace, ing.Name, string(ing.UID)),
		Object: ing,
	}
}

func TestBackendMissingRule(t *testing.T) {
	t.Parallel()

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-ing"},
		Spec: networkingv1.IngressSpec{
			DefaultBackend: &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{Name: "web"},
			},
		},
	}

	builder := graph.NewBuilder()
	builder.AddNode(*ingressNode(ing))
	g := builder.Build()

	if got := (ingress.BackendMissingRule{}).Evaluate(diagnose.RuleContext{Graph: g}, ingressNode(ing)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 (missing backend service)", len(got))
	}

	builder.AddNode(graph.Node{Ref: resource.NewReference(resource.ReferenceKindService, "v1", "default", "web", "")})
	g = builder.Build()
	if got := (ingress.BackendMissingRule{}).Evaluate(diagnose.RuleContext{Graph: g}, ingressNode(ing)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 once backend service exists", len(got))
	}
}
