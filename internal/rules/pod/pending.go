package pod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// PendingRule flags pods that remain unschedulable.
type PendingRule struct{}

// ID returns the rule identifier.
func (PendingRule) ID() string { return "pod/pending" }

// Description returns a human-readable description of the rule.
func (PendingRule) Description() string {
	return "Detects pods that cannot be scheduled"
}

// AppliesTo returns the kinds this rule evaluates.
func (PendingRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPod}
}

// Evaluate inspects pending pods for scheduling failures.
func (r PendingRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	pod, ok := graph.As[*corev1.Pod](node)
	if !ok || pod.Status.Phase != corev1.PodPending {
		return nil
	}

	event, scheduled := diagnose.HasEventReason(ctx, node.Ref, "FailedScheduling")
	if !scheduled {
		// Pending without a scheduling failure is often a transient state
		// (e.g. ContainerCreating); leave that to the more specific rules.
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Pod cannot be scheduled",
			Severity:   diagnose.SeverityError,
			Confidence: 0.8,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Event", event.Message, event.Reason),
			},
			Explanation: "The scheduler could not place the pod on any node. This is usually caused by insufficient resources, node selectors/affinity, or taints.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Review scheduling constraints and cluster capacity",
					Command: fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
				},
			},
		},
	}
}
