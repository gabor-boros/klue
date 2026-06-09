// Package cronjob contains diagnostic rules for CronJobs.
package cronjob

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// SuspendedRule flags CronJobs that are suspended and therefore never fire.
type SuspendedRule struct{}

func (SuspendedRule) ID() string { return "cronjob/suspended" }

func (SuspendedRule) Description() string {
	return "Detects CronJobs that are suspended"
}

func (SuspendedRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindCronJob}
}

func (r SuspendedRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	cj, ok := graph.As[*batchv1.CronJob](node)
	if !ok || cj.Spec.Suspend == nil || !*cj.Spec.Suspend {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "CronJob is suspended",
			Severity:   diagnose.SeverityInfo,
			Confidence: 0.9,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Spec", "spec.suspend=true", "Suspended"),
			},
			Explanation: "The CronJob is suspended, so no new Jobs are created until it is resumed. This may be intentional.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Resume the CronJob if it should be running",
					Command: fmt.Sprintf("kubectl patch cronjob %s -n %s -p '{\"spec\":{\"suspend\":false}}'", cj.Name, cj.Namespace),
				},
			},
		},
	}
}

// JobFailuresRule flags CronJobs whose most recent owned Jobs are failing.
type JobFailuresRule struct{}

func (JobFailuresRule) ID() string { return "cronjob/job-failures" }

func (JobFailuresRule) Description() string {
	return "Detects CronJobs whose recent Jobs are failing"
}

func (JobFailuresRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindCronJob}
}

func (r JobFailuresRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	cj, ok := graph.As[*batchv1.CronJob](node)
	if !ok || ctx.Graph == nil {
		return nil
	}

	var failed []string
	for _, owned := range ctx.Graph.GetOutboundNodes(*node) {
		j, ok := owned.Object.(*batchv1.Job)
		if !ok {
			continue
		}
		if jobFailed(j) {
			failed = append(failed, j.Name)
		}
	}

	if len(failed) == 0 {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      fmt.Sprintf("CronJob has %d failing Job(s)", len(failed)),
			Severity:   diagnose.SeverityError,
			Confidence: 0.75,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Jobs", fmt.Sprintf("failing jobs: %v", failed), "JobFailures"),
			},
			Explanation: "Jobs created by this CronJob are failing, so the scheduled task is not completing successfully.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the most recent Job created by the CronJob",
					Command: fmt.Sprintf("kubectl get jobs -n %s --sort-by=.metadata.creationTimestamp", cj.Namespace),
				},
			},
		},
	}
}

// jobFailed reports whether a Job has a true Failed condition.
func jobFailed(j *batchv1.Job) bool {
	for _, condition := range j.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
