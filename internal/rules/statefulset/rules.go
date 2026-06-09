// Package statefulset contains diagnostic rules for StatefulSets.
package statefulset

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// UnavailableRule flags StatefulSets with fewer ready replicas than desired.
type UnavailableRule struct{}

func (UnavailableRule) ID() string { return "statefulset/unavailable" }

func (UnavailableRule) Description() string {
	return "Detects StatefulSets with unavailable replicas"
}

func (UnavailableRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindStatefulSet}
}

func (r UnavailableRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	sts, ok := graph.As[*appsv1.StatefulSet](node)
	if !ok {
		return nil
	}

	desired := int32(1)
	if sts.Spec.Replicas != nil {
		desired = *sts.Spec.Replicas
	}
	if desired == 0 || sts.Status.ReadyReplicas >= desired {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      fmt.Sprintf("StatefulSet has %d/%d replicas ready", sts.Status.ReadyReplicas, desired),
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.7,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Status", fmt.Sprintf("ready=%d desired=%d current=%d updated=%d", sts.Status.ReadyReplicas, desired, sts.Status.CurrentReplicas, sts.Status.UpdatedReplicas), ""),
			},
			Explanation: "Some replicas are not ready. StatefulSet pods roll out sequentially, so a single stuck pod blocks the rest.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the StatefulSet pods in ordinal order",
					Command: fmt.Sprintf("kubectl get pods -n %s -l app=%s", sts.Namespace, sts.Name),
				},
			},
		},
	}
}

// RolloutStuckRule flags StatefulSets whose update revision has not rolled out
// to all replicas.
type RolloutStuckRule struct{}

func (RolloutStuckRule) ID() string { return "statefulset/rollout-stuck" }

func (RolloutStuckRule) Description() string {
	return "Detects StatefulSet rollouts that have not progressed"
}

func (RolloutStuckRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindStatefulSet}
}

func (r RolloutStuckRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	sts, ok := graph.As[*appsv1.StatefulSet](node)
	if !ok {
		return nil
	}

	desired := int32(1)
	if sts.Spec.Replicas != nil {
		desired = *sts.Spec.Replicas
	}
	if desired == 0 {
		return nil
	}

	// A rollout is in progress when the update and current revisions differ.
	if sts.Status.UpdateRevision == "" || sts.Status.UpdateRevision == sts.Status.CurrentRevision {
		return nil
	}
	if sts.Status.UpdatedReplicas >= desired {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "StatefulSet rollout has not completed",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.6,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Status", fmt.Sprintf("updated=%d desired=%d updateRevision=%s currentRevision=%s", sts.Status.UpdatedReplicas, desired, sts.Status.UpdateRevision, sts.Status.CurrentRevision), ""),
			},
			Explanation: "A newer revision is only partially rolled out. The next ordinal pod is likely failing to become ready.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the rollout status",
					Command: fmt.Sprintf("kubectl rollout status statefulset/%s -n %s", sts.Name, sts.Namespace),
				},
			},
		},
	}
}
