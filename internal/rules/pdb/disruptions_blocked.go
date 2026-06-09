// Package pdb contains diagnostic rules for PodDisruptionBudgets.
package pdb

import (
	"fmt"

	policyv1 "k8s.io/api/policy/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// DisruptionsBlockedRule flags PodDisruptionBudgets that currently allow no
// voluntary disruptions, which blocks node drains and rollouts.
type DisruptionsBlockedRule struct{}

// ID returns the rule identifier.
func (DisruptionsBlockedRule) ID() string { return "pdb/disruptions-blocked" }

// Description returns a human-readable description of the rule.
func (DisruptionsBlockedRule) Description() string {
	return "Detects PodDisruptionBudgets that block voluntary disruptions"
}

// AppliesTo returns the kinds this rule evaluates.
func (DisruptionsBlockedRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPodDisruptionBudget}
}

// Evaluate reports budgets that the controller has observed but which currently
// allow zero disruptions.
func (r DisruptionsBlockedRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	budget, ok := graph.As[*policyv1.PodDisruptionBudget](node)
	if !ok {
		return nil
	}

	// Only meaningful once the controller has observed the budget.
	if budget.Status.ObservedGeneration == 0 && budget.Status.ExpectedPods == 0 {
		return nil
	}
	if budget.Status.DisruptionsAllowed > 0 {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "PodDisruptionBudget allows no disruptions",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.7,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, diagnose.EvidenceStatus, fmt.Sprintf("disruptionsAllowed=%d currentHealthy=%d desiredHealthy=%d expected=%d", budget.Status.DisruptionsAllowed, budget.Status.CurrentHealthy, budget.Status.DesiredHealthy, budget.Status.ExpectedPods), ""),
			},
			Explanation: "No voluntary disruptions are currently allowed, so node drains and rolling updates of the covered pods will block until more replicas become healthy.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the budget and the health of covered pods",
					Command: fmt.Sprintf("kubectl describe pdb %s -n %s", budget.Name, budget.Namespace),
				},
			},
		},
	}
}
