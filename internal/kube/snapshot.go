package kube

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	certificatesv1 "k8s.io/api/certificates/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DynamicObject is a resource fetched as unstructured data, either a built-in
// type without a typed fetcher or a discovered custom resource. It pairs the raw
// object with the descriptor it was fetched from so the graph builder can reason
// about its kind and scope.
type DynamicObject struct {
	Resource APIResource
	Object   *unstructured.Unstructured
}

// ResourceSnapshot is a point-in-time collection of the Kubernetes objects
// relevant to a diagnosis. It is the input from which the resource graph and
// event index are built.
type ResourceSnapshot struct {
	// Namespace is the namespace the namespaced objects were fetched from.
	Namespace string

	// Namespaced core resources.
	Pods                   []corev1.Pod
	Services               []corev1.Service
	EndpointSlices         []discoveryv1.EndpointSlice
	ConfigMaps             []corev1.ConfigMap
	Secrets                []corev1.Secret
	PersistentVolumeClaims []corev1.PersistentVolumeClaim
	Events                 []corev1.Event

	// Cluster-scoped core resources.
	Nodes             []corev1.Node
	PersistentVolumes []corev1.PersistentVolume

	// Workloads.
	Deployments  []appsv1.Deployment
	ReplicaSets  []appsv1.ReplicaSet
	StatefulSets []appsv1.StatefulSet
	DaemonSets   []appsv1.DaemonSet
	Jobs         []batchv1.Job
	CronJobs     []batchv1.CronJob

	// Networking.
	Ingresses       []networkingv1.Ingress
	NetworkPolicies []networkingv1.NetworkPolicy

	// Storage (cluster-scoped).
	StorageClasses []storagev1.StorageClass

	// Autoscaling.
	HorizontalPodAutoscalers []autoscalingv2.HorizontalPodAutoscaler

	// Policy.
	PodDisruptionBudgets []policyv1.PodDisruptionBudget

	// RBAC.
	Roles               []rbacv1.Role
	RoleBindings        []rbacv1.RoleBinding
	ClusterRoles        []rbacv1.ClusterRole
	ClusterRoleBindings []rbacv1.ClusterRoleBinding

	// Certificates (cluster-scoped).
	CertificateSigningRequests []certificatesv1.CertificateSigningRequest

	// Coordination.
	Leases []coordinationv1.Lease

	// Dynamic holds resources fetched as unstructured data: built-in kinds that
	// are not retrieved through the typed clientset and discovered custom
	// resources.
	Dynamic []DynamicObject
}
