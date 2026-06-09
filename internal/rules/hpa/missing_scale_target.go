package hpa

import (
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

// MissingScaleTargetRule flags HPAs whose scale target does not exist in the
// graph.
type MissingScaleTargetRule struct{}

// ID returns the rule identifier.
func (MissingScaleTargetRule) ID() string { return "hpa/missing-scale-target" }

// Description returns a human-readable description of the rule.
func (MissingScaleTargetRule) Description() string {
	return "Detects HorizontalPodAutoscalers referencing a missing scale target"
}

// AppliesTo returns the kinds this rule evaluates.
func (MissingScaleTargetRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindHorizontalPodAutoscaler}
}

// Evaluate checks that the HPA scale target exists in the graph.
func (r MissingScaleTargetRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	hpa, ok := graph.As[*autoscalingv2.HorizontalPodAutoscaler](node)
	if !ok {
		return nil
	}

	keepScaleTarget := func(rel kube.Relationship) bool {
		return rel.EdgeKind == graph.EdgeScaleTarget
	}

	return ruleutil.MissingRelationships(ctx, kube.TypedRelationships(hpa), keepScaleTarget, func(rel kube.Relationship) diagnose.Finding {
		return diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("HPA scale target %s/%s does not exist", rel.Target.Kind, rel.Target.Name),
			Severity:   diagnose.SeverityError,
			Confidence: 0.8,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "ScaleTargetRef", fmt.Sprintf("%s %q is referenced but missing (%s)", rel.Target.Kind, rel.Target.Name, rel.Path), ""),
			},
			Explanation: "The autoscaler points at a workload that does not exist, so it has nothing to scale.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Verify the scale target name and kind",
					Command: fmt.Sprintf("kubectl get %s %s -n %s", ruleutil.KubectlKind(rel.Target.Kind), rel.Target.Name, hpa.Namespace),
				},
			},
		}
	})
}
