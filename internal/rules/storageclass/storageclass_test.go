package storageclass_test

import (
	"testing"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/storageclass"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(sc *storagev1.StorageClass) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindStorageClass, "storage.k8s.io/v1", "", sc.Name, string(sc.UID)),
		Object: sc,
	}
}

func TestNoProvisionerRule(t *testing.T) {
	t.Parallel()

	static := &storagev1.StorageClass{
		ObjectMeta:  metav1.ObjectMeta{Name: "local"},
		Provisioner: "kubernetes.io/no-provisioner",
	}
	if got := (storageclass.NoProvisionerRule{}).Evaluate(diagnose.RuleContext{}, node(static)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for no-provisioner class", len(got))
	}

	dynamic := &storagev1.StorageClass{
		ObjectMeta:  metav1.ObjectMeta{Name: "fast"},
		Provisioner: "ebs.csi.aws.com",
	}
	if got := (storageclass.NoProvisionerRule{}).Evaluate(diagnose.RuleContext{}, node(dynamic)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for dynamic provisioner", len(got))
	}
}

func TestWaitForFirstConsumerRule(t *testing.T) {
	t.Parallel()

	mode := storagev1.VolumeBindingWaitForFirstConsumer
	wait := &storagev1.StorageClass{
		ObjectMeta:        metav1.ObjectMeta{Name: "fast"},
		Provisioner:       "ebs.csi.aws.com",
		VolumeBindingMode: &mode,
	}
	if got := (storageclass.WaitForFirstConsumerRule{}).Evaluate(diagnose.RuleContext{}, node(wait)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for WaitForFirstConsumer", len(got))
	}

	immediate := storagev1.VolumeBindingImmediate
	now := &storagev1.StorageClass{
		ObjectMeta:        metav1.ObjectMeta{Name: "fast"},
		Provisioner:       "ebs.csi.aws.com",
		VolumeBindingMode: &immediate,
	}
	if got := (storageclass.WaitForFirstConsumerRule{}).Evaluate(diagnose.RuleContext{}, node(now)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for Immediate binding", len(got))
	}
}
