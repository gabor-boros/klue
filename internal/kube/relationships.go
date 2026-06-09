package kube

import (
	"sort"
	"strings"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// RelationshipRole describes the semantic role of a relationship.
type RelationshipRole string

const (
	RelationshipRoleReferences RelationshipRole = "references"
	RelationshipRoleUses       RelationshipRole = "uses"
	RelationshipRoleProduces   RelationshipRole = "produces"
)

// Relationship represents a typed reference discovered from a resource object.
type Relationship struct {
	EdgeKind graph.EdgeKind
	Target   resource.Reference
	Reason   string
	Path     string
	Role     RelationshipRole
}

// RelationshipScopeResolver reports whether a resource kind is namespace scoped.
// The bool return value indicates whether the scope is known.
type RelationshipScopeResolver func(apiVersion string, kind resource.Kind) (namespaced bool, known bool)

// RelationshipHasProducer reports whether an unresolved target has at least one
// inbound producer edge.
func RelationshipHasProducer(g *graph.Graph, node graph.Node) bool {
	for _, edge := range g.GetInboundEdges(node) {
		if edge.Kind == graph.EdgeProduces {
			return true
		}
	}
	return false
}

// TypedRelationships extracts reusable typed-object relationships that can be
// shared by graph building and diagnostics.
func TypedRelationships(obj any) []Relationship {
	switch typed := obj.(type) {
	case *corev1.Pod:
		return podRelationships(typed)
	case *networkingv1.Ingress:
		return ingressRelationships(typed)
	case *corev1.PersistentVolumeClaim:
		return pvcRelationships(typed)
	case *autoscalingv2.HorizontalPodAutoscaler:
		return hpaRelationships(typed)
	case *rbacv1.RoleBinding:
		return roleBindingRelationships(&typed.RoleRef, typed.Namespace)
	case *rbacv1.ClusterRoleBinding:
		return roleBindingRelationships(&typed.RoleRef, "")
	default:
		return nil
	}
}

// OwnerReferenceTargets returns owner references as resource references.
func OwnerReferenceTargets(meta metav1.Object) []resource.Reference {
	owners := meta.GetOwnerReferences()
	if len(owners) == 0 {
		return nil
	}

	rels := make([]resource.Reference, 0, len(owners))
	for _, owner := range owners {
		ns := namespaceForTarget(resource.Kind(owner.Kind), owner.APIVersion, meta.GetNamespace(), "", nil)
		rels = append(rels, resource.NewReference(resource.Kind(owner.Kind), owner.APIVersion, ns, owner.Name, string(owner.UID)))
	}
	sort.Slice(rels, func(i, j int) bool {
		return rels[i].LogicalKey() < rels[j].LogicalKey()
	})
	return rels
}

func podRelationships(pod *corev1.Pod) []Relationship {
	configMaps, secrets := PodConfigRefs(pod)
	relationships := make([]Relationship, 0, len(configMaps)+len(secrets)+len(pod.Spec.Volumes)+1)

	for _, name := range configMaps {
		relationships = append(relationships, Relationship{
			EdgeKind: graph.EdgeUsesConfigMap,
			Target:   resource.NewReference(resource.ReferenceKindConfigMap, apiVersionCore, pod.Namespace, name, ""),
			Reason:   "configmap",
			Path:     "spec",
			Role:     RelationshipRoleUses,
		})
	}
	for _, name := range secrets {
		relationships = append(relationships, Relationship{
			EdgeKind: graph.EdgeUsesSecret,
			Target:   resource.NewReference(resource.ReferenceKindSecret, apiVersionCore, pod.Namespace, name, ""),
			Reason:   "secret",
			Path:     "spec",
			Role:     RelationshipRoleUses,
		})
	}

	for _, vol := range pod.Spec.Volumes {
		if vol.PersistentVolumeClaim == nil || vol.PersistentVolumeClaim.ClaimName == "" {
			continue
		}
		relationships = append(relationships, Relationship{
			EdgeKind: graph.EdgeMounts,
			Target:   resource.NewReference(resource.ReferenceKindPersistentVolumeClaim, apiVersionCore, pod.Namespace, vol.PersistentVolumeClaim.ClaimName, ""),
			Reason:   "pvc",
			Path:     "spec.volumes[].persistentVolumeClaim.claimName",
			Role:     RelationshipRoleReferences,
		})
	}

	if pod.Spec.NodeName != "" {
		relationships = append(relationships, Relationship{
			EdgeKind: graph.EdgeScheduledOn,
			Target:   resource.NewReference(resource.ReferenceKindNode, apiVersionCore, "", pod.Spec.NodeName, ""),
			Reason:   "node",
			Path:     "spec.nodeName",
			Role:     RelationshipRoleReferences,
		})
	}

	sortRelationships(relationships)
	return relationships
}

func ingressRelationships(ing *networkingv1.Ingress) []Relationship {
	services := IngressBackendServiceNames(ing)
	secrets := IngressTLSSecretNames(ing)
	relationships := make([]Relationship, 0, len(services)+len(secrets))

	for _, name := range services {
		relationships = append(relationships, Relationship{
			EdgeKind: graph.EdgeReferences,
			Target:   resource.NewReference(resource.ReferenceKindService, apiVersionCore, ing.Namespace, name, ""),
			Reason:   "backend",
			Path:     "spec.rules[].http.paths[].backend.service.name",
			Role:     RelationshipRoleReferences,
		})
	}
	for _, name := range secrets {
		relationships = append(relationships, Relationship{
			EdgeKind: graph.EdgeUsesSecret,
			Target:   resource.NewReference(resource.ReferenceKindSecret, apiVersionCore, ing.Namespace, name, ""),
			Reason:   "tls",
			Path:     "spec.tls[].secretName",
			Role:     RelationshipRoleUses,
		})
	}

	sortRelationships(relationships)
	return relationships
}

func pvcRelationships(pvc *corev1.PersistentVolumeClaim) []Relationship {
	relationships := make([]Relationship, 0, 2)
	if pvc.Spec.VolumeName != "" {
		relationships = append(relationships, Relationship{
			EdgeKind: graph.EdgeReferences,
			Target:   resource.NewReference(resource.ReferenceKindPersistentVolume, apiVersionCore, "", pvc.Spec.VolumeName, ""),
			Reason:   "volume",
			Path:     "spec.volumeName",
			Role:     RelationshipRoleReferences,
		})
	}
	if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName != "" {
		relationships = append(relationships, Relationship{
			EdgeKind: graph.EdgeUsesStorageClass,
			Target:   resource.NewReference(resource.ReferenceKindStorageClass, apiVersionStorage, "", *pvc.Spec.StorageClassName, ""),
			Reason:   "storageclass",
			Path:     "spec.storageClassName",
			Role:     RelationshipRoleUses,
		})
	}
	sortRelationships(relationships)
	return relationships
}

func hpaRelationships(hpa *autoscalingv2.HorizontalPodAutoscaler) []Relationship {
	target := hpa.Spec.ScaleTargetRef
	if target.Kind == "" || target.Name == "" {
		return nil
	}
	return []Relationship{{
		EdgeKind: graph.EdgeScaleTarget,
		Target:   resource.NewReference(resource.Kind(target.Kind), target.APIVersion, hpa.Namespace, target.Name, ""),
		Reason:   "scaleTargetRef",
		Path:     "spec.scaleTargetRef",
		Role:     RelationshipRoleReferences,
	}}
}

func roleBindingRelationships(roleRef *rbacv1.RoleRef, namespace string) []Relationship {
	if roleRef == nil || roleRef.Kind == "" || roleRef.Name == "" {
		return nil
	}

	targetNamespace := namespace
	if resource.Kind(roleRef.Kind) == resource.ReferenceKindClusterRole {
		targetNamespace = ""
	}

	return []Relationship{{
		EdgeKind: graph.EdgeRoleRef,
		Target:   resource.NewReference(resource.Kind(roleRef.Kind), apiVersionRBAC, targetNamespace, roleRef.Name, ""),
		Reason:   "roleRef",
		Path:     "roleRef",
		Role:     RelationshipRoleReferences,
	}}
}

// PodConfigRefs returns unique ConfigMap and Secret names referenced by a Pod.
func PodConfigRefs(pod *corev1.Pod) (configMaps, secrets []string) {
	configMapSet := make(map[string]struct{})
	secretSet := make(map[string]struct{})

	containers := make([]corev1.Container, 0, len(pod.Spec.Containers)+len(pod.Spec.InitContainers))
	containers = append(containers, pod.Spec.InitContainers...)
	containers = append(containers, pod.Spec.Containers...)

	for i := range containers {
		c := &containers[i]
		for _, env := range c.Env {
			if env.ValueFrom == nil {
				continue
			}
			if ref := env.ValueFrom.ConfigMapKeyRef; ref != nil && ref.Name != "" {
				configMapSet[ref.Name] = struct{}{}
			}
			if ref := env.ValueFrom.SecretKeyRef; ref != nil && ref.Name != "" {
				secretSet[ref.Name] = struct{}{}
			}
		}
		for _, envFrom := range c.EnvFrom {
			if ref := envFrom.ConfigMapRef; ref != nil && ref.Name != "" {
				configMapSet[ref.Name] = struct{}{}
			}
			if ref := envFrom.SecretRef; ref != nil && ref.Name != "" {
				secretSet[ref.Name] = struct{}{}
			}
		}
	}

	for _, vol := range pod.Spec.Volumes {
		if vol.ConfigMap != nil && vol.ConfigMap.Name != "" {
			configMapSet[vol.ConfigMap.Name] = struct{}{}
		}
		if vol.Secret != nil && vol.Secret.SecretName != "" {
			secretSet[vol.Secret.SecretName] = struct{}{}
		}
	}

	for _, pull := range pod.Spec.ImagePullSecrets {
		if pull.Name != "" {
			secretSet[pull.Name] = struct{}{}
		}
	}

	return sortedKeys(configMapSet), sortedKeys(secretSet)
}

// IngressBackendServiceNames returns unique backend service names for an Ingress.
func IngressBackendServiceNames(ing *networkingv1.Ingress) []string {
	set := make(map[string]struct{})

	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil && ing.Spec.DefaultBackend.Service.Name != "" {
		set[ing.Spec.DefaultBackend.Service.Name] = struct{}{}
	}

	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service != nil && path.Backend.Service.Name != "" {
				set[path.Backend.Service.Name] = struct{}{}
			}
		}
	}

	return sortedKeys(set)
}

// IngressTLSSecretNames returns unique TLS secret names for an Ingress.
func IngressTLSSecretNames(ing *networkingv1.Ingress) []string {
	set := make(map[string]struct{})
	for _, tls := range ing.Spec.TLS {
		if tls.SecretName != "" {
			set[tls.SecretName] = struct{}{}
		}
	}
	return sortedKeys(set)
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func sortRelationships(relationships []Relationship) {
	sort.Slice(relationships, func(i, j int) bool {
		if relationships[i].Target.LogicalKey() != relationships[j].Target.LogicalKey() {
			return relationships[i].Target.LogicalKey() < relationships[j].Target.LogicalKey()
		}
		if relationships[i].EdgeKind != relationships[j].EdgeKind {
			return relationships[i].EdgeKind < relationships[j].EdgeKind
		}
		if relationships[i].Path != relationships[j].Path {
			return relationships[i].Path < relationships[j].Path
		}
		return relationships[i].Reason < relationships[j].Reason
	})
}

func namespaceForTarget(kind resource.Kind, apiVersion, sourceNamespace, explicitNamespace string, resolver RelationshipScopeResolver) string {
	if explicitNamespace != "" {
		return explicitNamespace
	}

	if resolver != nil {
		if namespaced, known := resolver(apiVersion, kind); known && !namespaced {
			return ""
		}
	}

	if entry, found := LookupKind(kind); found && !entry.Namespaced {
		return ""
	}

	if strings.EqualFold(string(kind), "ClusterIssuer") {
		return ""
	}

	return sourceNamespace
}

func relationshipEdgeData(rel Relationship) graph.EdgeData {
	return graph.EdgeData{Path: rel.Path, Role: string(rel.Role)}
}
