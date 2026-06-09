package pod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

// ConfigMissingRule flags pods that reference ConfigMaps or Secrets that do not
// exist in the cluster.
type ConfigMissingRule struct{}

// ID returns the rule identifier.
func (ConfigMissingRule) ID() string { return "pod/config-missing" }

// Description returns a human-readable description of the rule.
func (ConfigMissingRule) Description() string {
	return "Detects references to missing ConfigMaps or Secrets"
}

// AppliesTo returns the kinds this rule evaluates.
func (ConfigMissingRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindPod}
}

// Evaluate checks that every referenced ConfigMap and Secret exists in the graph.
func (r ConfigMissingRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	pod, ok := graph.As[*corev1.Pod](node)
	if !ok {
		return nil
	}

	keepConfigRefs := func(rel kube.Relationship) bool {
		return rel.Target.Kind == resource.ReferenceKindConfigMap || rel.Target.Kind == resource.ReferenceKindSecret
	}

	return ruleutil.MissingRelationships(ctx, kube.TypedRelationships(pod), keepConfigRefs, func(rel kube.Relationship) diagnose.Finding {
		label := configRefLabel(rel.Target.Kind)
		name := rel.Target.Name
		return diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("%s %q referenced by the pod does not exist", label, name),
			Severity:   diagnose.SeverityError,
			Confidence: 0.85,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Reference", fmt.Sprintf("%s/%s is referenced but missing (%s)", label, name, rel.Path), ""),
			},
			Explanation: fmt.Sprintf("The pod references %s %q which is not present in namespace %q.", label, name, pod.Namespace),
			Suggestions: []diagnose.Suggestion{
				{
					Title:   fmt.Sprintf("Create or correct the %s reference", label),
					Command: fmt.Sprintf("kubectl get %s %s -n %s", kindResource(label), name, pod.Namespace),
				},
			},
		}
	})
}

// configRefLabel returns the human-readable label for a config reference kind.
func configRefLabel(kind resource.Kind) string {
	if kind == resource.ReferenceKindSecret {
		return "Secret"
	}
	return "ConfigMap"
}

func kindResource(label string) string {
	if label == "Secret" {
		return "secret"
	}
	return "configmap"
}
