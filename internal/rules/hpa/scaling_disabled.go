// Package hpa contains diagnostic rules for HorizontalPodAutoscalers.
package hpa

import (
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// ScalingDisabledRule flags HPAs that cannot scale, usually because metrics are
// unavailable or the scale target is missing.
type ScalingDisabledRule struct{}

// ID returns the rule identifier.
func (ScalingDisabledRule) ID() string { return "hpa/scaling-disabled" }

// Description returns a human-readable description of the rule.
func (ScalingDisabledRule) Description() string {
	return "Detects HorizontalPodAutoscalers unable to compute or apply scaling"
}

// AppliesTo returns the kinds this rule evaluates.
func (ScalingDisabledRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindHorizontalPodAutoscaler}
}

// Evaluate reports HPAs whose AbleToScale or ScalingActive condition is False.
func (r ScalingDisabledRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	hpa, ok := graph.As[*autoscalingv2.HorizontalPodAutoscaler](node)
	if !ok {
		return nil
	}

	var findings []diagnose.Finding
	for _, condition := range hpa.Status.Conditions {
		failing := (condition.Type == autoscalingv2.AbleToScale || condition.Type == autoscalingv2.ScalingActive) &&
			condition.Status == corev1.ConditionFalse
		if !failing {
			continue
		}

		findings = append(findings, diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("HPA cannot scale (%s=False)", condition.Type),
			Severity:   diagnose.SeverityError,
			Confidence: 0.8,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, diagnose.EvidenceCondition, condition.Message, condition.Reason),
			},
			Explanation: "The autoscaler cannot retrieve metrics or apply a scale decision, so the workload will not scale with load.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the HPA status and metrics server",
					Command: fmt.Sprintf("kubectl describe hpa %s -n %s", hpa.Name, hpa.Namespace),
				},
			},
		})
	}

	return findings
}
