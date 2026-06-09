package kube

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gabor-boros/klue/pkg/condition"
	"github.com/gabor-boros/klue/pkg/resource"
)

// podStatus maps a pod phase to a resource status.
func podStatus(pod *corev1.Pod) resource.Status {
	if pod.DeletionTimestamp != nil {
		return resource.StatusTerminating
	}

	switch pod.Status.Phase {
	case corev1.PodPending:
		return resource.StatusPending
	case corev1.PodRunning:
		if podConditionTrue(pod.Status.Conditions, corev1.PodReady) {
			return resource.StatusReady
		}
		return resource.StatusRunning
	case corev1.PodSucceeded:
		return resource.StatusSucceeded
	case corev1.PodFailed:
		return resource.StatusFailed
	default:
		return resource.StatusUnknown
	}
}

// pvcStatus maps a PVC phase to a resource status.
func pvcStatus(pvc *corev1.PersistentVolumeClaim) resource.Status {
	switch pvc.Status.Phase {
	case corev1.ClaimBound:
		return resource.StatusReady
	case corev1.ClaimPending:
		return resource.StatusPending
	case corev1.ClaimLost:
		return resource.StatusFailed
	default:
		return resource.StatusUnknown
	}
}

// deploymentStatus maps Deployment rollout/availability to a resource status.
func deploymentStatus(deploy *appsv1.Deployment) resource.Status {
	if deploy.Spec.Paused {
		return resource.StatusSuspended
	}

	desired := int32(1)
	if deploy.Spec.Replicas != nil {
		desired = *deploy.Spec.Replicas
	}
	if desired == 0 {
		return resource.StatusHealthy
	}

	for _, condition := range deploy.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing && condition.Reason == "ProgressDeadlineExceeded" {
			return resource.StatusDegraded
		}
	}

	if deploy.Status.AvailableReplicas >= desired {
		return resource.StatusReady
	}
	if deploy.Status.UpdatedReplicas < desired || deploy.Status.AvailableReplicas < desired {
		return resource.StatusProgressing
	}

	return resource.StatusUnknown
}

// replicaSetStatus maps ReplicaSet replica readiness to a resource status.
func replicaSetStatus(rs *appsv1.ReplicaSet) resource.Status {
	desired := rs.Status.Replicas
	if desired == 0 {
		return resource.StatusHealthy
	}
	if rs.Status.ReadyReplicas >= desired {
		return resource.StatusReady
	}
	return resource.StatusProgressing
}

// statefulSetStatus maps StatefulSet rollout progression to a resource status.
func statefulSetStatus(sts *appsv1.StatefulSet) resource.Status {
	desired := int32(1)
	if sts.Spec.Replicas != nil {
		desired = *sts.Spec.Replicas
	}
	if desired == 0 {
		return resource.StatusHealthy
	}
	if sts.Status.ReadyReplicas >= desired {
		return resource.StatusReady
	}
	return resource.StatusProgressing
}

// jobStatus maps Job completion state to a resource status.
func jobStatus(job *batchv1.Job) resource.Status {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return resource.StatusFailed
		}
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return resource.StatusCompleted
		}
	}
	if job.Status.Active > 0 {
		return resource.StatusRunning
	}
	return resource.StatusUnknown
}

// cronJobStatus maps CronJob scheduling state to a resource status.
func cronJobStatus(cj *batchv1.CronJob) resource.Status {
	if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
		return resource.StatusSuspended
	}
	return resource.StatusHealthy
}

// serviceStatus maps Service configuration to a resource status.
func serviceStatus(svc *corev1.Service) resource.Status {
	if len(svc.Spec.Selector) > 0 {
		return resource.StatusHealthy
	}
	return resource.StatusUnknown
}

// endpointSliceStatus maps EndpointSlice readiness to a resource status.
func endpointSliceStatus(slice *discoveryv1.EndpointSlice) resource.Status {
	for _, endpoint := range slice.Endpoints {
		if endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready && len(endpoint.Addresses) > 0 {
			return resource.StatusReady
		}
	}
	return resource.StatusNotReady
}

// pvStatus maps PV phases to a resource status.
func pvStatus(pv *corev1.PersistentVolume) resource.Status {
	switch pv.Status.Phase {
	case corev1.VolumeBound:
		return resource.StatusReady
	case corev1.VolumePending:
		return resource.StatusPending
	case corev1.VolumeReleased, corev1.VolumeFailed:
		return resource.StatusFailed
	case corev1.VolumeAvailable:
		return resource.StatusHealthy
	default:
		return resource.StatusUnknown
	}
}

// nodeStatus maps Node readiness, pressure and scheduling state to a status.
func nodeStatus(node *corev1.Node) resource.Status {
	if node.DeletionTimestamp != nil {
		return resource.StatusTerminating
	}

	ready := nodeCondition(node.Status.Conditions, corev1.NodeReady)
	if ready == corev1.ConditionFalse || ready == corev1.ConditionUnknown {
		return resource.StatusNotReady
	}

	pressureTypes := []corev1.NodeConditionType{
		corev1.NodeMemoryPressure,
		corev1.NodeDiskPressure,
		corev1.NodePIDPressure,
		corev1.NodeNetworkUnavailable,
	}
	for _, conditionType := range pressureTypes {
		if nodeCondition(node.Status.Conditions, conditionType) == corev1.ConditionTrue {
			return resource.StatusDegraded
		}
	}

	if node.Spec.Unschedulable {
		return resource.StatusSuspended
	}

	if ready == corev1.ConditionTrue {
		return resource.StatusReady
	}

	return resource.StatusUnknown
}

// ingressStatus maps Ingress load balancer assignment to a resource status.
func ingressStatus(ing *networkingv1.Ingress) resource.Status {
	if len(ing.Status.LoadBalancer.Ingress) > 0 {
		return resource.StatusReady
	}
	return resource.StatusProgressing
}

// storageClassStatus returns a stable status for a StorageClass resource.
func storageClassStatus(_ *storagev1.StorageClass) resource.Status {
	return resource.StatusHealthy
}

// daemonSetStatus maps DaemonSet scheduling/availability to a resource status.
func daemonSetStatus(ds *appsv1.DaemonSet) resource.Status {
	desired := ds.Status.DesiredNumberScheduled
	if desired == 0 {
		return resource.StatusHealthy
	}
	if ds.Status.NumberUnavailable > 0 {
		return resource.StatusDegraded
	}
	if ds.Status.NumberReady >= desired {
		return resource.StatusReady
	}
	return resource.StatusProgressing
}

// hpaStatus maps HorizontalPodAutoscaler conditions to a resource status.
func hpaStatus(hpa *autoscalingv2.HorizontalPodAutoscaler) resource.Status {
	for _, condition := range hpa.Status.Conditions {
		switch condition.Type {
		case autoscalingv2.ScalingActive:
			if condition.Status == corev1.ConditionFalse {
				return resource.StatusDegraded
			}
		case autoscalingv2.AbleToScale:
			if condition.Status == corev1.ConditionFalse {
				return resource.StatusDegraded
			}
		}
	}
	return resource.StatusHealthy
}

// pdbStatus maps PodDisruptionBudget disruption headroom to a resource status.
func pdbStatus(pdb *policyv1.PodDisruptionBudget) resource.Status {
	if pdb.Status.ExpectedPods > 0 && pdb.Status.CurrentHealthy < pdb.Status.DesiredHealthy {
		return resource.StatusDegraded
	}
	return resource.StatusHealthy
}

// csrStatus maps a CertificateSigningRequest approval state to a status.
func csrStatus(csr *certificatesv1.CertificateSigningRequest) resource.Status {
	for _, condition := range csr.Status.Conditions {
		switch condition.Type {
		case certificatesv1.CertificateDenied, certificatesv1.CertificateFailed:
			return resource.StatusFailed
		case certificatesv1.CertificateApproved:
			if len(csr.Status.Certificate) > 0 {
				return resource.StatusReady
			}
			return resource.StatusProgressing
		}
	}
	return resource.StatusPending
}

// unstructuredStatus derives a status from the common fields of an unstructured
// object: deletion timestamp, status.phase and status.conditions.
func unstructuredStatus(obj *unstructured.Unstructured) resource.Status {
	if obj.GetDeletionTimestamp() != nil {
		return resource.StatusTerminating
	}

	if phase, found, _ := unstructured.NestedString(obj.Object, "status", "phase"); found && phase != "" {
		switch phase {
		case "Active", "Bound", "Available":
			return resource.StatusReady
		case "Pending":
			return resource.StatusPending
		case "Failed", "Lost":
			return resource.StatusFailed
		case "Released", "Terminating":
			return resource.StatusTerminating
		case "Succeeded":
			return resource.StatusSucceeded
		}
	}

	if condition.AnyFailing(condition.FromUnstructured(obj)) {
		return resource.StatusDegraded
	}

	return resource.StatusHealthy
}

func podConditionTrue(conditions []corev1.PodCondition, conditionType corev1.PodConditionType) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func nodeCondition(conditions []corev1.NodeCondition, conditionType corev1.NodeConditionType) corev1.ConditionStatus {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return corev1.ConditionUnknown
}
