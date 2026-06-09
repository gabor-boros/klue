package kube_test

import (
	"context"
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/gabor-boros/klue/internal/kube"
)

func BenchmarkClientFetch(b *testing.B) {
	objects := make([]runtime.Object, 0, 1200)
	objects = append(objects,
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}},
		&corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "pv-1"}},
		&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "standard"}},
	)

	for i := 0; i < 800; i++ {
		objects = append(objects, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      fmt.Sprintf("pod-%04d", i),
				Labels:    map[string]string{"app": fmt.Sprintf("app-%d", i%40)},
			},
		})
	}
	for i := 0; i < 200; i++ {
		objects = append(objects,
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: fmt.Sprintf("svc-%03d", i)},
				Spec:       corev1.ServiceSpec{Selector: map[string]string{"app": fmt.Sprintf("app-%d", i%40)}},
			},
			&discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      fmt.Sprintf("slice-%03d", i),
					Labels:    map[string]string{discoveryv1.LabelServiceName: fmt.Sprintf("svc-%03d", i)},
				},
			},
		)
	}
	for i := 0; i < 200; i++ {
		objects = append(objects, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: fmt.Sprintf("deploy-%03d", i)},
		})
	}

	clientset := kubefake.NewClientset(objects...)
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), dynamicListKinds())
	client := kube.NewClientForInterfaces(clientset, dynamicClient)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := client.Fetch(context.Background(), "default"); err != nil {
			b.Fatalf("Fetch() error = %v", err)
		}
	}
}

func BenchmarkBuildGraph(b *testing.B) {
	snapshot := &kube.ResourceSnapshot{Namespace: "default"}
	snapshot.Nodes = []corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}}}

	for i := 0; i < 1200; i++ {
		snapshot.Pods = append(snapshot.Pods, corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      fmt.Sprintf("pod-%04d", i),
				UID:       typesUID(fmt.Sprintf("pod-uid-%04d", i)),
				Labels:    map[string]string{"app": fmt.Sprintf("app-%d", i%40)},
			},
		})
	}
	for i := 0; i < 240; i++ {
		snapshot.Services = append(snapshot.Services, corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      fmt.Sprintf("svc-%03d", i),
				UID:       typesUID(fmt.Sprintf("svc-uid-%03d", i)),
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app": fmt.Sprintf("app-%d", i%40)},
			},
		})
		snapshot.EndpointSlices = append(snapshot.EndpointSlices, discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      fmt.Sprintf("slice-%03d", i),
				UID:       typesUID(fmt.Sprintf("slice-uid-%03d", i)),
				Labels: map[string]string{
					discoveryv1.LabelServiceName: fmt.Sprintf("svc-%03d", i),
				},
			},
			Endpoints: []discoveryv1.Endpoint{
				{
					Addresses: []string{fmt.Sprintf("10.0.%d.%d", i/255, i%255+1)},
					Conditions: discoveryv1.EndpointConditions{
						Ready: boolPtr(true),
					},
				},
			},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = snapshot.BuildGraph()
	}
}

func BenchmarkBuildGraphWithDynamicRelationships(b *testing.B) {
	certEntry := kube.APIResource{
		Group: "cert-manager.io", Version: "v1", Resource: "certificates",
		Kind: "Certificate", Namespaced: true, Custom: true,
	}
	issuerEntry := kube.APIResource{
		Group: "cert-manager.io", Version: "v1", Resource: "issuers",
		Kind: "Issuer", Namespaced: true, Custom: true,
	}

	snapshot := &kube.ResourceSnapshot{Namespace: "default"}
	for i := 0; i < 800; i++ {
		cert := &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "Certificate",
			"metadata": map[string]any{
				"name":      fmt.Sprintf("cert-%04d", i),
				"namespace": "default",
				"uid":       fmt.Sprintf("cert-uid-%04d", i),
			},
			"spec": map[string]any{
				"secretName": fmt.Sprintf("tls-%04d", i),
				"issuerRef": map[string]any{
					"name": "issuer-shared",
					"kind": "Issuer",
				},
			},
		}}
		snapshot.Dynamic = append(snapshot.Dynamic, kube.DynamicObject{Resource: certEntry, Object: cert})
	}
	issuer := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "cert-manager.io/v1",
		"kind":       "Issuer",
		"metadata": map[string]any{
			"name":      "issuer-shared",
			"namespace": "default",
			"uid":       "issuer-shared-uid",
		},
		"spec": map[string]any{
			"acme": map[string]any{
				"privateKeySecretRef": map[string]any{"name": "acme-account-key"},
			},
		},
	}}
	snapshot.Dynamic = append(snapshot.Dynamic, kube.DynamicObject{Resource: issuerEntry, Object: issuer})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = snapshot.BuildGraph()
	}
}

func boolPtr(v bool) *bool { return &v }
