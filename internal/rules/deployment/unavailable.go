package deployment

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// UnavailableRule flags deployments with fewer available replicas than desired.
type UnavailableRule struct{}

// ID returns the rule identifier.
func (UnavailableRule) ID() string { return "deployment/unavailable" }

// Description returns a human-readable description of the rule.
func (UnavailableRule) Description() string {
	return "Detects deployments with unavailable replicas"
}

// AppliesTo returns the kinds this rule evaluates.
func (UnavailableRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindDeployment}
}

// Evaluate compares available replicas against the desired count.
func (r UnavailableRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	deploy, ok := graph.As[*appsv1.Deployment](node)
	if !ok {
		return nil
	}

	desired := int32(1)
	if deploy.Spec.Replicas != nil {
		desired = *deploy.Spec.Replicas
	}

	if desired == 0 || deploy.Status.AvailableReplicas >= desired {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Deployment has %d/%d replicas available", deploy.Status.AvailableReplicas, desired),
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.7,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Status", fmt.Sprintf("available=%d desired=%d ready=%d", deploy.Status.AvailableReplicas, desired, deploy.Status.ReadyReplicas), ""),
			},
			Explanation: "Some replicas are not available. Inspect the owned pods to find the underlying cause.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the deployment's pods",
					Command: fmt.Sprintf("kubectl get pods -n %s -l app=%s", deploy.Namespace, deploy.Name),
				},
			},
		},
	}
}
