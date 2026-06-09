// Package daemonset contains diagnostic rules for DaemonSets.
package daemonset

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// UnavailableRule flags DaemonSets that are not scheduled or ready on all
// eligible nodes.
type UnavailableRule struct{}

func (UnavailableRule) ID() string { return "daemonset/unavailable" }

func (UnavailableRule) Description() string {
	return "Detects DaemonSets not available on all eligible nodes"
}

func (UnavailableRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindDaemonSet}
}

func (r UnavailableRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	ds, ok := graph.As[*appsv1.DaemonSet](node)
	if !ok {
		return nil
	}

	desired := ds.Status.DesiredNumberScheduled
	if desired == 0 {
		return nil
	}
	if ds.Status.NumberUnavailable == 0 && ds.Status.NumberReady >= desired {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      fmt.Sprintf("DaemonSet has %d/%d pods ready", ds.Status.NumberReady, desired),
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.7,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Status", fmt.Sprintf("desired=%d ready=%d available=%d unavailable=%d", desired, ds.Status.NumberReady, ds.Status.NumberAvailable, ds.Status.NumberUnavailable), ""),
			},
			Explanation: "The DaemonSet is not running healthily on every eligible node. Affected nodes may lack resources or be failing to pull the image.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the DaemonSet pods across nodes",
					Command: fmt.Sprintf("kubectl get pods -n %s -l app=%s -o wide", ds.Namespace, ds.Name),
				},
			},
		},
	}
}

// MisscheduledRule flags DaemonSets with pods running on nodes that should not
// run them, which indicates stale scheduling after a selector or taint change.
type MisscheduledRule struct{}

func (MisscheduledRule) ID() string { return "daemonset/misscheduled" }

func (MisscheduledRule) Description() string {
	return "Detects DaemonSet pods scheduled on ineligible nodes"
}

func (MisscheduledRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindDaemonSet}
}

func (r MisscheduledRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	ds, ok := graph.As[*appsv1.DaemonSet](node)
	if !ok || ds.Status.NumberMisscheduled == 0 {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      fmt.Sprintf("DaemonSet has %d misscheduled pods", ds.Status.NumberMisscheduled),
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.6,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Status", fmt.Sprintf("misscheduled=%d", ds.Status.NumberMisscheduled), ""),
			},
			Explanation: "Some DaemonSet pods are running on nodes that no longer match the node selector or tolerate the node's taints.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Review the DaemonSet node selector and tolerations",
					Command: fmt.Sprintf("kubectl describe daemonset %s -n %s", ds.Name, ds.Namespace),
				},
			},
		},
	}
}
