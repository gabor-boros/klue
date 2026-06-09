package kube_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
)

func typesUID(uid string) types.UID {
	return types.UID(uid)
}

func edgeExists(edges []graph.Edge, kind graph.EdgeKind, fromName, toName string) bool {
	for _, edge := range edges {
		if edge.Kind == kind && edge.From.Ref.Name == fromName && edge.To.Ref.Name == toName {
			return true
		}
	}
	return false
}

func TestResourceSnapshotBuildGraph(t *testing.T) {
	t.Parallel()

	scName := "standard"
	snapshot := &kube.ResourceSnapshot{
		Namespace: "default",
		Deployments: []appsv1.Deployment{
			{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web", UID: typesUID("d1")}},
		},
		ReplicaSets: []appsv1.ReplicaSet{
			{ObjectMeta: metav1.ObjectMeta{
				Namespace:       "default",
				Name:            "web-rs",
				UID:             typesUID("r1"),
				OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "web", UID: typesUID("d1")}},
			}},
		},
		Pods: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "default",
					Name:            "web-pod",
					UID:             typesUID("p1"),
					Labels:          map[string]string{"app": "web"},
					OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "web-rs", UID: typesUID("r1")}},
				},
				Spec: corev1.PodSpec{
					NodeName: "node-1",
					Volumes: []corev1.Volume{
						{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data"}}},
						{Name: "cfg", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "cfg"}}}},
						{Name: "sec", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "sec"}}},
					},
				},
			},
		},
		Services: []corev1.Service{
			{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web", UID: typesUID("s1")},
				Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": "web"}},
			},
		},
		EndpointSlices: []discoveryv1.EndpointSlice{
			{ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "web-abc123",
				UID:       typesUID("e1"),
				Labels:    map[string]string{discoveryv1.LabelServiceName: "web"},
			}},
		},
		PersistentVolumeClaims: []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "data", UID: typesUID("c1")},
				Spec:       corev1.PersistentVolumeClaimSpec{VolumeName: "vol-1", StorageClassName: &scName},
			},
		},
		PersistentVolumes: []corev1.PersistentVolume{
			{ObjectMeta: metav1.ObjectMeta{Name: "vol-1", UID: typesUID("v1")}},
		},
		StorageClasses: []storagev1.StorageClass{
			{ObjectMeta: metav1.ObjectMeta{Name: scName, UID: typesUID("sc1")}},
		},
		ConfigMaps: []corev1.ConfigMap{
			{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "cfg", UID: typesUID("cm1")}},
		},
		Secrets: []corev1.Secret{
			{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "sec", UID: typesUID("se1")}},
			{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "tls", UID: typesUID("se2")}},
		},
		Nodes: []corev1.Node{
			{ObjectMeta: metav1.ObjectMeta{Name: "node-1", UID: typesUID("n1")}},
		},
		Ingresses: []networkingv1.Ingress{
			{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-ing", UID: typesUID("i1")},
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{{SecretName: "tls"}},
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{Name: "web"},
					},
				},
			},
		},
	}

	g := snapshot.BuildGraph()
	edges := g.GetEdges()

	cases := []struct {
		name     string
		kind     graph.EdgeKind
		from, to string
	}{
		{"deployment owns replicaset", graph.EdgeOwns, "web", "web-rs"},
		{"replicaset owns pod", graph.EdgeOwns, "web-rs", "web-pod"},
		{"service selects pod", graph.EdgeSelectedBy, "web", "web-pod"},
		{"endpointslice references service", graph.EdgeReferences, "web-abc123", "web"},
		{"pod uses configmap", graph.EdgeUsesConfigMap, "web-pod", "cfg"},
		{"pod uses secret", graph.EdgeUsesSecret, "web-pod", "sec"},
		{"pod mounts pvc", graph.EdgeMounts, "web-pod", "data"},
		{"pvc references pv", graph.EdgeReferences, "data", "vol-1"},
		{"pvc uses storageclass", graph.EdgeUsesStorageClass, "data", "standard"},
		{"pod scheduled on node", graph.EdgeScheduledOn, "web-pod", "node-1"},
		{"ingress references service", graph.EdgeReferences, "web-ing", "web"},
		{"ingress uses tls secret", graph.EdgeUsesSecret, "web-ing", "tls"},
	}

	for _, tc := range cases {
		if !edgeExists(edges, tc.kind, tc.from, tc.to) {
			t.Errorf("expected edge %q: %s %s -> %s", tc.name, tc.kind, tc.from, tc.to)
		}
	}
}
