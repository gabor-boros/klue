package builtin

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/condition"
	"github.com/gabor-boros/klue/pkg/resource"
)

// FailedConditionRule detects standard status conditions reported in a failing
// state. It operates on dynamically fetched (unstructured) resources, leaving
// typed resources to their dedicated rules.
type FailedConditionRule struct{}

// ID returns the rule identifier.
func (FailedConditionRule) ID() string { return "builtin/failed-condition" }

// Description returns a human-readable description of the rule.
func (FailedConditionRule) Description() string {
	return "Detects failing status conditions on built-in resources"
}

// AppliesTo returns the kinds this rule evaluates.
func (FailedConditionRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.KindAny}
}

// Evaluate inspects status.conditions for a failing positive condition
// (Ready/Available=False) or a failing negative condition (Failed/Degraded=True).
func (r FailedConditionRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	obj, ok := graph.As[*unstructured.Unstructured](node)
	if !ok {
		return nil
	}

	var findings []diagnose.Finding
	for _, c := range condition.FromUnstructured(obj) {
		if !condition.IsFailing(c) {
			continue
		}

		findings = append(findings, diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Condition %s=%s", c.Type, c.Status),
			Severity:   diagnose.SeverityError,
			Confidence: 0.6,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, diagnose.EvidenceCondition, c.Message, c.Reason),
			},
			Explanation: fmt.Sprintf("The resource reports condition %q in a failing state (%s).", c.Type, c.Status),
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the resource status",
					Command: fmt.Sprintf("kubectl describe %s %s -n %s", describeKind(node.Ref.Kind), node.Ref.Name, node.Ref.Namespace),
				},
			},
		})
	}

	return findings
}
