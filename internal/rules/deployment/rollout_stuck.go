// Package deployment contains diagnostic rules for Deployments.
package deployment

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// RolloutStuckRule flags deployments whose rollout has exceeded its progress
// deadline.
type RolloutStuckRule struct{}

// ID returns the rule identifier.
func (RolloutStuckRule) ID() string { return "deployment/rollout-stuck" }

// Description returns a human-readable description of the rule.
func (RolloutStuckRule) Description() string {
	return "Detects deployments whose rollout is stuck (ProgressDeadlineExceeded)"
}

// AppliesTo returns the kinds this rule evaluates.
func (RolloutStuckRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindDeployment}
}

// Evaluate inspects the Progressing condition for a stalled rollout.
func (r RolloutStuckRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	deploy, ok := graph.As[*appsv1.Deployment](node)
	if !ok {
		return nil
	}

	for _, condition := range deploy.Status.Conditions {
		if condition.Type != appsv1.DeploymentProgressing {
			continue
		}
		if condition.Reason != "ProgressDeadlineExceeded" {
			continue
		}

		return []diagnose.Finding{
			{
				ID:         r.ID(),
				Title:      "Deployment rollout is stuck",
				Severity:   diagnose.SeverityError,
				Confidence: 0.85,
				Resource:   node.Ref,
				Evidence: []diagnose.Evidence{
					diagnose.NewEvidence(node.Ref, "Condition", condition.Message, condition.Reason),
				},
				Explanation: "The deployment did not progress within its deadline. New pods may be failing to become ready.",
				Suggestions: []diagnose.Suggestion{
					{
						Title:   "Inspect the rollout status",
						Command: fmt.Sprintf("kubectl rollout status deployment/%s -n %s", deploy.Name, deploy.Namespace),
					},
				},
			},
		}
	}

	return nil
}
