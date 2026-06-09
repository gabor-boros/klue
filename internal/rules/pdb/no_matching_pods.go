package pdb

import (
	"fmt"

	policyv1 "k8s.io/api/policy/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// NoMatchingPodsRule flags PodDisruptionBudgets whose selector matches no pods.
type NoMatchingPodsRule struct{}

// ID returns the rule identifier.
func (NoMatchingPodsRule) ID() string { return "pdb/no-matching-pods" }

// Description returns a human-readable description of the rule.
func (NoMatchingPodsRule) Description() string {
	return "Detects PodDisruptionBudgets whose selector matches no pods"
}

// AppliesTo returns the kinds this rule evaluates.
func (NoMatchingPodsRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPodDisruptionBudget}
}

// Evaluate reports budgets whose selector protects no pods in the graph.
func (r NoMatchingPodsRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	budget, ok := graph.As[*policyv1.PodDisruptionBudget](node)
	if !ok || ctx.Graph == nil || budget.Spec.Selector == nil {
		return nil
	}

	// The graph wires PDBs to selected pods via protect edges.
	for _, edge := range ctx.Graph.GetOutboundEdges(*node) {
		if edge.Kind == graph.EdgeProtects {
			return nil
		}
	}
	if budget.Status.ExpectedPods > 0 {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "PodDisruptionBudget selects no pods",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.65,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Selector", fmt.Sprintf("selector=%v matches no pods", budget.Spec.Selector.MatchLabels), ""),
			},
			Explanation: "The budget's selector matches no pods, so it provides no protection. The selector may be misconfigured.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Compare the selector with pod labels",
					Command: fmt.Sprintf("kubectl get pods -n %s --show-labels", budget.Namespace),
				},
			},
		},
	}
}
