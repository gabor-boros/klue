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

// MountFailureRule flags pods whose warning events indicate mount/volume issues.
type MountFailureRule struct{}

// ID returns the rule identifier.
func (MountFailureRule) ID() string { return "pod/mount-failure" }

// Description returns a human-readable description of the rule.
func (MountFailureRule) Description() string {
	return "Detects mount and volume attachment failures from warning events"
}

// AppliesTo returns the kinds this rule evaluates.
func (MountFailureRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPod}
}

// Evaluate inspects warning events for mount failures on pods.
func (r MountFailureRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	pod, ok := graph.As[*corev1.Pod](node)
	if !ok {
		return nil
	}

	event, found := ruleutil.LatestWarningEvent(ctx, node.Ref, func(event corev1.Event) bool {
		signal, ok := evidence.ParseWarningEventSignal(event)
		return ok && signal.Category == "mount"
	}, "FailedMount", "FailedAttachVolume", "FailedMapVolume", "Failed")
	if !found {
		return nil
	}

	signal, _ := evidence.ParseWarningEventSignal(event)
	title := "Pod volume mount or attachment is failing"
	if signal.Volume != "" {
		title = fmt.Sprintf("Pod volume %q mount or attachment is failing", signal.Volume)
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      title,
			Severity:   diagnose.SeverityError,
			Confidence: 0.85,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				ruleutil.NewEventEvidence(node.Ref, event),
			},
			Explanation: "Kubernetes warning events indicate volume mount/attachment failures for this pod. Cause: " + mountCauseExplanation(signal.Cause) + ".",
			Suggestions: []diagnose.Suggestion{
				{
					Title:       "Inspect pod volumes and related objects",
					Command:     fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
					Explanation: "Review volume events and verify referenced Secrets, ConfigMaps, PVCs, and CSI driver status.",
				},
			},
		},
	}
}

func mountCauseExplanation(cause string) string {
	switch cause {
	case "missing-secret":
		return "missing Secret referenced by a volume"
	case "missing-configmap":
		return "missing ConfigMap referenced by a volume"
	case "missing-pvc":
		return "missing or unresolved PersistentVolumeClaim"
	case "mount-timeout":
		return "mount operation timed out"
	default:
		return "volume mount or attachment failure"
	}
}
