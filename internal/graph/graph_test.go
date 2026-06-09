package graph_test

import (
	"testing"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestNewGraph(t *testing.T) {
	t.Parallel()

	kubeGraph := graph.NewGraph()

	if kubeGraph == nil {
		t.Errorf("NewGraph() = %v, want non-nil", kubeGraph)
	}
}

func TestGraph_GetOutboundNodes(t *testing.T) {
	t.Parallel()

	fromNode := graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
	}

	toNode := graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod-2", ""),
	}

	edge := graph.Edge{
		Kind: graph.EdgeOwns,
		From: fromNode,
		To:   toNode,
	}

	kubeGraph := graph.NewGraph()
	kubeGraph.AddNode(fromNode)
	kubeGraph.AddNode(toNode)
	kubeGraph.AddEdge(edge)

	outboundNodes := kubeGraph.GetOutboundNodes(fromNode)
	if len(outboundNodes) != 1 {
		t.Errorf("GetOutboundNodes() = %v, want 1 node", outboundNodes)
	}
	if outboundNodes[0].Ref != toNode.Ref {
		t.Errorf("GetOutboundNodes() = %v, want node with Ref %s", outboundNodes, toNode.Ref)
	}
}

func TestGraph_GetInboundNodes(t *testing.T) {
	t.Parallel()

	fromNode := graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
	}

	toNode := graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod-2", ""),
	}

	edge := graph.Edge{
		Kind: graph.EdgeOwns,
		From: fromNode,
		To:   toNode,
	}

	kubeGraph := graph.NewGraph()
	kubeGraph.AddNode(fromNode)
	kubeGraph.AddNode(toNode)
	kubeGraph.AddEdge(edge)

	inboundNodes := kubeGraph.GetInboundNodes(toNode)
	if len(inboundNodes) != 1 {
		t.Errorf("GetInboundNodes() = %v, want 1 node", inboundNodes)
	}
	if inboundNodes[0].Ref != fromNode.Ref {
		t.Errorf("GetInboundNodes() = %v, want node with Ref %s", inboundNodes, fromNode.Ref)
	}
}

func TestGraph_GetRelatedNodes(t *testing.T) {
	t.Parallel()

	fromNode := graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
	}

	toNode := graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod-2", ""),
	}

	edge := graph.Edge{
		Kind: graph.EdgeOwns,
		From: fromNode,
		To:   toNode,
	}

	kubeGraph := graph.NewGraph()
	kubeGraph.AddNode(fromNode)
	kubeGraph.AddNode(toNode)
	kubeGraph.AddEdge(edge)

	relatedNodes := kubeGraph.GetRelatedNodes(fromNode)
	if len(relatedNodes) != 1 {
		t.Errorf("GetRelatedNodes() = %v, want 1 node", relatedNodes)
	}
	if relatedNodes[0].Ref != toNode.Ref {
		t.Errorf("GetRelatedNodes() = %v, want node with Ref %s", relatedNodes, toNode.Ref)
	}
}

func TestGraph_GetNodes(t *testing.T) {
	t.Parallel()

	fromNode := graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
	}

	toNode := graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod-2", ""),
	}

	edge := graph.Edge{
		Kind: graph.EdgeOwns,
		From: fromNode,
		To:   toNode,
	}

	kubeGraph := graph.NewGraph()
	kubeGraph.AddNode(fromNode)
	kubeGraph.AddNode(toNode)
	kubeGraph.AddEdge(edge)

	nodes := kubeGraph.GetNodes()
	if len(nodes) != 2 {
		t.Errorf("GetNodes() = %v, want 2 nodes", nodes)
	}
	if nodes[0].Ref != fromNode.Ref {
		t.Errorf("GetNodes() = %v, want node with Ref %s", nodes, fromNode.Ref)
	}
	if nodes[1].Ref != toNode.Ref {
		t.Errorf("GetNodes() = %v, want node with Ref %s", nodes, toNode.Ref)
	}
}

func TestGraph_GetEdges(t *testing.T) {
	t.Parallel()

	fromNode := graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod", ""),
	}

	toNode := graph.Node{
		Ref: resource.NewReference("v1", "Pod", "default", "test-pod-2", ""),
	}

	edge := graph.Edge{
		Kind: graph.EdgeOwns,
		From: fromNode,
		To:   toNode,
	}

	kubeGraph := graph.NewGraph()
	kubeGraph.AddNode(fromNode)
	kubeGraph.AddNode(toNode)
	kubeGraph.AddEdge(edge)

	edges := kubeGraph.GetEdges()
	if len(edges) != 1 {
		t.Errorf("GetEdges() = %v, want 1 edge", edges)
	}
	if edges[0].Kind != graph.EdgeOwns {
		t.Errorf("GetEdges() = %v, want edge with Kind %s", edges, graph.EdgeOwns)
	}
	if edges[0].From.Ref != fromNode.Ref {
		t.Errorf("GetEdges() = %v, want edge with From Ref %s", edges, fromNode.Ref)
	}
	if edges[0].To.Ref != toNode.Ref {
		t.Errorf("GetEdges() = %v, want edge with To Ref %s", edges, toNode.Ref)
	}
}
