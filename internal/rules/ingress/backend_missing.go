// Package ingress contains diagnostic rules for Ingresses.
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

// BackendMissingRule flags ingresses whose backend services do not exist.
type BackendMissingRule struct{}

// ID returns the rule identifier.
func (BackendMissingRule) ID() string { return "ingress/backend-missing" }

// Description returns a human-readable description of the rule.
func (BackendMissingRule) Description() string {
	return "Detects ingress backends that reference missing services"
}

// AppliesTo returns the kinds this rule evaluates.
func (BackendMissingRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindIngress}
}

// Evaluate checks that every backend service exists in the graph.
func (r BackendMissingRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	ing, ok := graph.As[*networkingv1.Ingress](node)
	if !ok {
		return nil
	}

	keepServices := func(rel kube.Relationship) bool {
		return rel.Target.Kind == resource.ReferenceKindService
	}

	return ruleutil.MissingRelationships(ctx, kube.TypedRelationships(ing), keepServices, func(rel kube.Relationship) diagnose.Finding {
		name := rel.Target.Name
		return diagnose.Finding{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Ingress backend service %q does not exist", name),
			Severity:   diagnose.SeverityError,
			Confidence: 0.85,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Backend", fmt.Sprintf("service %q is referenced but missing (%s)", name, rel.Path), ""),
			},
			Explanation: fmt.Sprintf("The ingress routes traffic to service %q which is not present in namespace %q.", name, ing.Namespace),
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Verify the backend service exists",
					Command: fmt.Sprintf("kubectl get service %s -n %s", name, ing.Namespace),
				},
			},
		}
	})
}
