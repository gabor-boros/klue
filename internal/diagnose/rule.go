package diagnose

import (
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// Rule evaluates a single node of the resource graph and reports findings that
// explain why the node may be unhealthy.
type Rule interface {
	ID() string
	Description() string
	AppliesTo() []resource.Kind
	Evaluate(ctx RuleContext, node *graph.Node) []Finding
}
