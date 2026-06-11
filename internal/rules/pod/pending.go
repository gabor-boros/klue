package pod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
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

	event, scheduled := ruleutil.LatestWarningEvent(ctx, node.Ref, nil, "FailedScheduling")
	if !scheduled {
		// Pending without a scheduling failure is often a transient state
		// (e.g. ContainerCreating); leave that to the more specific rules.
		return nil
	}

	cause := "insufficient resources, node selectors/affinity, or taints"
	if signal, ok := evidence.ParseWarningEventSignal(event); ok && signal.Category == "scheduling" {
		cause = schedulingCauseExplanation(signal.Cause)
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Pod cannot be scheduled",
			Severity:   diagnose.SeverityError,
			Confidence: 0.8,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, diagnose.EvidenceEvent, event.Message, event.Reason),
			},
			Explanation: fmt.Sprintf("The scheduler could not place the pod on any node. The latest warning event suggests %s.", cause),
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Review scheduling constraints and cluster capacity",
					Command: fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
				},
			},
		},
	}
}

func schedulingCauseExplanation(cause string) string {
	switch cause {
	case "insufficient-cpu":
		return "insufficient CPU on schedulable nodes"
	case "insufficient-memory":
		return "insufficient memory on schedulable nodes"
	case "selector-or-affinity":
		return "node selector or affinity constraints that do not match available nodes"
	case "taints-or-tolerations":
		return "taints/tolerations mismatch"
	case "pvc-binding":
		return "unbound PVC constraints blocking scheduling"
	case "topology-spread":
		return "topology spread constraints that cannot currently be satisfied"
	default:
		return "a scheduling constraint or resource availability issue"
	}
}
