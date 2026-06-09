package kube

import (
	"fmt"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// API group/version strings used when building references.
const (
	apiVersionCore         = "v1"
	apiVersionApps         = "apps/v1"
	apiVersionBatch        = "batch/v1"
	apiVersionDiscovery    = "discovery.k8s.io/v1"
	apiVersionNetworking   = "networking.k8s.io/v1"
	apiVersionStorage      = "storage.k8s.io/v1"
	apiVersionAutoscaling  = "autoscaling/v2"
	apiVersionPolicy       = "policy/v1"
	apiVersionRBAC         = "rbac.authorization.k8s.io/v1"
	apiVersionCertificates = "certificates.k8s.io/v1"
	apiVersionCoordination = "coordination.k8s.io/v1"
)

// graphBuilder accumulates nodes and edges while mapping a snapshot, keeping
// lookup tables so edges can resolve their endpoints.
type graphBuilder struct {
	builder       *graph.Builder
	byUID         map[string]graph.Node
	byLogical     map[string]graph.Node
	nodes         []graph.Node
	edgeKeys      map[string]struct{}
	resourceScope map[string]bool
}

// BuildGraph maps the snapshot into a resource graph of nodes and edges.
func (s *ResourceSnapshot) BuildGraph() *graph.Graph {
	b := &graphBuilder{
		builder:       graph.NewBuilder(),
		byUID:         make(map[string]graph.Node),
		byLogical:     make(map[string]graph.Node),
		edgeKeys:      make(map[string]struct{}),
		resourceScope: make(map[string]bool),
	}

	b.indexResourceScope(s)
	b.addNodes(s)
	b.addEdges(s)

	return b.builder.Build()
}

// addNode creates a node and registers it in the lookup tables.
func (b *graphBuilder) addNode(kind resource.Kind, apiVersion, namespace, name, uid string, obj any, labels map[string]string, status resource.Status) {
	ref := resource.NewReference(kind, apiVersion, namespace, name, uid)
	node := graph.Node{Ref: ref, Object: obj, Labels: labels, Status: status}

	b.builder.AddNode(node)
	b.byLogical[ref.LogicalKey()] = node
	b.nodes = append(b.nodes, node)
	if uid != "" {
		b.byUID[uid] = node
	}
}

// lookupLogical resolves a name-based reference to a previously added node.
func (b *graphBuilder) lookupLogical(kind resource.Kind, apiVersion, namespace, name string) (graph.Node, bool) {
	ref := resource.NewReference(kind, apiVersion, namespace, name, "")
	node, ok := b.byLogical[ref.LogicalKey()]
	return node, ok
}

// addEdge records an edge between two nodes.
func (b *graphBuilder) addEdge(kind graph.EdgeKind, from, to graph.Node, reason string) {
	b.addEdgeWithData(kind, from, to, reason, graph.EdgeData{})
}

func (b *graphBuilder) addEdgeWithData(kind graph.EdgeKind, from, to graph.Node, reason string, data graph.EdgeData) {
	key := fmt.Sprintf("%s|%s|%s|%s|%s", from.Ref.Key(), to.Ref.LogicalKey(), kind, reason, data.Path)
	if _, exists := b.edgeKeys[key]; exists {
		return
	}
	b.edgeKeys[key] = struct{}{}
	b.builder.AddEdge(graph.Edge{Kind: kind, From: from, To: to, Reason: reason, Data: data})
}

func (b *graphBuilder) addRelationship(from graph.Node, relationship Relationship) {
	targetRef := relationship.Target
	targetRef.Namespace = namespaceForTarget(targetRef.Kind, targetRef.APIVersion, from.Ref.Namespace, targetRef.Namespace, b.resolveScope)

	target, found := b.lookupLogical(targetRef.Kind, targetRef.APIVersion, targetRef.Namespace, targetRef.Name)
	if !found {
		target = b.ensurePlaceholder(targetRef, relationship.Reason)
	}

	b.addEdgeWithData(relationship.EdgeKind, from, target, relationship.Reason, relationshipEdgeData(relationship))
}

func (b *graphBuilder) applyRelationships(from graph.Node, relationships []Relationship) {
	for i := range relationships {
		b.addRelationship(from, relationships[i])
	}
}

func (b *graphBuilder) ensurePlaceholder(ref resource.Reference, reason string) graph.Node {
	logicalKey := ref.LogicalKey()
	if existing, found := b.byLogical[logicalKey]; found {
		return existing
	}

	placeholder := graph.Node{
		Ref:         resource.NewReference(ref.Kind, ref.APIVersion, ref.Namespace, ref.Name, ""),
		Status:      resource.StatusMissing,
		Placeholder: &graph.Placeholder{Reason: reason},
	}

	b.builder.AddNode(placeholder)
	b.byLogical[placeholder.Ref.LogicalKey()] = placeholder
	b.nodes = append(b.nodes, placeholder)
	return placeholder
}

func (b *graphBuilder) indexResourceScope(s *ResourceSnapshot) {
	for _, entry := range BuiltinCatalog() {
		b.resourceScope[scopeKey(entry.APIVersion(), entry.Kind)] = entry.Namespaced
	}
	for i := range s.Dynamic {
		entry := s.Dynamic[i].Resource
		b.resourceScope[scopeKey(entry.APIVersion(), entry.Kind)] = entry.Namespaced
	}
}

func (b *graphBuilder) resolveScope(apiVersion string, kind resource.Kind) (bool, bool) {
	namespaced, found := b.resourceScope[scopeKey(apiVersion, kind)]
	return namespaced, found
}

func scopeKey(apiVersion string, kind resource.Kind) string {
	return fmt.Sprintf("%s/%s", apiVersion, kind)
}
