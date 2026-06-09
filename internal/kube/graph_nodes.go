package kube

import (
	"github.com/gabor-boros/klue/pkg/resource"
)

// addNodes adds a graph node for every object in the typed snapshot, then adds
// the dynamically fetched (unstructured) objects.
func (b *graphBuilder) addNodes(s *ResourceSnapshot) {
	for i := range s.Pods {
		p := &s.Pods[i]
		b.addNode(resource.ReferenceKindPod, apiVersionCore, p.Namespace, p.Name, string(p.UID), p, p.Labels, podStatus(p))
	}
	for i := range s.Services {
		o := &s.Services[i]
		b.addNode(resource.ReferenceKindService, apiVersionCore, o.Namespace, o.Name, string(o.UID), o, o.Labels, serviceStatus(o))
	}
	for i := range s.EndpointSlices {
		o := &s.EndpointSlices[i]
		b.addNode(resource.ReferenceKindEndpointSlice, apiVersionDiscovery, o.Namespace, o.Name, string(o.UID), o, o.Labels, endpointSliceStatus(o))
	}
	for i := range s.ConfigMaps {
		o := &s.ConfigMaps[i]
		b.addNode(resource.ReferenceKindConfigMap, apiVersionCore, o.Namespace, o.Name, string(o.UID), o, o.Labels, resource.StatusHealthy)
	}
	for i := range s.Secrets {
		o := &s.Secrets[i]
		b.addNode(resource.ReferenceKindSecret, apiVersionCore, o.Namespace, o.Name, string(o.UID), o, o.Labels, resource.StatusHealthy)
	}
	for i := range s.PersistentVolumeClaims {
		o := &s.PersistentVolumeClaims[i]
		b.addNode(resource.ReferenceKindPersistentVolumeClaim, apiVersionCore, o.Namespace, o.Name, string(o.UID), o, o.Labels, pvcStatus(o))
	}
	for i := range s.PersistentVolumes {
		o := &s.PersistentVolumes[i]
		b.addNode(resource.ReferenceKindPersistentVolume, apiVersionCore, o.Namespace, o.Name, string(o.UID), o, o.Labels, pvStatus(o))
	}
	for i := range s.Nodes {
		o := &s.Nodes[i]
		b.addNode(resource.ReferenceKindNode, apiVersionCore, o.Namespace, o.Name, string(o.UID), o, o.Labels, nodeStatus(o))
	}
	for i := range s.Deployments {
		o := &s.Deployments[i]
		b.addNode(resource.ReferenceKindDeployment, apiVersionApps, o.Namespace, o.Name, string(o.UID), o, o.Labels, deploymentStatus(o))
	}
	for i := range s.ReplicaSets {
		o := &s.ReplicaSets[i]
		b.addNode(resource.ReferenceKindReplicaSet, apiVersionApps, o.Namespace, o.Name, string(o.UID), o, o.Labels, replicaSetStatus(o))
	}
	for i := range s.StatefulSets {
		o := &s.StatefulSets[i]
		b.addNode(resource.ReferenceKindStatefulSet, apiVersionApps, o.Namespace, o.Name, string(o.UID), o, o.Labels, statefulSetStatus(o))
	}
	for i := range s.Jobs {
		o := &s.Jobs[i]
		b.addNode(resource.ReferenceKindJob, apiVersionBatch, o.Namespace, o.Name, string(o.UID), o, o.Labels, jobStatus(o))
	}
	for i := range s.CronJobs {
		o := &s.CronJobs[i]
		b.addNode(resource.ReferenceKindCronJob, apiVersionBatch, o.Namespace, o.Name, string(o.UID), o, o.Labels, cronJobStatus(o))
	}
	for i := range s.Ingresses {
		o := &s.Ingresses[i]
		b.addNode(resource.ReferenceKindIngress, apiVersionNetworking, o.Namespace, o.Name, string(o.UID), o, o.Labels, ingressStatus(o))
	}
	for i := range s.StorageClasses {
		o := &s.StorageClasses[i]
		b.addNode(resource.ReferenceKindStorageClass, apiVersionStorage, o.Namespace, o.Name, string(o.UID), o, o.Labels, storageClassStatus(o))
	}
	for i := range s.DaemonSets {
		o := &s.DaemonSets[i]
		b.addNode(resource.ReferenceKindDaemonSet, apiVersionApps, o.Namespace, o.Name, string(o.UID), o, o.Labels, daemonSetStatus(o))
	}
	for i := range s.NetworkPolicies {
		o := &s.NetworkPolicies[i]
		b.addNode(resource.ReferenceKindNetworkPolicy, apiVersionNetworking, o.Namespace, o.Name, string(o.UID), o, o.Labels, resource.StatusHealthy)
	}
	for i := range s.HorizontalPodAutoscalers {
		o := &s.HorizontalPodAutoscalers[i]
		b.addNode(resource.ReferenceKindHorizontalPodAutoscaler, apiVersionAutoscaling, o.Namespace, o.Name, string(o.UID), o, o.Labels, hpaStatus(o))
	}
	for i := range s.PodDisruptionBudgets {
		o := &s.PodDisruptionBudgets[i]
		b.addNode(resource.ReferenceKindPodDisruptionBudget, apiVersionPolicy, o.Namespace, o.Name, string(o.UID), o, o.Labels, pdbStatus(o))
	}
	for i := range s.Roles {
		o := &s.Roles[i]
		b.addNode(resource.ReferenceKindRole, apiVersionRBAC, o.Namespace, o.Name, string(o.UID), o, o.Labels, resource.StatusHealthy)
	}
	for i := range s.RoleBindings {
		o := &s.RoleBindings[i]
		b.addNode(resource.ReferenceKindRoleBinding, apiVersionRBAC, o.Namespace, o.Name, string(o.UID), o, o.Labels, resource.StatusHealthy)
	}
	for i := range s.ClusterRoles {
		o := &s.ClusterRoles[i]
		b.addNode(resource.ReferenceKindClusterRole, apiVersionRBAC, o.Namespace, o.Name, string(o.UID), o, o.Labels, resource.StatusHealthy)
	}
	for i := range s.ClusterRoleBindings {
		o := &s.ClusterRoleBindings[i]
		b.addNode(resource.ReferenceKindClusterRoleBinding, apiVersionRBAC, o.Namespace, o.Name, string(o.UID), o, o.Labels, resource.StatusHealthy)
	}
	for i := range s.CertificateSigningRequests {
		o := &s.CertificateSigningRequests[i]
		b.addNode(resource.ReferenceKindCertificateSigningRequest, apiVersionCertificates, o.Namespace, o.Name, string(o.UID), o, o.Labels, csrStatus(o))
	}
	for i := range s.Leases {
		o := &s.Leases[i]
		b.addNode(resource.ReferenceKindLease, apiVersionCoordination, o.Namespace, o.Name, string(o.UID), o, o.Labels, resource.StatusHealthy)
	}

	b.addDynamicNodes(s)
}

// addDynamicNodes adds graph nodes for built-in resources fetched as
// unstructured data. Resources already represented by a typed node (matched by
// logical key) are skipped so the typed object remains canonical.
func (b *graphBuilder) addDynamicNodes(s *ResourceSnapshot) {
	for i := range s.Dynamic {
		obj := s.Dynamic[i].Object
		entry := s.Dynamic[i].Resource

		ref := resource.NewReference(entry.Kind, entry.APIVersion(), obj.GetNamespace(), obj.GetName(), string(obj.GetUID()))
		if _, exists := b.byLogical[ref.LogicalKey()]; exists {
			continue
		}

		b.addNode(entry.Kind, entry.APIVersion(), obj.GetNamespace(), obj.GetName(), string(obj.GetUID()), obj, obj.GetLabels(), unstructuredStatus(obj))
	}
}
