package builtin

import (
	"fmt"
	"sort"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

// MissingReferenceRule reports unresolved placeholder nodes introduced by graph
// relationship extraction. When a producer edge exists, traversal should
// continue to that producer instead of stopping at the placeholder.
type MissingReferenceRule struct{}

// ID returns the rule identifier.
func (MissingReferenceRule) ID() string { return "builtin/missing-reference" }

// Description returns a human-readable description of the rule.
func (MissingReferenceRule) Description() string {
	return "Detects unresolved resource references represented as placeholder nodes"
}

// AppliesTo returns the kinds this rule evaluates.
func (MissingReferenceRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.KindAny}
}

// Evaluate emits a finding for missing placeholder nodes without producer links.
func (r MissingReferenceRule) Evaluate(ctx diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	if ctx.Graph == nil || !node.IsPlaceholder() {
		return nil
	}

	if kube.RelationshipHasProducer(ctx.Graph, *node) {
		return nil
	}

	inbound := ctx.Graph.GetInboundEdges(*node)
	if len(inbound) == 0 {
		return nil
	}

	sort.Slice(inbound, func(i, j int) bool {
		if inbound[i].From.Ref.LogicalKey() != inbound[j].From.Ref.LogicalKey() {
			return inbound[i].From.Ref.LogicalKey() < inbound[j].From.Ref.LogicalKey()
		}
		return inbound[i].Reason < inbound[j].Reason
	})

	evidenceMessage := fmt.Sprintf("%s %q is referenced but missing", node.Ref.Kind, node.Ref.Name)
	if path := inbound[0].Data.Path; path != "" {
		evidenceMessage = fmt.Sprintf("%s (%s)", evidenceMessage, path)
	}

	command := fmt.Sprintf("kubectl get %s %s -n %s", describeKind(node.Ref.Kind), node.Ref.Name, node.Ref.Namespace)
	if entry, found := kube.LookupKind(node.Ref.Kind); found && !entry.Namespaced {
		command = fmt.Sprintf("kubectl get %s %s", describeKind(node.Ref.Kind), node.Ref.Name)
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      fmt.Sprintf("Missing referenced %s %q", node.Ref.Kind, node.Ref.Name),
			Severity:   diagnose.SeverityError,
			Confidence: 0.7,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(inbound[0].From.Ref, diagnose.EvidenceReference, evidenceMessage, inbound[0].Reason),
			},
			Explanation: fmt.Sprintf("At least one resource references %s %q, but that object does not exist in the cluster.", node.Ref.Kind, node.Ref.Name),
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Create the missing resource or fix the reference",
					Command: command,
				},
			},
		},
	}
}
