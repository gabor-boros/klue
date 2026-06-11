package pod

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

// ImagePullRule flags containers that cannot pull their image.
type ImagePullRule struct{}

// ID returns the rule identifier.
func (ImagePullRule) ID() string { return "pod/image-pull" }

// Description returns a human-readable description of the rule.
func (ImagePullRule) Description() string {
	return "Detects containers that fail to pull their image (ImagePullBackOff/ErrImagePull)"
}

// AppliesTo returns the kinds this rule evaluates.
func (ImagePullRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPod}
}

// Evaluate inspects container statuses for image pull failures.
func (r ImagePullRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	pod, ok := graph.As[*corev1.Pod](node)
	if !ok {
		return nil
	}

	var findings []diagnose.Finding
	for _, status := range pod.Status.ContainerStatuses {
		waiting := status.State.Waiting
		if waiting == nil {
			continue
		}
		if waiting.Reason != "ImagePullBackOff" && waiting.Reason != "ErrImagePull" {
			continue
		}

		matcher := imagePullEventMatcher(status.Name, status.Image)
		matchedEvent, hasMatchedEvent := ruleutil.LatestWarningEvent(
			ctx,
			node.Ref,
			matcher,
			"Failed",
			"BackOff",
			"FailedToRetrieveImagePullSecret",
		)
		eventEvidence := ruleutil.EventEvidenceMatching(
			ctx,
			node.Ref,
			matcher,
			"Failed",
			"BackOff",
			"FailedToRetrieveImagePullSecret",
		)

		confidence := diagnose.Confidence(0.9)
		if len(eventEvidence) > 0 {
			confidence = 0.95
		}

		signal := evidence.EventSignal{}
		if hasMatchedEvent {
			if parsed, ok := evidence.ParseWarningEventSignal(matchedEvent); ok {
				signal = parsed
			}
		}

		explanation := imagePullExplanation(waiting.Message, signal, len(eventEvidence) > 0)
		suggestions := imagePullSuggestions(waiting.Message, signal, pod.Name, pod.Namespace)

		evidenceItems := []diagnose.Evidence{
			diagnose.NewEvidence(node.Ref, diagnose.EvidenceStatus, waiting.Message, waiting.Reason),
		}
		evidenceItems = append(evidenceItems, eventEvidence...)

		findings = append(findings, diagnose.Finding{
			ID:          r.ID(),
			Title:       fmt.Sprintf("Container %q cannot pull image %q", status.Name, status.Image),
			Severity:    diagnose.SeverityError,
			Confidence:  confidence,
			Resource:    node.Ref,
			Evidence:    evidenceItems,
			Explanation: explanation,
			Suggestions: suggestions,
		})
	}

	return findings
}

func imagePullEventMatcher(container, image string) func(corev1.Event) bool {
	return func(event corev1.Event) bool {
		return ruleutil.MatchImagePullEvent(event, container, image)
	}
}

func imagePullExplanation(waitingMessage string, signal evidence.EventSignal, correlated bool) string {
	if signal.Category == "image-pull" {
		switch signal.Cause {
		case "registry-auth":
			return "Image pull events indicate registry authentication or pull-secret issues. Verify imagePullSecrets and ServiceAccount configuration."
		case "image-not-found":
			return "Image pull events indicate the requested image or tag cannot be found in the registry."
		case "registry-tls":
			return "Image pull events indicate TLS or certificate validation failures against the registry endpoint."
		case "registry-network":
			return "Image pull events indicate registry reachability issues (DNS, TCP connectivity, or timeout)."
		}
	}

	lowerMessage := strings.ToLower(waitingMessage)
	switch {
	case strings.Contains(lowerMessage, "secret"):
		if correlated {
			return "Image pull events indicate registry credential issues. Verify imagePullSecrets and ServiceAccount configuration."
		}
		return "The registry credentials may be missing or invalid. Verify imagePullSecrets and ServiceAccount configuration."
	case strings.Contains(lowerMessage, "not found"), strings.Contains(lowerMessage, "manifest unknown"):
		if correlated {
			return "Image pull events indicate the requested image or tag cannot be found in the registry."
		}
		return "The image name or tag may be wrong. Verify the image reference exists in the registry."
	default:
		if correlated {
			return "Image pull warning events corroborate a registry pull failure. Check image reference, credentials, and registry reachability."
		}
		return "The image name may be wrong or the registry may require credentials (imagePullSecrets)."
	}
}

func imagePullSuggestions(waitingMessage string, signal evidence.EventSignal, podName, namespace string) []diagnose.Suggestion {
	describeSuggestion := diagnose.Suggestion{
		Title:       "Verify the image reference and pull secrets",
		Command:     fmt.Sprintf("kubectl describe pod %s -n %s", podName, namespace),
		Explanation: "Check pull events, image reference, and whether required imagePullSecrets are attached.",
	}

	lowerMessage := strings.ToLower(waitingMessage)
	if signal.Category == "image-pull" {
		switch signal.Cause {
		case "registry-network":
			return []diagnose.Suggestion{
				describeSuggestion,
				{
					Title:       "Verify registry network reachability",
					Command:     fmt.Sprintf("kubectl run netcheck -n %s --rm -it --image=busybox --restart=Never -- nslookup ghcr.io", namespace),
					Explanation: "Validate DNS and network access from the cluster to your image registry endpoint.",
				},
			}
		case "registry-tls":
			return []diagnose.Suggestion{
				describeSuggestion,
				{
					Title:       "Verify registry TLS trust and certificates",
					Command:     fmt.Sprintf("kubectl describe pod %s -n %s", podName, namespace),
					Explanation: "Check certificate trust configuration for private registries and cluster node pull settings.",
				},
			}
		}
	}

	switch {
	case strings.Contains(lowerMessage, "secret"):
		return []diagnose.Suggestion{
			describeSuggestion,
			{
				Title:       "Inspect imagePullSecrets and ServiceAccount",
				Command:     fmt.Sprintf("kubectl get sa default -n %s -o yaml", namespace),
				Explanation: "Confirm the ServiceAccount or Pod references valid imagePullSecrets for the registry.",
			},
		}
	case strings.Contains(lowerMessage, "not found"), strings.Contains(lowerMessage, "manifest unknown"):
		return []diagnose.Suggestion{
			describeSuggestion,
			{
				Title:       "Validate the image repository and tag",
				Command:     "kubectl get pod " + podName + " -n " + namespace + " -o jsonpath='{.spec.containers[*].image}'",
				Explanation: "Compare the referenced image tags with those published in your registry.",
			},
		}
	default:
		return []diagnose.Suggestion{describeSuggestion}
	}
}
