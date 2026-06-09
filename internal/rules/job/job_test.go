package job_test

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/job"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(j *batchv1.Job) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindJob, "batch/v1", j.Namespace, j.Name, string(j.UID)),
		Object: j,
	}
}

func TestFailedRule(t *testing.T) {
	t.Parallel()

	backoff := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "import"},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: "BackoffLimitExceeded", Message: "Job has reached the specified backoff limit"},
			},
		},
	}
	findings := job.FailedRule{}.Evaluate(diagnose.RuleContext{}, node(backoff))
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if findings[0].Title != "Job exceeded its backoff limit" {
		t.Errorf("Title = %q, want backoff-specific title", findings[0].Title)
	}

	running := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "import"},
		Status:     batchv1.JobStatus{Active: 1},
	}
	if got := (job.FailedRule{}).Evaluate(diagnose.RuleContext{}, node(running)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for a running job", len(got))
	}
}
