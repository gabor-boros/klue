// Package pod contains diagnostic rules for Pods.
package pod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// CrashLoopRule flags containers stuck in CrashLoopBackOff.
type CrashLoopRule struct{}

// ID returns the rule identifier.
func (CrashLoopRule) ID() string { return "pod/crashloop" }

// Description returns a human-readable description of the rule.
func (CrashLoopRule) Description() string {
	return "Detects containers repeatedly crashing (CrashLoopBackOff)"
}

// AppliesTo returns the kinds this rule evaluates.
func (CrashLoopRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPod}
}

// Evaluate inspects container statuses for CrashLoopBackOff.
func (r CrashLoopRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	pod, ok := graph.As[*corev1.Pod](node)
	if !ok {
		return nil
	}

	var findings []diagnose.Finding
	for _, status := range pod.Status.ContainerStatuses {
		waiting := status.State.Waiting
		if waiting == nil || waiting.Reason != "CrashLoopBackOff" {
			continue
		}

		findings = append(findings, diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Container %q is in CrashLoopBackOff", status.Name),
			Severity:   diagnose.SeverityCritical,
			Confidence: 0.95,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "ContainerStatus", waiting.Message, fmt.Sprintf("restartCount=%d", status.RestartCount)),
			},
			Explanation: fmt.Sprintf("Container %q has restarted %d times and is backing off.", status.Name, status.RestartCount),
			Suggestions: []diagnose.Suggestion{
				{
					Title:       "Inspect the previous container logs",
					Command:     fmt.Sprintf("kubectl logs %s -c %s --previous -n %s", pod.Name, status.Name, pod.Namespace),
					Explanation: "The crash reason is usually visible in the logs of the previous run.",
				},
			},
		})
	}

	return findings
}
