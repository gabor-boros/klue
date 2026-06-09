// Package service contains diagnostic rules for Services.
package service

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// selectedPods returns the pod nodes selected by the service node.
func selectedPods(g *graph.Graph, node *graph.Node) []graph.Node {
	if g == nil {
		return nil
	}

	var pods []graph.Node
	for _, edge := range g.GetOutboundEdges(*node) {
		if edge.Kind == graph.EdgeSelectedBy {
			pods = append(pods, edge.To)
		}
	}

	return pods
}

// SelectorMismatchRule flags services whose selector matches no pods.
type SelectorMismatchRule struct{}

// ID returns the rule identifier.
func (SelectorMismatchRule) ID() string { return "service/selector-mismatch" }

// Description returns a human-readable description of the rule.
func (SelectorMismatchRule) Description() string {
	return "Detects services whose selector matches no pods"
}

// AppliesTo returns the kinds this rule evaluates.
func (SelectorMismatchRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindService}
}

// Evaluate checks whether the service selects any pod.
func (r SelectorMismatchRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	svc, ok := graph.As[*corev1.Service](node)
	if !ok || len(svc.Spec.Selector) == 0 {
		return nil
	}

	if len(selectedPods(ctx.Graph, node)) > 0 {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Service selector matches no pods",
			Severity:   diagnose.SeverityError,
			Confidence: 0.8,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Selector", fmt.Sprintf("selector=%v", svc.Spec.Selector), ""),
			},
			Explanation: "No pod in the namespace carries all of the labels in the service selector.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Compare the selector with pod labels",
					Command: fmt.Sprintf("kubectl get pods -n %s --show-labels", svc.Namespace),
				},
			},
		},
	}
}
