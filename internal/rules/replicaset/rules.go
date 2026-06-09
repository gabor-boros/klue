// Package replicaset contains diagnostic rules for ReplicaSets.
package replicaset

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// UnavailableRule flags ReplicaSets with fewer ready replicas than desired.
type UnavailableRule struct{}

func (UnavailableRule) ID() string { return "replicaset/unavailable" }

func (UnavailableRule) Description() string {
	return "Detects ReplicaSets with unavailable replicas"
}

func (UnavailableRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindReplicaSet}
}

func (r UnavailableRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	rs, ok := graph.As[*appsv1.ReplicaSet](node)
	if !ok {
		return nil
	}

	desired := int32(1)
	if rs.Spec.Replicas != nil {
		desired = *rs.Spec.Replicas
	}
	if desired == 0 || rs.Status.ReadyReplicas >= desired {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      fmt.Sprintf("ReplicaSet has %d/%d replicas ready", rs.Status.ReadyReplicas, desired),
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.65,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Status", fmt.Sprintf("ready=%d desired=%d available=%d", rs.Status.ReadyReplicas, desired, rs.Status.AvailableReplicas), ""),
			},
			Explanation: "The ReplicaSet cannot reach its desired replica count. Inspect the owned pods to find the underlying cause.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the ReplicaSet and its pods",
					Command: fmt.Sprintf("kubectl describe replicaset %s -n %s", rs.Name, rs.Namespace),
				},
			},
		},
	}
}

// ReplicaFailureRule flags ReplicaSets whose ReplicaFailure condition is true,
// which surfaces quota, limit-range or admission errors during pod creation.
type ReplicaFailureRule struct{}

func (ReplicaFailureRule) ID() string { return "replicaset/replica-failure" }

func (ReplicaFailureRule) Description() string {
	return "Detects ReplicaSets reporting a ReplicaFailure condition"
}

func (ReplicaFailureRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindReplicaSet}
}

func (r ReplicaFailureRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	rs, ok := graph.As[*appsv1.ReplicaSet](node)
	if !ok {
		return nil
	}

	for _, condition := range rs.Status.Conditions {
		if condition.Type != appsv1.ReplicaSetReplicaFailure || condition.Status != "True" {
			continue
		}

		return []diagnose.Finding{
			{
				ID:         r.ID(),
				Title:      "ReplicaSet cannot create pods",
				Severity:   diagnose.SeverityError,
				Confidence: 0.8,
				Resource:   node.Ref,
				Evidence: []diagnose.Evidence{
					diagnose.NewEvidence(node.Ref, "Condition", condition.Message, condition.Reason),
				},
				Explanation: "The ReplicaSet failed to create pods, often due to resource quotas, limit ranges, or admission webhooks rejecting the pod template.",
				Suggestions: []diagnose.Suggestion{
					{
						Title:   "Inspect the ReplicaSet conditions and namespace quotas",
						Command: fmt.Sprintf("kubectl describe replicaset %s -n %s", rs.Name, rs.Namespace),
					},
				},
			},
		}
	}

	return nil
}
