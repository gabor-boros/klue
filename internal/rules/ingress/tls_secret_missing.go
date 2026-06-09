package ingress

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

// TLSSecretMissingRule flags ingresses referencing missing TLS secrets.
type TLSSecretMissingRule struct{}

// ID returns the rule identifier.
func (TLSSecretMissingRule) ID() string { return "ingress/tls-secret-missing" }

// Description returns a human-readable description of the rule.
func (TLSSecretMissingRule) Description() string {
	return "Detects ingress TLS configuration referencing missing secrets"
}

// AppliesTo returns the kinds this rule evaluates.
func (TLSSecretMissingRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindIngress}
}

// Evaluate checks that every TLS secret exists in the graph.
func (r TLSSecretMissingRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	ing, ok := graph.As[*networkingv1.Ingress](node)
	if !ok {
		return nil
	}

	keepTLSSecrets := func(rel kube.Relationship) bool {
		return rel.EdgeKind == graph.EdgeUsesSecret && rel.Reason == "tls"
	}

	return ruleutil.MissingRelationships(ctx, kube.TypedRelationships(ing), keepTLSSecrets, func(rel kube.Relationship) diagnose.Finding {
		name := rel.Target.Name
		return diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Ingress TLS secret %q does not exist", name),
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.75,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "TLS", fmt.Sprintf("secret %q is referenced but missing (%s)", name, rel.Path), ""),
			},
			Explanation: fmt.Sprintf("The ingress references TLS secret %q which is not present in namespace %q, so TLS termination will fail.", name, ing.Namespace),
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Create or correct the TLS secret",
					Command: fmt.Sprintf("kubectl get secret %s -n %s", name, ing.Namespace),
				},
			},
		}
	})
}
