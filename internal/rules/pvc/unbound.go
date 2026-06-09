// Package pvc contains diagnostic rules for PersistentVolumeClaims.
package pvc

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// UnboundRule flags PVCs that remain unbound.
type UnboundRule struct{}

// ID returns the rule identifier.
func (UnboundRule) ID() string { return "pvc/unbound" }

// Description returns a human-readable description of the rule.
func (UnboundRule) Description() string {
	return "Detects PersistentVolumeClaims stuck in Pending"
}

// AppliesTo returns the kinds this rule evaluates.
func (UnboundRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPersistentVolumeClaim}
}

// Evaluate inspects the PVC phase for an unbound claim.
func (r UnboundRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	pvc, ok := graph.As[*corev1.PersistentVolumeClaim](node)
	if !ok || pvc.Status.Phase == corev1.ClaimBound {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "PersistentVolumeClaim is not bound",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.8,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Status", fmt.Sprintf("phase=%s", pvc.Status.Phase), ""),
			},
			Explanation: "The claim has not been bound to a volume. Check the storage class and provisioner.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the PVC and its events",
					Command: fmt.Sprintf("kubectl describe pvc %s -n %s", pvc.Name, pvc.Namespace),
				},
			},
		},
	}
}
