package cronjob_test

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/cronjob"
	"github.com/gabor-boros/klue/pkg/resource"
)

func cronNode(cj *batchv1.CronJob) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindCronJob, "batch/v1", cj.Namespace, cj.Name, string(cj.UID)),
		Object: cj,
	}
}

func boolPtr(b bool) *bool { return &b }

func TestSuspendedRule(t *testing.T) {
	t.Parallel()

	suspended := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "backup"},
		Spec:       batchv1.CronJobSpec{Suspend: boolPtr(true)},
	}
	if got := (cronjob.SuspendedRule{}).Evaluate(diagnose.RuleContext{}, cronNode(suspended)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for suspended cronjob", len(got))
	}

	active := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "backup"},
		Spec:       batchv1.CronJobSpec{Suspend: boolPtr(false)},
	}
	if got := (cronjob.SuspendedRule{}).Evaluate(diagnose.RuleContext{}, cronNode(active)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for active cronjob", len(got))
	}
}

func TestJobFailuresRule(t *testing.T) {
	t.Parallel()

	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "backup", UID: "cj1"},
	}
	failedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "backup-123", UID: "j1"},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue}},
		},
	}

	builder := graph.NewBuilder()
	cn := cronNode(cj)
	jn := graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindJob, "batch/v1", "default", "backup-123", "j1"),
		Object: failedJob,
	}
	builder.AddNode(*cn)
	builder.AddNode(jn)
	builder.AddEdge(graph.Edge{Kind: graph.EdgeOwns, From: *cn, To: jn})
	g := builder.Build()

	if got := (cronjob.JobFailuresRule{}).Evaluate(diagnose.RuleContext{Graph: g}, cn); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for failing owned job", len(got))
	}
}
