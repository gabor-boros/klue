package service

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// NoEndpointsRule flags services that have no ready endpoints.
type NoEndpointsRule struct{}

// ID returns the rule identifier.
func (NoEndpointsRule) ID() string { return "service/no-endpoints" }

// Description returns a human-readable description of the rule.
func (NoEndpointsRule) Description() string {
	return "Detects services with no ready endpoints"
}

// AppliesTo returns the kinds this rule evaluates.
func (NoEndpointsRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindService}
}

// Evaluate checks the backing Endpoints object for ready addresses.
func (r NoEndpointsRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	svc, ok := graph.As[*corev1.Service](node)
	if !ok || len(svc.Spec.Selector) == 0 || ctx.Graph == nil {
		return nil
	}

	if hasReadyEndpoints(ctx, node) {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Service has no ready endpoints",
			Severity:   diagnose.SeverityError,
			Confidence: 0.85,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Endpoints", "no ready addresses back this service", ""),
			},
			Explanation: "Traffic to the service will fail because no ready pods back it. The selected pods may be unhealthy or missing.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect the service endpoint slices",
					Command: fmt.Sprintf("kubectl get endpointslices -n %s -l %s=%s", svc.Namespace, discoveryv1.LabelServiceName, svc.Name),
				},
			},
		},
	}
}

// hasReadyEndpoints reports whether any EndpointSlice backing the service has at
// least one ready address. Slices are linked to the service via inbound graph
// edges.
func hasReadyEndpoints(ctx diagnose.RuleContext, node *graph.Node) bool {
	for _, related := range ctx.Graph.GetInboundNodes(*node) {
		if related.Ref.Kind != resource.ReferenceKindEndpointSlice {
			continue
		}

		slice, ok := related.Object.(*discoveryv1.EndpointSlice)
		if !ok {
			continue
		}

		for _, endpoint := range slice.Endpoints {
			if endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready && len(endpoint.Addresses) > 0 {
				return true
			}
		}
	}

	return false
}
