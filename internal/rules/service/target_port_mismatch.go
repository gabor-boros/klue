package service

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// TargetPortMismatchRule flags service ports whose target port is not exposed
// by any selected pod.
type TargetPortMismatchRule struct{}

// ID returns the rule identifier.
func (TargetPortMismatchRule) ID() string { return "service/target-port-mismatch" }

// Description returns a human-readable description of the rule.
func (TargetPortMismatchRule) Description() string {
	return "Detects service target ports not exposed by selected pods"
}

// AppliesTo returns the kinds this rule evaluates.
func (TargetPortMismatchRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindService}
}

// Evaluate verifies that each service port maps to a container port on the
// selected pods.
func (r TargetPortMismatchRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	svc, ok := graph.As[*corev1.Service](node)
	if !ok || len(svc.Spec.Selector) == 0 {
		return nil
	}

	pods := selectedPods(ctx.Graph, node)
	if len(pods) == 0 {
		// Without backing pods this is covered by selector-mismatch/no-endpoints.
		return nil
	}

	var findings []diagnose.Finding
	for _, port := range svc.Spec.Ports {
		if targetPortMatches(port.TargetPort, pods) {
			continue
		}

		findings = append(findings, diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Service port %d targets %q which no pod exposes", port.Port, port.TargetPort.String()),
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.65,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Port", fmt.Sprintf("targetPort=%s", port.TargetPort.String()), ""),
			},
			Explanation: "The service target port does not match any container port on the selected pods, so connections will be refused.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Align the service targetPort with the container port",
					Command: fmt.Sprintf("kubectl describe service %s -n %s", svc.Name, svc.Namespace),
				},
			},
		})
	}

	return findings
}

// targetPortMatches reports whether any selected pod exposes the target port,
// matching by number or by named port.
func targetPortMatches(target intstr.IntOrString, pods []graph.Node) bool {
	for i := range pods {
		pod, ok := pods[i].Object.(*corev1.Pod)
		if !ok {
			continue
		}
		for _, container := range pod.Spec.Containers {
			for _, cp := range container.Ports {
				switch target.Type {
				case intstr.Int:
					if cp.ContainerPort == target.IntVal {
						return true
					}
				case intstr.String:
					if cp.Name == target.StrVal {
						return true
					}
				}
			}
		}
	}

	return false
}
