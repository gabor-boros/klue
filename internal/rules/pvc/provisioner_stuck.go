package pvc

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

// ProvisionerStuckRule flags PVCs whose provisioner is failing.
type ProvisionerStuckRule struct{}

// ID returns the rule identifier.
func (ProvisionerStuckRule) ID() string { return "pvc/provisioner-stuck" }

// Description returns a human-readable description of the rule.
func (ProvisionerStuckRule) Description() string {
	return "Detects failed or stuck volume provisioning"
}

// AppliesTo returns the kinds this rule evaluates.
func (ProvisionerStuckRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPersistentVolumeClaim}
}

// Evaluate inspects PVC events for provisioning failures.
func (r ProvisionerStuckRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	pvc, ok := graph.As[*corev1.PersistentVolumeClaim](node)
	if !ok || pvc.Status.Phase == corev1.ClaimBound {
		return nil
	}

	event, failed := ruleutil.LatestWarningEvent(ctx, node.Ref, nil, "ProvisioningFailed")
	if !failed {
		return nil
	}

	cause := "external provisioner failure"
	if signal, ok := evidence.ParseWarningEventSignal(event); ok && signal.Category == "provisioning" {
		cause = provisioningCauseExplanation(signal.Cause)
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Volume provisioning is failing",
			Severity:   diagnose.SeverityError,
			Confidence: 0.8,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				ruleutil.NewEventEvidence(node.Ref, event),
			},
			Explanation: "The external provisioner reported a failure while creating the volume. Likely cause: " + cause + ".",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the PVC events and provisioner logs",
					Command: fmt.Sprintf("kubectl describe pvc %s -n %s", pvc.Name, pvc.Namespace),
				},
			},
		},
	}
}

func provisioningCauseExplanation(cause string) string {
	switch cause {
	case "quota-exceeded":
		return "quota exceeded in the storage backend or namespace"
	case "permission-denied":
		return "permission or IAM policy denied by the storage backend"
	case "topology-constraint":
		return "zone/topology constraints preventing provisioning"
	default:
		return "provisioning failure reported by the CSI provisioner"
	}
}
