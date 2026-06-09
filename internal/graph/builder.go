package graph

// Builder is a builder for the graph.
type Builder struct {
	graph *Graph
}

// NewBuilder creates a new builder.
func NewBuilder() *Builder {
	return &Builder{graph: NewGraph()}
}

// Build builds the graph.
func (b *Builder) Build() *Graph {
	return b.graph
}

// AddNode adds a node to the graph.
func (b *Builder) AddNode(node Node) {
	b.graph.AddNode(node)
}

// AddEdge adds an edge to the graph.
func (b *Builder) AddEdge(edge Edge) {
	b.graph.AddEdge(edge)
}
