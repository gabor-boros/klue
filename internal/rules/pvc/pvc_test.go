package pvc_test

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
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

func TestProvisionerStuckRule(t *testing.T) {
	t.Parallel()

	claim := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "data", UID: "pvc-uid"},
		Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
	}
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-1"},
			Type:       corev1.EventTypeWarning,
			Reason:     "ProvisioningFailed",
			Message:    "failed to provision volume with StorageClass fast: permission denied",
			LastTimestamp: metav1.Time{
				Time: time.Now(),
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:       "PersistentVolumeClaim",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "data",
				UID:        "pvc-uid",
			},
		},
	}

	findings := (pvc.ProvisionerStuckRule{}).Evaluate(diagnose.RuleContext{
		Events: evidence.NewEventIndex(events),
	}, pvcNode(claim))
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if findings[0].Evidence[0].Type != diagnose.EvidenceEvent {
		t.Fatalf("evidence type = %q, want %q", findings[0].Evidence[0].Type, diagnose.EvidenceEvent)
	}
}
