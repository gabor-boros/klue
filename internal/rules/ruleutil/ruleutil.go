// Package ruleutil provides helpers shared across the diagnostic rule packages
// so that common patterns are implemented once and behave consistently.
package ruleutil

import (
	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

// KubectlKind returns the lowercase, kubectl-friendly token for a resource kind
// (for example "deployment" for "Deployment"). It falls back to the kind name
// when the kind is not in the catalog, so it is safe for custom resources.
func KubectlKind(kind resource.Kind) string {
	if entry, ok := kube.LookupCommandToken(string(kind)); ok {
		return entry.CommandToken()
	}
	return string(kind)
}

// MissingRelationships returns a finding for every relationship that passes the
// keep filter and whose target is absent from the graph. It centralises the
// "this object references something that does not exist" check shared by the
// typed reference rules: the graph nil-guard, the target lookup and the
// per-relationship iteration.
//
// keep may be nil to consider every relationship. build is invoked only for
// relationships whose target is missing and is responsible for producing the
// rule-specific finding.
func MissingRelationships(
	ctx diagnose.RuleContext,
	relationships []kube.Relationship,
	keep func(kube.Relationship) bool,
	build func(kube.Relationship) diagnose.Finding,
) []diagnose.Finding {
	if ctx.Graph == nil {
		return nil
	}

	var findings []diagnose.Finding
	for _, relationship := range relationships {
		if keep != nil && !keep(relationship) {
			continue
		}
		if _, found := ctx.Graph.FindByRef(relationship.Target); found {
			continue
		}
		findings = append(findings, build(relationship))
	}

	return findings
}
