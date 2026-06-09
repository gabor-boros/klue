package graph

import (
	"sort"

	"github.com/gabor-boros/klue/pkg/resource"
)

// Graph is a graph of nodes and edges.
type Graph struct {
	nodes          map[string][]Node
	nodesByLogical map[string][]Node
	outboundEdges  map[string][]Edge
	inboundEdges   map[string][]Edge
}

// NewGraph creates a new graph.
func NewGraph() *Graph {
	return &Graph{
		nodes:          make(map[string][]Node),
		nodesByLogical: make(map[string][]Node),
		outboundEdges:  make(map[string][]Edge),
		inboundEdges:   make(map[string][]Edge),
	}
}

// AddNode adds a node to the graph.
func (g *Graph) AddNode(node Node) {
	g.nodes[node.Ref.Key()] = append(g.nodes[node.Ref.Key()], node)
	g.nodesByLogical[node.Ref.LogicalKey()] = append(g.nodesByLogical[node.Ref.LogicalKey()], node)
}

// AddEdge adds an edge to the graph.
func (g *Graph) AddEdge(edge Edge) {
	g.outboundEdges[edge.From.Ref.Key()] = append(g.outboundEdges[edge.From.Ref.Key()], edge)
	g.inboundEdges[edge.To.Ref.Key()] = append(g.inboundEdges[edge.To.Ref.Key()], edge)
}

// FindByRef returns the first node matching the given reference. It first
// matches on the reference key (UID-based when a UID is present) and falls back
// to a logical-key match so callers can look up a node without knowing its UID.
func (g *Graph) FindByRef(ref resource.Reference) (Node, bool) {
	if nodes, ok := g.nodes[ref.Key()]; ok && len(nodes) > 0 {
		return nodes[0], true
	}

	if nodes, ok := g.nodesByLogical[ref.LogicalKey()]; ok && len(nodes) > 0 {
		return nodes[0], true
	}

	return Node{}, false
}

// GetOutboundEdges returns the outbound edges of a given node.
func (g *Graph) GetOutboundEdges(node Node) []Edge {
	return g.outboundEdges[node.Ref.Key()]
}

// GetOutboundNodes returns the outbound nodes of a given node.
func (g *Graph) GetOutboundNodes(node Node) []Node {
	edges := g.GetOutboundEdges(node)
	outboundNodes := make([]Node, 0, len(edges))

	for _, edge := range edges {
		outboundNodes = append(outboundNodes, edge.To)
	}

	return outboundNodes
}

// GetInboundEdges returns the inbound edges of a given node.
func (g *Graph) GetInboundEdges(node Node) []Edge {
	return g.inboundEdges[node.Ref.Key()]
}

// GetInboundNodes returns the inbound nodes of a given node.
func (g *Graph) GetInboundNodes(node Node) []Node {
	edges := g.GetInboundEdges(node)
	inboundNodes := make([]Node, 0, len(edges))

	for _, edge := range edges {
		inboundNodes = append(inboundNodes, edge.From)
	}

	return inboundNodes
}

// GetRelatedNodes returns the related nodes of a given node.
func (g *Graph) GetRelatedNodes(node Node) []Node {
	return append(g.GetOutboundNodes(node), g.GetInboundNodes(node)...)
}

// GetNodes returns all nodes in the graph, ordered by their reference key for
// deterministic output.
func (g *Graph) GetNodes() []Node {
	keys := make([]string, 0, len(g.nodes))
	for key := range g.nodes {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	nodes := make([]Node, 0, len(g.nodes))
	for _, key := range keys {
		nodes = append(nodes, g.nodes[key]...)
	}

	return nodes
}

// GetEdges returns all edges in the graph, ordered by the source node reference
// key for deterministic output. Each edge is returned once; the inbound index
// is only a reverse lookup of the same edges.
func (g *Graph) GetEdges() []Edge {
	keys := make([]string, 0, len(g.outboundEdges))
	for key := range g.outboundEdges {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	edges := make([]Edge, 0, len(g.outboundEdges))
	for _, key := range keys {
		edges = append(edges, g.outboundEdges[key]...)
	}

	return edges
}
