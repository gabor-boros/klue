package graph

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// As returns the node's underlying object as type T. The second return value
// reports whether the object was of that type, mirroring the comma-ok form of a
// type assertion so rules can bail out cleanly on a mismatch.
func As[T any](node *Node) (T, bool) {
	typed, ok := node.Object.(T)
	return typed, ok
}

// Meta returns the node's object as a metav1.Object. Both typed API objects and
// unstructured objects satisfy this interface, so it works for any real node.
func Meta(node *Node) (metav1.Object, bool) {
	meta, ok := node.Object.(metav1.Object)
	return meta, ok
}
