package builtin

import (
	"fmt"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

// OrphanedOwnerRule detects owner references that point to a missing owner,
// which can leave the resource un-managed (for example a ReplicaSet whose
// Deployment was deleted without cascading, or a custom resource whose owning
// controller object no longer exists).
type OrphanedOwnerRule struct{}

// ID returns the rule identifier.
func (OrphanedOwnerRule) ID() string { return "builtin/orphaned-owner" }

// Description returns a human-readable description of the rule.
func (OrphanedOwnerRule) Description() string {
	return "Detects resources whose owner reference points to a missing object"
}

// AppliesTo returns the kinds this rule evaluates.
func (OrphanedOwnerRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.KindAny}
}

// Evaluate checks that every owner reference resolves to a node in the graph.
// Owners of any group are checked, including custom resources, since discovered
// CRDs are now part of the graph.
func (r OrphanedOwnerRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	meta, ok := graph.Meta(node)
	if !ok || ctx.Graph == nil {
		return nil
	}

	var findings []diagnose.Finding
	for _, ref := range kube.OwnerReferenceTargets(meta) {
		ownerKind := ref.Kind
		ownerName := ref.Name
		ownerUID := ref.UID
		if _, found := ctx.Graph.FindByRef(ref); found {
			continue
		}

		findings = append(findings, diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Owner %s/%s is missing", ownerKind, ownerName),
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.6,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "OwnerReference", fmt.Sprintf("owner %s %q (uid %s) was not found", ownerKind, ownerName, ownerUID), "OrphanedOwner"),
			},
			Explanation: fmt.Sprintf("The resource is owned by %s %q which no longer exists, so it may be unmanaged or left behind by a deletion.", ownerKind, ownerName),
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Verify whether the resource should still exist",
					Command: fmt.Sprintf("kubectl get %s %s -n %s -o yaml", describeKind(node.Ref.Kind), node.Ref.Name, node.Ref.Namespace),
				},
			},
		})
	}

	return findings
}
