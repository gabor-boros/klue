package rbac

import (
	"fmt"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

// MissingRoleRule flags (Cluster)RoleBindings whose roleRef points to a role
// that does not exist in the graph.
type MissingRoleRule struct{}

// ID returns the rule identifier.
func (MissingRoleRule) ID() string { return "rbac/missing-role" }

// Description returns a human-readable description of the rule.
func (MissingRoleRule) Description() string {
	return "Detects RoleBindings referencing a missing Role or ClusterRole"
}

// AppliesTo returns the kinds this rule evaluates.
func (MissingRoleRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindRoleBinding, resource.ReferenceKindClusterRoleBinding}
}

// Evaluate checks that the role referenced by the binding exists in the graph.
func (r MissingRoleRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	roleRef, name, ok := bindingRoleRef(node.Object)
	if !ok {
		return nil
	}

	keepRoleRef := func(rel kube.Relationship) bool {
		return rel.EdgeKind == graph.EdgeRoleRef
	}

	return ruleutil.MissingRelationships(ctx, kube.TypedRelationships(node.Object), keepRoleRef, func(rel kube.Relationship) diagnose.Finding {
		return diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Binding references missing %s %q", roleRef.Kind, roleRef.Name),
			Severity:   diagnose.SeverityError,
			Confidence: 0.8,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "RoleRef", fmt.Sprintf("%s %q is referenced but missing (%s)", roleRef.Kind, roleRef.Name, rel.Path), ""),
			},
			Explanation: fmt.Sprintf("The binding %q grants a role that does not exist, so it confers no permissions.", name),
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Verify the referenced role exists",
					Command: fmt.Sprintf("kubectl get %s %s", ruleutil.KubectlKind(resource.Kind(roleRef.Kind)), roleRef.Name),
				},
			},
		}
	})
}
