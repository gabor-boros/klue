package diagnose

import (
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/graph"
)

// RuleContext carries the shared, read-only state that diagnostic rules use
// while evaluating a node of the resource graph.
type RuleContext struct {
	Graph   *graph.Graph
	Events  *evidence.EventIndex
	Options DiagnoseOptions
}
