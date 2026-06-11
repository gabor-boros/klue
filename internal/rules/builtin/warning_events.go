package builtin

import (
	"fmt"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// WarningEventsRule surfaces recent warning events recorded against any
// resource. Its confidence is intentionally low so that specific typed findings
// rank above it as the likely root cause.
type WarningEventsRule struct{}

// ID returns the rule identifier.
func (WarningEventsRule) ID() string { return "builtin/warning-events" }

// Description returns a human-readable description of the rule.
func (WarningEventsRule) Description() string {
	return "Surfaces recent warning events recorded against any resource"
}

// AppliesTo returns the kinds this rule evaluates.
func (WarningEventsRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.KindAny}
}

// Evaluate reports the most recent warning event within the configured window.
func (r WarningEventsRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	warnings := diagnose.WarningEvents(ctx, node.Ref)
	if len(warnings) == 0 {
		return nil
	}

	latest := warnings[len(warnings)-1]

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Warning event: %s", latest.Reason),
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.4,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, diagnose.EvidenceEvent, latest.Message, latest.Reason),
			},
			Explanation: "Kubernetes recorded a recent warning event for this resource that may explain its behaviour.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the resource events",
					Command: fmt.Sprintf("kubectl describe %s %s -n %s", describeKind(node.Ref.Kind), node.Ref.Name, node.Ref.Namespace),
				},
			},
		},
	}
}
