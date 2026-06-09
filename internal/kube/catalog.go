package kube

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/gabor-boros/klue/pkg/resource"
)

// APIResource describes a single Kubernetes API resource, whether a built-in
// type or a custom resource backed by a CustomResourceDefinition. The same
// descriptor drives typed/dynamic fetching, graph construction and CLI command
// registration so a resource only needs to be declared (or discovered) once.
type APIResource struct {
	// Group is the API group ("" for the core group).
	Group string

	// Version is the preferred served version.
	Version string

	// Resource is the lowercase plural resource name (e.g. "deployments").
	Resource string

	// Kind is the resource Kind (e.g. "Deployment").
	Kind resource.Kind

	// Namespaced reports whether the resource is namespace-scoped.
	Namespaced bool

	// Typed reports whether the typed fetcher already retrieves this resource.
	// When false, the resource is fetched dynamically as unstructured data.
	// Custom resources are always dynamic.
	Typed bool

	// Aliases are additional CLI command tokens (plural form, kubectl short
	// names) that resolve to this resource.
	Aliases []string

	// Custom reports whether the resource is a custom resource (backed by a
	// CustomResourceDefinition) discovered at runtime rather than a built-in.
	Custom bool
}

// APIVersion returns the apiVersion string ("v1" for core, "group/version"
// otherwise).
func (b APIResource) APIVersion() string {
	if b.Group == "" {
		return b.Version
	}
	return b.Group + "/" + b.Version
}

// GroupVersionResource returns the GVR used by the dynamic client.
func (b APIResource) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: b.Group, Version: b.Version, Resource: b.Resource}
}

// CommandToken returns the primary CLI command token for the resource (the
// lowercase Kind).
func (b APIResource) CommandToken() string {
	return strings.ToLower(string(b.Kind))
}

// builtinCatalog is the canonical list of Kubernetes built-in resources klue
// understands. CRD instance resources are intentionally excluded: only
// Kubernetes-owned API groups appear here.
var builtinCatalog = []APIResource{
	// Core group (v1).
	{Group: "", Version: "v1", Resource: "pods", Kind: resource.ReferenceKindPod, Namespaced: true, Typed: true, Aliases: []string{"pods", "po"}},
	{Group: "", Version: "v1", Resource: "services", Kind: resource.ReferenceKindService, Namespaced: true, Typed: true, Aliases: []string{"services", "svc"}},
	{Group: "", Version: "v1", Resource: "configmaps", Kind: resource.ReferenceKindConfigMap, Namespaced: true, Typed: true, Aliases: []string{"configmaps", "cm"}},
	{Group: "", Version: "v1", Resource: "secrets", Kind: resource.ReferenceKindSecret, Namespaced: true, Typed: true, Aliases: []string{"secrets"}},
	{Group: "", Version: "v1", Resource: "persistentvolumeclaims", Kind: resource.ReferenceKindPersistentVolumeClaim, Namespaced: true, Typed: true, Aliases: []string{"persistentvolumeclaims", "pvc"}},
	{Group: "", Version: "v1", Resource: "persistentvolumes", Kind: resource.ReferenceKindPersistentVolume, Namespaced: false, Typed: true, Aliases: []string{"persistentvolumes", "pv"}},
	{Group: "", Version: "v1", Resource: "nodes", Kind: resource.ReferenceKindNode, Namespaced: false, Typed: true, Aliases: []string{"nodes", "no"}},
	{Group: "", Version: "v1", Resource: "serviceaccounts", Kind: resource.ReferenceKindServiceAccount, Namespaced: true, Typed: false, Aliases: []string{"serviceaccounts", "sa"}},
	{Group: "", Version: "v1", Resource: "replicationcontrollers", Kind: resource.ReferenceKindReplicationController, Namespaced: true, Typed: false, Aliases: []string{"replicationcontrollers", "rc"}},
	{Group: "", Version: "v1", Resource: "resourcequotas", Kind: resource.ReferenceKindResourceQuota, Namespaced: true, Typed: false, Aliases: []string{"resourcequotas", "quota"}},
	{Group: "", Version: "v1", Resource: "limitranges", Kind: resource.ReferenceKindLimitRange, Namespaced: true, Typed: false, Aliases: []string{"limitranges", "limits"}},
	{Group: "", Version: "v1", Resource: "namespaces", Kind: resource.ReferenceKindNamespace, Namespaced: false, Typed: false, Aliases: []string{"namespaces", "ns"}},
	{Group: "", Version: "v1", Resource: "podtemplates", Kind: resource.ReferenceKindPodTemplate, Namespaced: true, Typed: false, Aliases: []string{"podtemplates"}},

	// apps/v1.
	{Group: "apps", Version: "v1", Resource: "deployments", Kind: resource.ReferenceKindDeployment, Namespaced: true, Typed: true, Aliases: []string{"deployments", "deploy"}},
	{Group: "apps", Version: "v1", Resource: "replicasets", Kind: resource.ReferenceKindReplicaSet, Namespaced: true, Typed: true, Aliases: []string{"replicasets", "rs"}},
	{Group: "apps", Version: "v1", Resource: "statefulsets", Kind: resource.ReferenceKindStatefulSet, Namespaced: true, Typed: true, Aliases: []string{"statefulsets", "sts"}},
	{Group: "apps", Version: "v1", Resource: "daemonsets", Kind: resource.ReferenceKindDaemonSet, Namespaced: true, Typed: true, Aliases: []string{"daemonsets", "ds"}},
	{Group: "apps", Version: "v1", Resource: "controllerrevisions", Kind: resource.ReferenceKindControllerRevision, Namespaced: true, Typed: false, Aliases: []string{"controllerrevisions"}},

	// batch/v1.
	{Group: "batch", Version: "v1", Resource: "jobs", Kind: resource.ReferenceKindJob, Namespaced: true, Typed: true, Aliases: []string{"jobs"}},
	{Group: "batch", Version: "v1", Resource: "cronjobs", Kind: resource.ReferenceKindCronJob, Namespaced: true, Typed: true, Aliases: []string{"cronjobs", "cj"}},

	// discovery.k8s.io/v1.
	{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices", Kind: resource.ReferenceKindEndpointSlice, Namespaced: true, Typed: true, Aliases: []string{"endpointslices"}},

	// networking.k8s.io/v1.
	{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses", Kind: resource.ReferenceKindIngress, Namespaced: true, Typed: true, Aliases: []string{"ingresses", "ing"}},
	{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies", Kind: resource.ReferenceKindNetworkPolicy, Namespaced: true, Typed: true, Aliases: []string{"networkpolicies", "netpol"}},
	{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses", Kind: resource.ReferenceKindIngressClass, Namespaced: false, Typed: false, Aliases: []string{"ingressclasses"}},

	// storage.k8s.io/v1.
	{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses", Kind: resource.ReferenceKindStorageClass, Namespaced: false, Typed: true, Aliases: []string{"storageclasses", "sc"}},

	// autoscaling/v2.
	{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers", Kind: resource.ReferenceKindHorizontalPodAutoscaler, Namespaced: true, Typed: true, Aliases: []string{"horizontalpodautoscalers", "hpa"}},

	// policy/v1.
	{Group: "policy", Version: "v1", Resource: "poddisruptionbudgets", Kind: resource.ReferenceKindPodDisruptionBudget, Namespaced: true, Typed: true, Aliases: []string{"poddisruptionbudgets", "pdb"}},

	// rbac.authorization.k8s.io/v1.
	{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "roles", Kind: resource.ReferenceKindRole, Namespaced: true, Typed: true, Aliases: []string{"roles"}},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings", Kind: resource.ReferenceKindRoleBinding, Namespaced: true, Typed: true, Aliases: []string{"rolebindings"}},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles", Kind: resource.ReferenceKindClusterRole, Namespaced: false, Typed: true, Aliases: []string{"clusterroles"}},
	{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings", Kind: resource.ReferenceKindClusterRoleBinding, Namespaced: false, Typed: true, Aliases: []string{"clusterrolebindings"}},

	// certificates.k8s.io/v1.
	{Group: "certificates.k8s.io", Version: "v1", Resource: "certificatesigningrequests", Kind: resource.ReferenceKindCertificateSigningRequest, Namespaced: false, Typed: true, Aliases: []string{"certificatesigningrequests", "csr"}},

	// coordination.k8s.io/v1.
	{Group: "coordination.k8s.io", Version: "v1", Resource: "leases", Kind: resource.ReferenceKindLease, Namespaced: true, Typed: true, Aliases: []string{"leases"}},

	// scheduling.k8s.io/v1.
	{Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses", Kind: resource.ReferenceKindPriorityClass, Namespaced: false, Typed: false, Aliases: []string{"priorityclasses", "pc"}},

	// node.k8s.io/v1.
	{Group: "node.k8s.io", Version: "v1", Resource: "runtimeclasses", Kind: resource.ReferenceKindRuntimeClass, Namespaced: false, Typed: false, Aliases: []string{"runtimeclasses"}},

	// apiregistration.k8s.io/v1.
	{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices", Kind: resource.ReferenceKindAPIService, Namespaced: false, Typed: false, Aliases: []string{"apiservices"}},

	// admissionregistration.k8s.io/v1.
	{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations", Kind: resource.ReferenceKindMutatingWebhookConfig, Namespaced: false, Typed: false, Aliases: []string{"mutatingwebhookconfigurations"}},
	{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations", Kind: resource.ReferenceKindValidatingWebhookConfig, Namespaced: false, Typed: false, Aliases: []string{"validatingwebhookconfigurations"}},
}

// builtinGroups is the set of API groups owned by Kubernetes. It is derived
// from the catalog and used to distinguish built-in resources from CRDs.
var builtinGroups = func() map[string]struct{} {
	groups := make(map[string]struct{})
	for _, entry := range builtinCatalog {
		groups[entry.Group] = struct{}{}
	}
	return groups
}()

// BuiltinCatalog returns a copy of the built-in resource catalog.
func BuiltinCatalog() []APIResource {
	out := make([]APIResource, len(builtinCatalog))
	copy(out, builtinCatalog)
	return out
}

// IsBuiltinGroup reports whether the given API group is a Kubernetes built-in
// group present in the catalog. CRD groups return false.
func IsBuiltinGroup(group string) bool {
	_, ok := builtinGroups[group]
	return ok
}

// IsSubresource reports whether the given discovery resource name refers to a
// subresource (e.g. "pods/status", "deployments/scale").
func IsSubresource(name string) bool {
	return strings.Contains(name, "/")
}

// LookupKind resolves a resource Kind to its catalog entry.
func LookupKind(kind resource.Kind) (APIResource, bool) {
	for _, entry := range builtinCatalog {
		if entry.Kind == kind {
			return entry, true
		}
	}
	return APIResource{}, false
}

// LookupCommandToken resolves a CLI token (primary name or alias) to its
// catalog entry.
func LookupCommandToken(token string) (APIResource, bool) {
	token = strings.ToLower(token)
	for _, entry := range builtinCatalog {
		if entry.CommandToken() == token {
			return entry, true
		}
		for _, alias := range entry.Aliases {
			if alias == token {
				return entry, true
			}
		}
	}
	return APIResource{}, false
}
