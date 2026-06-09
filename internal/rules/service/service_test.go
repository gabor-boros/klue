package service_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/service"
	"github.com/gabor-boros/klue/pkg/resource"
)

func serviceNode(svc *corev1.Service) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindService, "v1", svc.Namespace, svc.Name, string(svc.UID)),
		Object: svc,
	}
}

func TestSelectorMismatchRule(t *testing.T) {
	t.Parallel()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "web"}},
	}

	// Graph with the service but no selected pods.
	builder := graph.NewBuilder()
	node := serviceNode(svc)
	builder.AddNode(*node)
	g := builder.Build()

	findings := (service.SelectorMismatchRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node)
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 (no pods selected)", len(findings))
	}

	// Add a selected pod edge; the finding should disappear.
	podNode := graph.Node{Ref: resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web-pod", "p1")}
	builder.AddNode(podNode)
	builder.AddEdge(graph.Edge{Kind: graph.EdgeSelectedBy, From: *node, To: podNode})
	g = builder.Build()

	if got := (service.SelectorMismatchRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 once a pod is selected", len(got))
	}
}

func sliceNode(namespace, name, serviceName string, ready bool, addresses ...string) graph.Node {
	return graph.Node{
		Ref: resource.NewReference(resource.ReferenceKindEndpointSlice, "discovery.k8s.io/v1", namespace, name, name),
		Object: &discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
				Labels:    map[string]string{discoveryv1.LabelServiceName: serviceName},
			},
			Endpoints: []discoveryv1.Endpoint{
				{Addresses: addresses, Conditions: discoveryv1.EndpointConditions{Ready: &ready}},
			},
		},
	}
}

func TestNoEndpointsRule(t *testing.T) {
	t.Parallel()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "web"}},
	}

	// Service backed by an EndpointSlice with no ready addresses.
	builder := graph.NewBuilder()
	node := serviceNode(svc)
	builder.AddNode(*node)
	notReady := sliceNode("default", "web-abc", "web", false)
	builder.AddNode(notReady)
	builder.AddEdge(graph.Edge{Kind: graph.EdgeReferences, From: notReady, To: *node})
	g := builder.Build()

	if got := (service.NoEndpointsRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 (no ready endpoints)", len(got))
	}

	// Add a ready slice; the finding should disappear.
	ready := sliceNode("default", "web-def", "web", true, "10.0.0.1")
	builder.AddNode(ready)
	builder.AddEdge(graph.Edge{Kind: graph.EdgeReferences, From: ready, To: *node})
	g = builder.Build()

	if got := (service.NoEndpointsRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 once a ready endpoint exists", len(got))
	}
}
