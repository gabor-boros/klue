package pv_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/pv"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(volume *corev1.PersistentVolume) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindPersistentVolume, "v1", "", volume.Name, string(volume.UID)),
		Object: volume,
	}
}

func TestFailedRule(t *testing.T) {
	t.Parallel()

	failed := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "vol-1"},
		Status:     corev1.PersistentVolumeStatus{Phase: corev1.VolumeFailed, Reason: "RecycleFailed"},
	}
	if got := (pv.FailedRule{}).Evaluate(diagnose.RuleContext{}, node(failed)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for failed PV", len(got))
	}

	bound := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "vol-1"},
		Status:     corev1.PersistentVolumeStatus{Phase: corev1.VolumeBound},
	}
	if got := (pv.FailedRule{}).Evaluate(diagnose.RuleContext{}, node(bound)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for bound PV", len(got))
	}
}

func TestReleasedRetainedRule(t *testing.T) {
	t.Parallel()

	retained := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "vol-1"},
		Spec:       corev1.PersistentVolumeSpec{PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain},
		Status:     corev1.PersistentVolumeStatus{Phase: corev1.VolumeReleased},
	}
	if got := (pv.ReleasedRetainedRule{}).Evaluate(diagnose.RuleContext{}, node(retained)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for released+retained PV", len(got))
	}

	deleted := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "vol-1"},
		Spec:       corev1.PersistentVolumeSpec{PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete},
		Status:     corev1.PersistentVolumeStatus{Phase: corev1.VolumeReleased},
	}
	if got := (pv.ReleasedRetainedRule{}).Evaluate(diagnose.RuleContext{}, node(deleted)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for released PV with Delete policy", len(got))
	}
}
