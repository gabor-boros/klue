package pod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
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
func (r ImagePullRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
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

		findings = append(findings, diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Container %q cannot pull image %q", status.Name, status.Image),
			Severity:   diagnose.SeverityError,
			Confidence: 0.9,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "ContainerStatus", waiting.Message, waiting.Reason),
			},
			Explanation: "The image name may be wrong or the registry may require credentials (imagePullSecrets).",
			Suggestions: []diagnose.Suggestion{
				{
					Title:       "Verify the image reference and pull secrets",
					Command:     fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
					Explanation: "Check that the image exists and that any required imagePullSecrets are present.",
				},
			},
		})
	}

	return findings
}
