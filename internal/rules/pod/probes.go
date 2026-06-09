package pod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// ProbeRule flags pods whose containers are running but failing probes.
type ProbeRule struct{}

// ID returns the rule identifier.
func (ProbeRule) ID() string { return "pod/probe-failure" }

// Description returns a human-readable description of the rule.
func (ProbeRule) Description() string {
	return "Detects readiness/liveness probe failures"
}

// AppliesTo returns the kinds this rule evaluates.
func (ProbeRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPod}
}

// Evaluate inspects running-but-not-ready containers and Unhealthy events.
func (r ProbeRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	pod, ok := graph.As[*corev1.Pod](node)
	if !ok || pod.Status.Phase != corev1.PodRunning {
		return nil
	}

	event, unhealthy := diagnose.HasEventReason(ctx, node.Ref, "Unhealthy")
	if !unhealthy {
		return nil
	}

	notReady := false
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Running != nil && !status.Ready {
			notReady = true
			break
		}
	}
	if !notReady {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Pod is failing health probes",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.7,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Event", event.Message, event.Reason),
			},
			Explanation: "A container is running but reporting not-ready due to failing readiness/liveness probes.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Review probe configuration and application health",
					Command: fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
				},
			},
		},
	}
}
