package resource

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// Kind is the kind of the reference.
type Kind string

// KindAny is a wildcard kind. A rule that declares it in AppliesTo is evaluated
// against every node regardless of its concrete kind. It is intended for
// generic rules that operate on common Kubernetes patterns (events, status
// conditions, deletion timestamps) rather than a single resource type.
const KindAny Kind = "*"

const (
	ReferenceKindAPIService                Kind = "APIService"
	ReferenceKindCertificateSigningRequest Kind = "CertificateSigningRequest"
	ReferenceKindClusterRole               Kind = "ClusterRole"
	ReferenceKindClusterRoleBinding        Kind = "ClusterRoleBinding"
	ReferenceKindConfigMap                 Kind = "ConfigMap"
	ReferenceKindControllerRevision        Kind = "ControllerRevision"
	ReferenceKindCronJob                   Kind = "CronJob"
	ReferenceKindDaemonSet                 Kind = "DaemonSet"
	ReferenceKindDeployment                Kind = "Deployment"
	ReferenceKindEndpoint                  Kind = "Endpoint"
	ReferenceKindEndpoints                 Kind = "Endpoints"
	ReferenceKindEndpointSlice             Kind = "EndpointSlice"
	ReferenceKindEvent                     Kind = "Event"
	ReferenceKindHorizontalPodAutoscaler   Kind = "HorizontalPodAutoscaler"
	ReferenceKindIngress                   Kind = "Ingress"
	ReferenceKindIngressClass              Kind = "IngressClass"
	ReferenceKindJob                       Kind = "Job"
	ReferenceKindLease                     Kind = "Lease"
	ReferenceKindLimitRange                Kind = "LimitRange"
	ReferenceKindMutatingWebhookConfig     Kind = "MutatingWebhookConfiguration"
	ReferenceKindNamespace                 Kind = "Namespace"
	ReferenceKindNetworkPolicy             Kind = "NetworkPolicy"
	ReferenceKindNode                      Kind = "Node"
	ReferenceKindPersistentVolume          Kind = "PersistentVolume"
	ReferenceKindPersistentVolumeClaim     Kind = "PersistentVolumeClaim"
	ReferenceKindPod                       Kind = "Pod"
	ReferenceKindPodDisruptionBudget       Kind = "PodDisruptionBudget"
	ReferenceKindPodTemplate               Kind = "PodTemplate"
	ReferenceKindPriorityClass             Kind = "PriorityClass"
	ReferenceKindReplicaSet                Kind = "ReplicaSet"
	ReferenceKindReplicationController     Kind = "ReplicationController"
	ReferenceKindResourceQuota             Kind = "ResourceQuota"
	ReferenceKindRole                      Kind = "Role"
	ReferenceKindRoleBinding               Kind = "RoleBinding"
	ReferenceKindRuntimeClass              Kind = "RuntimeClass"
	ReferenceKindSecret                    Kind = "Secret"
	ReferenceKindService                   Kind = "Service"
	ReferenceKindServiceAccount            Kind = "ServiceAccount"
	ReferenceKindStatefulSet               Kind = "StatefulSet"
	ReferenceKindStorageClass              Kind = "StorageClass"
	ReferenceKindValidatingWebhookConfig   Kind = "ValidatingWebhookConfiguration"
)

// Reference is a reference to a resource.
type Reference struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       Kind   `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
	UID        string `json:"uid,omitempty"`
}

// NewReference creates a new resource reference.
func NewReference(kind Kind, apiVersion, namespace, name, uid string) Reference {
	if namespace == "" {
		namespace = "default"
	}

	return Reference{
		Kind:       kind,
		APIVersion: apiVersion,
		Namespace:  namespace,
		Name:       name,
		UID:        uid,
	}
}

// ReferenceFromObjectReference converts a Kubernetes core/v1 ObjectReference
// into a resource Reference. It is primarily used to map an Event's involved
// object onto the resource graph. Namespace defaulting is handled by
// NewReference.
func ReferenceFromObjectReference(ref corev1.ObjectReference) Reference {
	return NewReference(
		Kind(ref.Kind),
		ref.APIVersion,
		ref.Namespace,
		ref.Name,
		string(ref.UID),
	)
}

// Key returns a unique key for the resource based on the UID. If the UID is
// empty, it returns the logical key.
func (r Reference) Key() string {
	if r.UID == "" {
		return r.LogicalKey()
	}

	return fmt.Sprintf("uid/%s", r.UID)
}

// LogicalKey returns a unique key for the resource.
func (r Reference) LogicalKey() string {
	return fmt.Sprintf("%s/%s/%s/%s", r.APIVersion, r.Kind, r.Namespace, r.Name)
}

// Display returns a human-readable display name for the resource.
func (r Reference) Display() string {
	if r.UID == "" {
		return fmt.Sprintf("%s/%s/%s", r.Kind, r.Namespace, r.Name)
	}

	return fmt.Sprintf("%s/%s/%s (uid: %s)", r.Kind, r.Namespace, r.Name, r.UID)
}
