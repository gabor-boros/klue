package graph_test

import (
	"testing"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestNewBuilder(t *testing.T) {
	t.Parallel()

	builder := graph.NewBuilder()
	kubeGraph := builder.Build()

	if kubeGraph == nil {
		t.Errorf("NewBuilder() = %v, want non-nil", kubeGraph)
	}
}

func TestBuilder_AddNode(t *testing.T) {
	t.Parallel()

	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
	})

	kubeGraph := builder.Build()
	nodes := kubeGraph.GetNodes()

	if len(nodes) != 1 {
		t.Errorf("NewBuilder() = %v, want 1 node", nodes)
	}
}

func TestBuilder_AddEdge(t *testing.T) {
	t.Parallel()

	builder := graph.NewBuilder()
	builder.AddEdge(graph.Edge{
		Kind: graph.EdgeOwns,
		From: graph.Node{
			Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
		},
		To: graph.Node{
			Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
		},
	})

	kubeGraph := builder.Build()
	edges := kubeGraph.GetEdges()

	if len(edges) != 1 {
		t.Errorf("NewBuilder() = %v, want 1 edge", edges)
	}
}

func TestBuilder_Build(t *testing.T) {
	t.Parallel()

	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
	})
	builder.AddEdge(graph.Edge{
		Kind: graph.EdgeOwns,
		From: graph.Node{
			Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
		},
		To: graph.Node{
			Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
		},
	})

	kubeGraph := builder.Build()
	edges := kubeGraph.GetEdges()

	if len(edges) != 1 {
		t.Errorf("NewBuilder() = %v, want 1 edge", edges)
	}
	if edges[0].Kind != graph.EdgeOwns {
		t.Errorf("NewBuilder() = %v, want edge with Kind %s", edges, graph.EdgeOwns)
	}
	if edges[0].From.Ref != resource.NewReference("v1", "Pod", "default", "test-pod", "") {
		t.Errorf("NewBuilder() = %v, want edge with From Ref %s", edges, resource.NewReference("v1", "Pod", "default", "test-pod", ""))
	}
	if edges[0].To.Ref != resource.NewReference("v1", "Pod", "default", "test-pod", "") {
		t.Errorf("NewBuilder() = %v, want edge with To Ref %s", edges, resource.NewReference("v1", "Pod", "default", "test-pod", ""))
	}
}
