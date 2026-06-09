// Package pv contains diagnostic rules for PersistentVolumes.
package pv

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// FailedRule flags PersistentVolumes in the Failed phase.
type FailedRule struct{}

func (FailedRule) ID() string { return "pv/failed" }

func (FailedRule) Description() string {
	return "Detects PersistentVolumes that failed recycling or deletion"
}

func (FailedRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPersistentVolume}
}

func (r FailedRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	volume, ok := graph.As[*corev1.PersistentVolume](node)
	if !ok || volume.Status.Phase != corev1.VolumeFailed {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "PersistentVolume is in the Failed phase",
			Severity:   diagnose.SeverityError,
			Confidence: 0.85,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Status", fmt.Sprintf("phase=Failed reason=%s message=%s", volume.Status.Reason, volume.Status.Message), volume.Status.Reason),
			},
			Explanation: "The volume's automatic reclamation (recycle or delete) failed, so the backing storage may need manual cleanup.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the PersistentVolume status",
					Command: fmt.Sprintf("kubectl describe pv %s", volume.Name),
				},
			},
		},
	}
}

// ReleasedRetainedRule flags PersistentVolumes that are Released with a Retain
// reclaim policy, so they will never be reused until an operator intervenes.
type ReleasedRetainedRule struct{}

func (ReleasedRetainedRule) ID() string { return "pv/released-retained" }

func (ReleasedRetainedRule) Description() string {
	return "Detects released PersistentVolumes retained and unavailable for reuse"
}

func (ReleasedRetainedRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPersistentVolume}
}

func (r ReleasedRetainedRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	volume, ok := graph.As[*corev1.PersistentVolume](node)
	if !ok || volume.Status.Phase != corev1.VolumeReleased {
		return nil
	}
	if volume.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "PersistentVolume is released but retained",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.7,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Status", "phase=Released reclaimPolicy=Retain", "Released"),
			},
			Explanation: "The bound claim was deleted but the Retain policy keeps the volume. It will not be re-bound automatically and must be cleaned up or re-claimed manually.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Clear the claimRef to make the volume Available again",
					Command: fmt.Sprintf("kubectl patch pv %s -p '{\"spec\":{\"claimRef\":null}}'", volume.Name),
				},
			},
		},
	}
}
