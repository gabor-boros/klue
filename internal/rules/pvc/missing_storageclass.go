package pvc

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

// MissingStorageClassRule flags PVCs that reference a non-existent storage class.
type MissingStorageClassRule struct{}

// ID returns the rule identifier.
func (MissingStorageClassRule) ID() string { return "pvc/missing-storageclass" }

// Description returns a human-readable description of the rule.
func (MissingStorageClassRule) Description() string {
	return "Detects PVCs referencing a missing StorageClass"
}

// AppliesTo returns the kinds this rule evaluates.
func (MissingStorageClassRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPersistentVolumeClaim}
}

// Evaluate checks that the referenced storage class exists in the graph.
func (r MissingStorageClassRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	pvc, ok := graph.As[*corev1.PersistentVolumeClaim](node)
	if !ok {
		return nil
	}

	keepStorageClass := func(rel kube.Relationship) bool {
		return rel.Target.Kind == resource.ReferenceKindStorageClass
	}

	return ruleutil.MissingRelationships(ctx, kube.TypedRelationships(pvc), keepStorageClass, func(rel kube.Relationship) diagnose.Finding {
		className := rel.Target.Name
		return diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("StorageClass %q does not exist", className),
			Severity:   diagnose.SeverityError,
			Confidence: 0.85,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Reference", fmt.Sprintf("storageClassName=%s is missing (%s)", className, rel.Path), ""),
			},
			Explanation: "The PVC references a storage class that is not defined, so it can never be provisioned.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "List available storage classes",
					Command: "kubectl get storageclass",
				},
			},
		}
	})
}
