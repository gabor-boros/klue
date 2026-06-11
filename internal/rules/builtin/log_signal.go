package builtin

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// LogSignalRule surfaces failure patterns detected in container logs when pod
// status alone does not already explain the failure.
type LogSignalRule struct{}

func (LogSignalRule) ID() string { return "builtin/log-signal" }

func (LogSignalRule) Description() string {
	return "Detects failure patterns in container logs"
}

func (LogSignalRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPod}
}

func (r LogSignalRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	pod, ok := graph.As[*corev1.Pod](node)
	if !ok || ctx.Logs == nil {
		return nil
	}

	if hasStrongStatusSignal(pod) {
		return nil
	}

	var findings []diagnose.Finding
	for _, status := range pod.Status.ContainerStatuses {
		logs := ctx.Logs.ForPodContainer(node.Ref, status.Name)
		signal, line, matched := logs.BestSignal()
		if !matched || signal.ID == "error-keyword" {
			continue
		}

		confidence := 0.7 + signal.ConfidenceBoost
		if confidence > 0.95 {
			confidence = 0.95
		}

		evidence := []diagnose.Evidence{
			diagnose.NewEvidence(node.Ref, diagnose.EvidenceLog, logs.SummaryMessage(status.Name, false), logs.RawExcerpt(3)),
		}

		findings = append(findings, diagnose.Finding{
			ID:          r.ID(),
			Title:       fmt.Sprintf("Container %q logs show %s", status.Name, signal.Summary),
			Severity:    diagnose.SeverityWarning,
			Confidence:  diagnose.Confidence(confidence),
			Resource:    node.Ref,
			Evidence:    evidence,
			Explanation: fmt.Sprintf("Container %q logs contain %q, suggesting %s.", status.Name, truncateLine(line, 80), signal.Summary),
			Suggestions: logSignalSuggestions(signal, pod.Namespace, pod.Name),
		})
	}

	return findings
}

func hasStrongStatusSignal(pod *corev1.Pod) bool {
	for _, status := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
		if waiting := status.State.Waiting; waiting != nil {
			switch waiting.Reason {
			case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "CreateContainerConfigError":
				return true
			}
		}
		if terminated := status.State.Terminated; terminated != nil {
			switch terminated.Reason {
			case "Error", "OOMKilled":
				return true
			}
		}
	}
	return pod.Status.Phase == corev1.PodFailed
}

func logSignalSuggestions(signal evidence.LogSignal, namespace, podName string) []diagnose.Suggestion {
	switch signal.ID {
	case "connection-refused", "no-such-host":
		return []diagnose.Suggestion{
			{
				Title:       "Check upstream Services and Endpoints",
				Command:     "kubectl get svc,endpointslices -n " + namespace,
				Explanation: "Connection and DNS failures often point to a missing Service, selector mismatch, or unreachable backend.",
			},
		}
	case "config-missing":
		return []diagnose.Suggestion{
			{
				Title:       "Verify mounted ConfigMaps and Secrets",
				Command:     fmt.Sprintf("kubectl describe pod %s -n %s", podName, namespace),
				Explanation: "Missing files in logs often mean a volume mount or projected volume is misconfigured.",
			},
		}
	case "permission-denied":
		return []diagnose.Suggestion{
			{
				Title:       "Review RBAC and credentials",
				Command:     "kubectl auth can-i --list",
				Explanation: "Authorization failures in logs may indicate missing RoleBindings or invalid tokens.",
			},
		}
	default:
		return nil
	}
}

func truncateLine(line string, max int) string {
	if len(line) <= max {
		return line
	}
	return line[:max-3] + "..."
}
