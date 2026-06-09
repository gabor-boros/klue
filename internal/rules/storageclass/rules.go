// Package storageclass contains diagnostic rules for StorageClasses.
package storageclass

import (
	"fmt"

	storagev1 "k8s.io/api/storage/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// noProvisioner is the sentinel provisioner used for statically provisioned
// volumes; it never dynamically provisions PVCs.
const noProvisioner = "kubernetes.io/no-provisioner"

// NoProvisionerRule flags StorageClasses that cannot dynamically provision
// volumes because they use the no-provisioner sentinel.
type NoProvisionerRule struct{}

func (NoProvisionerRule) ID() string { return "storageclass/no-provisioner" }

func (NoProvisionerRule) Description() string {
	return "Detects StorageClasses that cannot dynamically provision volumes"
}

func (NoProvisionerRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindStorageClass}
}

func (r NoProvisionerRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	sc, ok := graph.As[*storagev1.StorageClass](node)
	if !ok || sc.Provisioner != noProvisioner {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "StorageClass cannot dynamically provision volumes",
			Severity:   diagnose.SeverityInfo,
			Confidence: 0.6,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Spec", fmt.Sprintf("provisioner=%s", sc.Provisioner), "NoProvisioner"),
			},
			Explanation: "This StorageClass uses the no-provisioner sentinel, so PVCs using it stay Pending until a matching PersistentVolume is created manually.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Pre-provision matching PersistentVolumes for this class",
					Command: fmt.Sprintf("kubectl get pv -o wide | grep %s", sc.Name),
				},
			},
		},
	}
}

// WaitForFirstConsumerRule informs that binding is deferred until a consuming
// pod is scheduled, which explains PVCs that remain Pending without errors.
type WaitForFirstConsumerRule struct{}

func (WaitForFirstConsumerRule) ID() string { return "storageclass/wait-for-first-consumer" }

func (WaitForFirstConsumerRule) Description() string {
	return "Explains deferred binding for WaitForFirstConsumer StorageClasses"
}

func (WaitForFirstConsumerRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindStorageClass}
}

func (r WaitForFirstConsumerRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	sc, ok := graph.As[*storagev1.StorageClass](node)
	if !ok || sc.VolumeBindingMode == nil || *sc.VolumeBindingMode != storagev1.VolumeBindingWaitForFirstConsumer {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "StorageClass defers binding until a consumer is scheduled",
			Severity:   diagnose.SeverityInfo,
			Confidence: 0.5,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Spec", "volumeBindingMode=WaitForFirstConsumer", "DeferredBinding"),
			},
			Explanation: "PVCs using this class stay Pending by design until a pod that mounts them is scheduled. A Pending PVC may simply have no consuming pod yet.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Confirm a pod consuming the PVC has been scheduled",
					Command: "kubectl get pods -A -o wide | grep -i pending",
				},
			},
		},
	}
}
