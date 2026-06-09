// Package job contains diagnostic rules for batch Jobs.
package job

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
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

func (r FailedRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
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

		return []diagnose.Finding{
			{
				ID:         r.ID(),
				Title:      title,
				Severity:   diagnose.SeverityError,
				Confidence: 0.85,
				Resource:   node.Ref,
				Evidence: []diagnose.Evidence{
					diagnose.NewEvidence(node.Ref, "Condition", condition.Message, condition.Reason),
					diagnose.NewEvidence(node.Ref, "Status", fmt.Sprintf("failed=%d succeeded=%d active=%d", j.Status.Failed, j.Status.Succeeded, j.Status.Active), ""),
				},
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
