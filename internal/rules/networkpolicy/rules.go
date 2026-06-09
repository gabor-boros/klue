// Package networkpolicy contains diagnostic rules for NetworkPolicies.
package networkpolicy

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// NoMatchingPodsRule flags NetworkPolicies whose pod selector matches no pods,
// so the policy has no effect.
type NoMatchingPodsRule struct{}

func (NoMatchingPodsRule) ID() string { return "networkpolicy/no-matching-pods" }

func (NoMatchingPodsRule) Description() string {
	return "Detects NetworkPolicies whose selector matches no pods"
}

func (NoMatchingPodsRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindNetworkPolicy}
}

func (r NoMatchingPodsRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	np, ok := graph.As[*networkingv1.NetworkPolicy](node)
	if !ok || ctx.Graph == nil {
		return nil
	}

	// An empty pod selector applies to all pods in the namespace, so it is not
	// a misconfiguration even if the graph happens to hold no pods.
	if len(np.Spec.PodSelector.MatchLabels) == 0 && len(np.Spec.PodSelector.MatchExpressions) == 0 {
		return nil
	}

	for _, edge := range ctx.Graph.GetOutboundEdges(*node) {
		if edge.Kind == graph.EdgeProtects {
			return nil
		}
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "NetworkPolicy selects no pods",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.6,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Selector", fmt.Sprintf("podSelector=%v matches no pods", np.Spec.PodSelector.MatchLabels), ""),
			},
			Explanation: "The policy's pod selector matches no pods in the namespace, so it neither allows nor restricts any traffic. The selector may be misconfigured.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Compare the pod selector with pod labels",
					Command: fmt.Sprintf("kubectl get pods -n %s --show-labels", np.Namespace),
				},
			},
		},
	}
}
