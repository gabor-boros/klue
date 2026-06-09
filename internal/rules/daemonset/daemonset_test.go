package daemonset_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/daemonset"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(ds *appsv1.DaemonSet) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindDaemonSet, "apps/v1", ds.Namespace, ds.Name, string(ds.UID)),
		Object: ds,
	}
}

func TestUnavailableRule(t *testing.T) {
	t.Parallel()

	degraded := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system", Name: "agent"},
		Status:     appsv1.DaemonSetStatus{DesiredNumberScheduled: 3, NumberReady: 1, NumberUnavailable: 2},
	}
	if got := (daemonset.UnavailableRule{}).Evaluate(diagnose.RuleContext{}, node(degraded)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(got))
	}

	healthy := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system", Name: "agent"},
		Status:     appsv1.DaemonSetStatus{DesiredNumberScheduled: 3, NumberReady: 3},
	}
	if got := (daemonset.UnavailableRule{}).Evaluate(diagnose.RuleContext{}, node(healthy)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 when fully ready", len(got))
	}
}

func TestMisscheduledRule(t *testing.T) {
	t.Parallel()

	miss := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system", Name: "agent"},
		Status:     appsv1.DaemonSetStatus{NumberMisscheduled: 2},
	}
	if got := (daemonset.MisscheduledRule{}).Evaluate(diagnose.RuleContext{}, node(miss)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for misscheduled pods", len(got))
	}

	ok := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system", Name: "agent"}}
	if got := (daemonset.MisscheduledRule{}).Evaluate(diagnose.RuleContext{}, node(ok)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 with no misscheduled pods", len(got))
	}
}
