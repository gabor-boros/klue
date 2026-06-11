package pod

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
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

	notReadyStatuses := make([]corev1.ContainerStatus, 0, len(pod.Status.ContainerStatuses))
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Running != nil && !status.Ready {
			notReadyStatuses = append(notReadyStatuses, status)
		}
	}
	if len(notReadyStatuses) == 0 {
		return nil
	}

	event, unhealthy := ruleutil.LatestWarningEvent(ctx, node.Ref, func(event corev1.Event) bool {
		for _, status := range notReadyStatuses {
			if ruleutil.MatchProbeEvent(event, status.Name) {
				return true
			}
		}
		return false
	}, "Unhealthy")
	if !unhealthy {
		return nil
	}

	evidenceItems := []diagnose.Evidence{
		ruleutil.NewEventEvidence(node.Ref, event),
	}
	for _, status := range notReadyStatuses {
		evidenceItems = append(evidenceItems, ruleutil.LogEvidence(ctx, node.Ref, status.Name, false)...)
	}

	explanation := "A running container is reporting not-ready due to probe failures."
	if strings.Contains(strings.ToLower(event.Message), "readiness") {
		explanation = "A running container is reporting not-ready due to failing readiness probes."
	} else if strings.Contains(strings.ToLower(event.Message), "liveness") {
		explanation = "A running container is repeatedly failing liveness probes."
	}
	for _, status := range notReadyStatuses {
		if logExplanation := ruleutil.LogExplanation(ctx, node.Ref, status.Name); logExplanation != "" {
			explanation += logExplanation
			break
		}
	}

	return []diagnose.Finding{
		{
			ID:          r.ID(),
			Title:       "Pod is failing health probes",
			Severity:    diagnose.SeverityWarning,
			Confidence:  0.7,
			Resource:    node.Ref,
			Evidence:    evidenceItems,
			Explanation: explanation,
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Review probe configuration and application health",
					Command: fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
				},
			},
		},
	}
}
