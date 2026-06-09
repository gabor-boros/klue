// Package lease contains diagnostic rules for coordination Leases.
package lease

import (
	"fmt"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// defaultLeaseDuration is used when a Lease does not declare its duration.
const defaultLeaseDuration = 15 * time.Second

// staleRenewMultiplier is how many lease durations may elapse past the last
// renewal before the holder is considered stale.
const staleRenewMultiplier = 4

// StaleRule flags Leases whose holder has not renewed within several lease
// durations, which typically means the leader-election holder is unhealthy.
type StaleRule struct{}

func (StaleRule) ID() string { return "lease/stale" }

func (StaleRule) Description() string {
	return "Detects Leases whose holder has stopped renewing"
}

func (StaleRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindLease}
}

func (r StaleRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	l, ok := graph.As[*coordinationv1.Lease](node)
	if !ok || ctx.Options.Now.IsZero() {
		return nil
	}
	if l.Spec.RenewTime == nil {
		return nil
	}

	duration := defaultLeaseDuration
	if l.Spec.LeaseDurationSeconds != nil && *l.Spec.LeaseDurationSeconds > 0 {
		duration = time.Duration(*l.Spec.LeaseDurationSeconds) * time.Second
	}

	multiplier := ctx.Options.LeaseStaleMultiplier
	if multiplier <= 0 {
		multiplier = staleRenewMultiplier
	}

	threshold := duration * time.Duration(multiplier)
	age := ctx.Options.Now.Sub(l.Spec.RenewTime.Time)
	if age <= threshold {
		return nil
	}

	holder := "<none>"
	if l.Spec.HolderIdentity != nil {
		holder = *l.Spec.HolderIdentity
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Lease has not been renewed recently",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.6,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Spec", fmt.Sprintf("holder=%s lastRenew=%s ago (threshold %s)", holder, age.Round(time.Second), threshold), "StaleLease"),
			},
			Explanation: "The lease holder has not renewed within several lease durations. The leader-electing component may be down or partitioned.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Check the health of the lease holder component",
					Command: fmt.Sprintf("kubectl describe lease %s -n %s", l.Name, l.Namespace),
				},
			},
		},
	}
}
