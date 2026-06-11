package kube_test

import (
	"testing"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

// findRelationship returns the first relationship whose target matches the kind
// and name, reporting whether one was found.
func findRelationship(rels []kube.Relationship, kind resource.Kind, name string) (kube.Relationship, bool) {
	for _, rel := range rels {
		if rel.Target.Kind == kind && rel.Target.Name == name {
			return rel, true
		}
	}
	return kube.Relationship{}, false
}

func TestTypedRelationshipsPod(t *testing.T) {
	t.Parallel()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
			Containers: []corev1.Container{
				{
					Name: "app",
					Env: []corev1.EnvVar{
						{Name: "TOKEN", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "api-secret"}}}},
					},
					EnvFrom: []corev1.EnvFromSource{
						{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"}}},
					},
				},
			},
			Volumes: []corev1.Volume{
				{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data-pvc"}}},
			},
		},
	}

	rels := kube.TypedRelationships(pod)

	if _, ok := findRelationship(rels, resource.ReferenceKindSecret, "api-secret"); !ok {
		t.Error("expected pod to reference secret api-secret")
	}
	if _, ok := findRelationship(rels, resource.ReferenceKindConfigMap, "app-config"); !ok {
		t.Error("expected pod to reference configmap app-config")
	}
	if rel, ok := findRelationship(rels, resource.ReferenceKindPersistentVolumeClaim, "data-pvc"); !ok || rel.EdgeKind != graph.EdgeMounts {
		t.Error("expected pod to mount pvc data-pvc")
	}
	if rel, ok := findRelationship(rels, resource.ReferenceKindNode, "node-1"); !ok || rel.EdgeKind != graph.EdgeScheduledOn {
		t.Error("expected pod to be scheduled on node-1")
	}
}

func TestTypedRelationshipsIngress(t *testing.T) {
	t.Parallel()

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{{SecretName: "web-tls"}},
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "web-svc"}}},
							},
						},
					},
				},
			},
		},
	}

	rels := kube.TypedRelationships(ing)

	if rel, ok := findRelationship(rels, resource.ReferenceKindService, "web-svc"); !ok || rel.Reason != "backend" {
		t.Error("expected ingress to reference backend service web-svc")
	}
	if rel, ok := findRelationship(rels, resource.ReferenceKindSecret, "web-tls"); !ok || rel.Reason != "tls" {
		t.Error("expected ingress to reference TLS secret web-tls")
	}
}

func TestTypedRelationshipsPVCStorageClass(t *testing.T) {
	t.Parallel()

	className := "fast"
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "data"},
		Spec:       corev1.PersistentVolumeClaimSpec{StorageClassName: &className},
	}

	rel, ok := findRelationship(kube.TypedRelationships(pvc), resource.ReferenceKindStorageClass, "fast")
	if !ok {
		t.Fatal("expected PVC to reference storage class fast")
	}
	if rel.EdgeKind != graph.EdgeUsesStorageClass {
		t.Errorf("storage class edge kind = %s, want %s", rel.EdgeKind, graph.EdgeUsesStorageClass)
	}
}

func TestTypedRelationshipsHPAScaleTarget(t *testing.T) {
	t.Parallel()

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{Kind: "Deployment", Name: "web", APIVersion: "apps/v1"},
		},
	}

	rel, ok := findRelationship(kube.TypedRelationships(hpa), resource.ReferenceKindDeployment, "web")
	if !ok {
		t.Fatal("expected HPA to reference scale target deployment web")
	}
	if rel.EdgeKind != graph.EdgeScaleTarget {
		t.Errorf("scale target edge kind = %s, want %s", rel.EdgeKind, graph.EdgeScaleTarget)
	}
}

func TestTypedRelationshipsRoleBinding(t *testing.T) {
	t.Parallel()

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bind"},
		RoleRef:    rbacv1.RoleRef{Kind: "Role", Name: "reader", APIGroup: "rbac.authorization.k8s.io"},
	}

	rel, ok := findRelationship(kube.TypedRelationships(rb), resource.ReferenceKindRole, "reader")
	if !ok {
		t.Fatal("expected role binding to reference role reader")
	}
	if rel.EdgeKind != graph.EdgeRoleRef {
		t.Errorf("role ref edge kind = %s, want %s", rel.EdgeKind, graph.EdgeRoleRef)
	}
}

func TestDynamicRelationshipsCertificate(t *testing.T) {
	t.Parallel()

	entry := kube.APIResource{Group: "cert-manager.io", Version: "v1", Resource: "certificates", Kind: resource.Kind("Certificate"), Namespaced: true}
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "cert-manager.io/v1",
		"kind":       "Certificate",
		"metadata":   map[string]any{"namespace": "default", "name": "web-cert"},
		"spec": map[string]any{
			"secretName": "web-tls",
			"issuerRef":  map[string]any{"name": "letsencrypt", "kind": "ClusterIssuer", "group": "cert-manager.io"},
		},
	}}

	rels := kube.DynamicRelationships(entry, obj, nil)

	secretRel, ok := findRelationship(rels, resource.ReferenceKindSecret, "web-tls")
	if !ok {
		t.Fatal("expected certificate to reference secret web-tls")
	}
	if secretRel.EdgeKind != graph.EdgeUsesSecret {
		t.Errorf("secret edge kind = %s, want %s", secretRel.EdgeKind, graph.EdgeUsesSecret)
	}

	if issuerRel, ok := findRelationship(rels, resource.Kind("ClusterIssuer"), "letsencrypt"); !ok || issuerRel.EdgeKind != graph.EdgeReferences {
		t.Fatal("expected certificate to reference ClusterIssuer letsencrypt")
	}
}
