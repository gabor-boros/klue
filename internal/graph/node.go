package graph

import (
	"github.com/gabor-boros/klue/pkg/resource"
)

// Node is a node in the graph. Object holds the underlying Kubernetes object
// (a typed API object or an *unstructured.Unstructured) and is nil for
// synthetic placeholder nodes created for unresolved references.
type Node struct {
	Ref resource.Reference

	Object      any
	Status      resource.Status
	Labels      map[string]string
	Placeholder *Placeholder
}

// Placeholder describes a synthetic node that stands in for a referenced object
// that was not found in the cluster snapshot.
type Placeholder struct {
	// Reason is a short description of how the missing target was referenced.
	Reason string
}

// IsPlaceholder reports whether the node is a synthetic stand-in for an
// unresolved reference target rather than a real object from the snapshot.
func (n Node) IsPlaceholder() bool {
	return n.Object == nil && n.Placeholder != nil
}
