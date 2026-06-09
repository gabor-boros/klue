package builtin

import (
	"fmt"
	"time"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// TerminatingStuckRule detects objects that have carried a deletion timestamp
// for longer than the grace period, which usually indicates a finalizer that is
// not completing.
type TerminatingStuckRule struct{}

// ID returns the rule identifier.
func (TerminatingStuckRule) ID() string { return "builtin/terminating-stuck" }

// Description returns a human-readable description of the rule.
func (TerminatingStuckRule) Description() string {
	return "Detects resources stuck terminating past their grace period"
}

// AppliesTo returns the kinds this rule evaluates.
func (TerminatingStuckRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.KindAny}
}

// Evaluate reports objects whose deletion timestamp is older than the grace
// period. A zero reference time disables the rule to keep diagnoses
// deterministic.
func (r TerminatingStuckRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	meta, ok := graph.Meta(node)
	if !ok {
		return nil
	}

	deletion := meta.GetDeletionTimestamp()
	if deletion == nil || ctx.Options.Now.IsZero() {
		return nil
	}

	grace := ctx.Options.TerminatingGracePeriod
	if grace <= 0 {
		grace = defaultTerminatingGracePeriod
	}

	age := ctx.Options.Now.Sub(deletion.Time)
	if age < grace {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Resource is stuck terminating",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.7,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Metadata", fmt.Sprintf("deletionTimestamp set %s ago; finalizers=%v", age.Round(time.Second), meta.GetFinalizers()), "Terminating"),
			},
			Explanation: "The resource has a deletion timestamp but has not been removed, usually because a finalizer is not completing.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the pending finalizers",
					Command: fmt.Sprintf("kubectl get %s %s -n %s -o jsonpath='{.metadata.finalizers}'", describeKind(node.Ref.Kind), node.Ref.Name, node.Ref.Namespace),
				},
			},
		},
	}
}
