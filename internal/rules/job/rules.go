// Package job contains diagnostic rules for batch Jobs.
package job

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

// FailedRule flags Jobs whose Failed condition is true, tailoring the
// explanation to the failure reason (backoff limit or deadline exceeded).
type FailedRule struct{}

func (FailedRule) ID() string { return "job/failed" }

func (FailedRule) Description() string {
	return "Detects Jobs that failed, including backoff and deadline exhaustion"
}

func (FailedRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindJob}
}

func (r FailedRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	j, ok := graph.As[*batchv1.Job](node)
	if !ok {
		return nil
	}

	for _, condition := range j.Status.Conditions {
		if condition.Type != batchv1.JobFailed || condition.Status != corev1.ConditionTrue {
			continue
		}

		title := "Job failed"
		explanation := "The Job did not complete successfully. Inspect the failed pods to find the root cause."
		switch condition.Reason {
		case "BackoffLimitExceeded":
			title = "Job exceeded its backoff limit"
			explanation = "The Job retried its pods up to backoffLimit and gave up. The container is failing on every attempt."
		case "DeadlineExceeded":
			title = "Job exceeded its active deadline"
			explanation = "The Job ran longer than activeDeadlineSeconds and was terminated before completing."
		}

		evidenceItems := make([]diagnose.Evidence, 0, 2)
		evidenceItems = append(
			evidenceItems,
			diagnose.NewEvidence(node.Ref, diagnose.EvidenceCondition, condition.Message, condition.Reason),
			diagnose.NewEvidence(node.Ref, diagnose.EvidenceStatus, fmt.Sprintf("failed=%d succeeded=%d active=%d", j.Status.Failed, j.Status.Succeeded, j.Status.Active), ""),
		)
		evidenceItems = append(evidenceItems, jobPodLogEvidence(ctx, node, j.Namespace)...)

		if logExplanation := jobPodLogExplanation(ctx, node, j.Namespace); logExplanation != "" {
			explanation += logExplanation
		}

		return []diagnose.Finding{
			{
				ID:          r.ID(),
				Title:       title,
				Severity:    diagnose.SeverityError,
				Confidence:  0.85,
				Resource:    node.Ref,
				Evidence:    evidenceItems,
				Explanation: explanation,
				Suggestions: []diagnose.Suggestion{
					{
						Title:   "Inspect the failed Job pods",
						Command: fmt.Sprintf("kubectl logs job/%s -n %s", j.Name, j.Namespace),
					},
				},
			},
		}
	}

	return nil
}

func jobPodLogEvidence(ctx diagnose.RuleContext, jobNode *graph.Node, namespace string) []diagnose.Evidence {
	if ctx.Graph == nil || ctx.Logs == nil {
		return nil
	}

	var evidenceItems []diagnose.Evidence
	for _, podNode := range ownedPods(ctx.Graph, jobNode) {
		pod, ok := graph.As[*corev1.Pod](&podNode)
		if !ok {
			continue
		}
		for _, status := range pod.Status.ContainerStatuses {
			evidenceItems = append(evidenceItems, ruleutil.LogEvidence(ctx, podNode.Ref, status.Name, false)...)
		}
		_ = namespace
	}
	return evidenceItems
}

func jobPodLogExplanation(ctx diagnose.RuleContext, jobNode *graph.Node, namespace string) string {
	if ctx.Graph == nil || ctx.Logs == nil {
		return ""
	}

	for _, podNode := range ownedPods(ctx.Graph, jobNode) {
		pod, ok := graph.As[*corev1.Pod](&podNode)
		if !ok {
			continue
		}
		for _, status := range pod.Status.ContainerStatuses {
			if explanation := ruleutil.LogExplanation(ctx, podNode.Ref, status.Name); explanation != "" {
				return explanation
			}
		}
		_ = namespace
	}
	return ""
}

func ownedPods(g *graph.Graph, owner *graph.Node) []graph.Node {
	var pods []graph.Node
	for _, edge := range g.GetOutboundEdges(*owner) {
		if edge.Kind != graph.EdgeOwns {
			continue
		}
		if edge.To.Ref.Kind == resource.ReferenceKindPod {
			pods = append(pods, edge.To)
		}
	}
	return pods
}
