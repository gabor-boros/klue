package pvc_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/pvc"
	"github.com/gabor-boros/klue/pkg/resource"
)

func pvcNode(claim *corev1.PersistentVolumeClaim) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindPersistentVolumeClaim, "v1", claim.Namespace, claim.Name, string(claim.UID)),
		Object: claim,
	}
}

func TestUnboundRule(t *testing.T) {
	t.Parallel()

	pending := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "data"},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
	}
	if got := (pvc.UnboundRule{}).Evaluate(diagnose.RuleContext{}, pvcNode(pending)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for pending PVC", len(got))
	}

	bound := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "data"},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
	}
	if got := (pvc.UnboundRule{}).Evaluate(diagnose.RuleContext{}, pvcNode(bound)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for bound PVC", len(got))
	}
}

func TestMissingStorageClassRule(t *testing.T) {
	t.Parallel()

	className := "fast"
	claim := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "data"},
		Spec:       corev1.PersistentVolumeClaimSpec{StorageClassName: &className},
	}

	builder := graph.NewBuilder()
	builder.AddNode(*pvcNode(claim))
	g := builder.Build()

	if got := (pvc.MissingStorageClassRule{}).Evaluate(diagnose.RuleContext{Graph: g}, pvcNode(claim)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 (missing storage class)", len(got))
	}

	builder.AddNode(graph.Node{Ref: resource.NewReference(resource.ReferenceKindStorageClass, "storage.k8s.io/v1", "", className, "")})
	g = builder.Build()
	if got := (pvc.MissingStorageClassRule{}).Evaluate(diagnose.RuleContext{Graph: g}, pvcNode(claim)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 once storage class exists", len(got))
	}
}
