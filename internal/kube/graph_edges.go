package kube

import (
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// serviceNamespaceKey identifies the endpoint slices belonging to a service.
type serviceNamespaceKey struct {
	namespace string
	service   string
}

func (b *graphBuilder) addEdges(s *ResourceSnapshot) {
	podsByNamespace := indexPodsByNamespace(s.Pods)
	slicesByService := indexEndpointSlicesByService(s.EndpointSlices)

	b.addOwnerEdges()
	b.addServiceEdges(s, podsByNamespace, slicesByService)
	b.addPodEdges(s)
	b.addPVCEdges(s)
	b.addIngressEdges(s)
	b.addAutoscalerEdges(s)
	b.addPodSelectorEdges(s, podsByNamespace)
	b.addRBACEdges(s)
	b.addDynamicEdges(s)
}

// addAutoscalerEdges wires HorizontalPodAutoscalers to their scale targets.
func (b *graphBuilder) addAutoscalerEdges(s *ResourceSnapshot) {
	for i := range s.HorizontalPodAutoscalers {
		hpa := &s.HorizontalPodAutoscalers[i]
		hpaNode, ok := b.byUID[string(hpa.UID)]
		if !ok {
			continue
		}
		b.applyRelationships(hpaNode, TypedRelationships(hpa))
	}
}

// addPodSelectorEdges wires PodDisruptionBudgets and NetworkPolicies to the pods
// their label selectors match.
func (b *graphBuilder) addPodSelectorEdges(s *ResourceSnapshot, podsByNamespace map[string][]*corev1.Pod) {
	for i := range s.PodDisruptionBudgets {
		pdb := &s.PodDisruptionBudgets[i]
		node, ok := b.byUID[string(pdb.UID)]
		if !ok || pdb.Spec.Selector == nil {
			continue
		}
		b.linkSelectedPods(node, podsByNamespace[pdb.Namespace], pdb.Spec.Selector.MatchLabels, "pdb-selector")
	}

	for i := range s.NetworkPolicies {
		np := &s.NetworkPolicies[i]
		node, ok := b.byUID[string(np.UID)]
		if !ok {
			continue
		}
		b.linkSelectedPods(node, podsByNamespace[np.Namespace], np.Spec.PodSelector.MatchLabels, "netpol-selector")
	}
}

// linkSelectedPods adds protect edges from a policy node to every pod in the
// namespace matched by the selector. An empty selector matches all pods.
func (b *graphBuilder) linkSelectedPods(from graph.Node, pods []*corev1.Pod, selector map[string]string, reason string) {
	for j := range pods {
		pod := pods[j]
		if !selectorMatches(selector, pod.Labels) {
			continue
		}
		if podNode, found := b.byUID[string(pod.UID)]; found {
			b.addEdge(graph.EdgeProtects, from, podNode, reason)
		}
	}
}

// addRBACEdges wires RoleBindings and ClusterRoleBindings to the (Cluster)Role
// they reference.
func (b *graphBuilder) addRBACEdges(s *ResourceSnapshot) {
	for i := range s.RoleBindings {
		rb := &s.RoleBindings[i]
		if node, ok := b.byUID[string(rb.UID)]; ok {
			b.applyRelationships(node, TypedRelationships(rb))
		}
	}
	for i := range s.ClusterRoleBindings {
		crb := &s.ClusterRoleBindings[i]
		if node, ok := b.byUID[string(crb.UID)]; ok {
			b.applyRelationships(node, TypedRelationships(crb))
		}
	}
}

// addOwnerEdges wires ownership edges from owners to the objects they own. It
// operates uniformly over every node whose object exposes metadata, so owner
// references resolve regardless of whether the owner or owned object is a typed
// built-in or a dynamically fetched custom resource.
func (b *graphBuilder) addOwnerEdges() {
	for _, owned := range b.nodes {
		meta, ok := owned.Object.(metav1.Object)
		if !ok {
			continue
		}
		for _, owner := range OwnerReferenceTargets(meta) {
			if ownerNode, found := b.byUID[owner.UID]; found {
				b.addEdge(graph.EdgeOwns, ownerNode, owned, string(owner.Kind))
			}
		}
	}
}

// addServiceEdges wires services to the pods they select and to their endpoints.
func (b *graphBuilder) addServiceEdges(s *ResourceSnapshot, podsByNamespace map[string][]*corev1.Pod, slicesByService map[serviceNamespaceKey][]*discoveryv1.EndpointSlice) {
	for i := range s.Services {
		svc := &s.Services[i]
		svcNode, ok := b.byLogical[resource.NewReference(resource.ReferenceKindService, apiVersionCore, svc.Namespace, svc.Name, "").LogicalKey()]
		if !ok {
			continue
		}

		if len(svc.Spec.Selector) > 0 {
			for _, pod := range podsByNamespace[svc.Namespace] {
				if !selectorMatches(svc.Spec.Selector, pod.Labels) {
					continue
				}
				if podNode, found := b.byUID[string(pod.UID)]; found {
					b.addEdge(graph.EdgeSelectedBy, svcNode, podNode, "selector")
				}
			}
		}

		for _, slice := range slicesByService[serviceNamespaceKey{namespace: svc.Namespace, service: svc.Name}] {
			if sliceNode, found := b.byUID[string(slice.UID)]; found {
				b.addEdge(graph.EdgeReferences, sliceNode, svcNode, "endpointslice")
			}
		}
	}
}

func indexPodsByNamespace(pods []corev1.Pod) map[string][]*corev1.Pod {
	index := make(map[string][]*corev1.Pod)
	for i := range pods {
		pod := &pods[i]
		index[pod.Namespace] = append(index[pod.Namespace], pod)
	}

	return index
}

func indexEndpointSlicesByService(slices []discoveryv1.EndpointSlice) map[serviceNamespaceKey][]*discoveryv1.EndpointSlice {
	index := make(map[serviceNamespaceKey][]*discoveryv1.EndpointSlice)
	for i := range slices {
		slice := &slices[i]
		serviceName := slice.Labels[discoveryv1.LabelServiceName]
		if serviceName == "" {
			continue
		}
		key := serviceNamespaceKey{namespace: slice.Namespace, service: serviceName}
		index[key] = append(index[key], slice)
	}

	return index
}

// addPodEdges wires pods to the configmaps, secrets, PVCs and nodes they use.
func (b *graphBuilder) addPodEdges(s *ResourceSnapshot) {
	for i := range s.Pods {
		pod := &s.Pods[i]
		podNode, ok := b.byUID[string(pod.UID)]
		if !ok {
			continue
		}
		b.applyRelationships(podNode, TypedRelationships(pod))
	}
}

// addPVCEdges wires PVCs to their bound PV and storage class.
func (b *graphBuilder) addPVCEdges(s *ResourceSnapshot) {
	for i := range s.PersistentVolumeClaims {
		pvc := &s.PersistentVolumeClaims[i]
		pvcNode, ok := b.byUID[string(pvc.UID)]
		if !ok {
			continue
		}
		b.applyRelationships(pvcNode, TypedRelationships(pvc))
	}
}

// addIngressEdges wires ingresses to backend services and TLS secrets.
func (b *graphBuilder) addIngressEdges(s *ResourceSnapshot) {
	for i := range s.Ingresses {
		ing := &s.Ingresses[i]
		ingNode, ok := b.byUID[string(ing.UID)]
		if !ok {
			continue
		}
		b.applyRelationships(ingNode, TypedRelationships(ing))
	}
}

func (b *graphBuilder) addDynamicEdges(s *ResourceSnapshot) {
	for i := range s.Dynamic {
		entry := s.Dynamic[i].Resource
		obj := s.Dynamic[i].Object
		node, found := b.lookupLogical(entry.Kind, entry.APIVersion(), obj.GetNamespace(), obj.GetName())
		if !found {
			continue
		}
		b.applyRelationships(node, DynamicRelationships(entry, obj, b.resolveScope))
	}
}

// selectorMatches reports whether labels contain all of the selector entries.
func selectorMatches(selector, labels map[string]string) bool {
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}
