package rbac

import (
	"fmt"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// NoSubjectsRule flags (Cluster)RoleBindings that have no subjects, so they
// grant permissions to nobody.
type NoSubjectsRule struct{}

// ID returns the rule identifier.
func (NoSubjectsRule) ID() string { return "rbac/no-subjects" }

// Description returns a human-readable description of the rule.
func (NoSubjectsRule) Description() string {
	return "Detects RoleBindings with no subjects"
}

// AppliesTo returns the kinds this rule evaluates.
func (NoSubjectsRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindRoleBinding, resource.ReferenceKindClusterRoleBinding}
}

// Evaluate reports bindings that resolve a roleRef but list no subjects.
func (r NoSubjectsRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	_, name, ok := bindingRoleRef(node.Object)
	if !ok {
		return nil
	}

	if bindingHasSubjects(node.Object) {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Binding has no subjects",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.6,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Subjects", "the binding lists no subjects", "NoSubjects"),
			},
			Explanation: fmt.Sprintf("The binding %q has an empty subjects list, so it grants its role to no users, groups, or service accounts.", name),
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Add the intended subjects or remove the binding",
					Command: fmt.Sprintf("kubectl describe %s", name),
				},
			},
		},
	}
}
