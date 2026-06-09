package lease_test

import (
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/lease"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(l *coordinationv1.Lease) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindLease, "coordination.k8s.io/v1", l.Namespace, l.Name, string(l.UID)),
		Object: l,
	}
}

func int32Ptr(n int32) *int32 { return &n }
func strPtr(s string) *string { return &s }

func TestStaleRule(t *testing.T) {
	t.Parallel()

	now := time.Now()
	stale := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system", Name: "leader"},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       strPtr("controller-a"),
			LeaseDurationSeconds: int32Ptr(15),
			RenewTime:            &metav1.MicroTime{Time: now.Add(-10 * time.Minute)},
		},
	}
	ctx := diagnose.RuleContext{Options: diagnose.DiagnoseOptions{Now: now}}
	if got := (lease.StaleRule{}).Evaluate(ctx, node(stale)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for stale lease", len(got))
	}

	fresh := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system", Name: "leader"},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       strPtr("controller-a"),
			LeaseDurationSeconds: int32Ptr(15),
			RenewTime:            &metav1.MicroTime{Time: now.Add(-5 * time.Second)},
		},
	}
	if got := (lease.StaleRule{}).Evaluate(ctx, node(fresh)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for a freshly renewed lease", len(got))
	}

	// Without a reference clock the rule is silent.
	if got := (lease.StaleRule{}).Evaluate(diagnose.RuleContext{}, node(stale)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 without a reference time", len(got))
	}
}
