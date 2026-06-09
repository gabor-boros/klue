package kube_test

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clienttesting "k8s.io/client-go/testing"

	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

// dynamicListKinds builds the GVR -> list-kind mapping the fake dynamic client
// needs from the built-in catalog, so listing any dynamic resource succeeds.
func dynamicListKinds() map[schema.GroupVersionResource]string {
	listKinds := make(map[schema.GroupVersionResource]string)
	for _, entry := range kube.BuiltinCatalog() {
		if entry.Typed {
			continue
		}
		listKinds[entry.GroupVersionResource()] = string(entry.Kind) + "List"
	}
	return listKinds
}

func unstructuredObject(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	if namespace != "" {
		obj.SetNamespace(namespace)
	}
	obj.SetName(name)
	return obj
}

func TestClientFetchDynamic(t *testing.T) {
	t.Parallel()

	typed := kubefake.NewClientset(&corev1.Pod{ObjectMeta: meta("default", "pod-a")})

	sa := unstructuredObject("v1", "ServiceAccount", "default", "builder")
	otherSA := unstructuredObject("v1", "ServiceAccount", "other", "builder")
	ns := unstructuredObject("v1", "Namespace", "", "default")

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		dynamicListKinds(),
		sa, otherSA, ns,
	)

	client := kube.NewClientForInterfaces(typed, dynamicClient)

	snapshot, err := client.Fetch(context.Background(), "default")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	var sawSA, sawNS bool
	for _, obj := range snapshot.Dynamic {
		switch obj.Resource.Kind {
		case resource.ReferenceKindServiceAccount:
			if obj.Object.GetNamespace() != "default" {
				t.Errorf("ServiceAccount namespace = %q, want default (namespace scoping failed)", obj.Object.GetNamespace())
			}
			sawSA = true
		case resource.ReferenceKindNamespace:
			sawNS = true
		}
	}

	if !sawSA {
		t.Error("dynamic fetch did not return the namespaced ServiceAccount")
	}
	if !sawNS {
		t.Error("dynamic fetch did not return the cluster-scoped Namespace")
	}
}

func TestClientFetchDynamicToleratesForbidden(t *testing.T) {
	t.Parallel()

	typed := kubefake.NewClientset()

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		dynamicListKinds(),
	)

	// Simulate a restricted cluster where listing one resource is forbidden.
	dynamicClient.PrependReactor("list", "resourcequotas", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "resourcequotas"}, "", nil)
	})

	client := kube.NewClientForInterfaces(typed, dynamicClient)

	if _, err := client.Fetch(context.Background(), "default"); err != nil {
		t.Fatalf("Fetch() error = %v, want nil (forbidden list must be tolerated)", err)
	}
}

func TestClientFetchWithoutDynamicClient(t *testing.T) {
	t.Parallel()

	// NewClientForInterface leaves the dynamic client nil; Fetch must still work.
	client := kube.NewClientForInterface(kubefake.NewClientset(&corev1.Pod{ObjectMeta: meta("default", "pod-a")}))

	snapshot, err := client.Fetch(context.Background(), "default")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(snapshot.Dynamic) != 0 {
		t.Errorf("Dynamic = %d, want 0 without a dynamic client", len(snapshot.Dynamic))
	}
}

func TestClientFetchDynamicDeterministicOrder(t *testing.T) {
	t.Parallel()

	typed := kubefake.NewClientset()

	saZ := unstructuredObject("v1", "ServiceAccount", "default", "z-builder")
	saA := unstructuredObject("v1", "ServiceAccount", "default", "a-builder")
	ns := unstructuredObject("v1", "Namespace", "", "default")

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		dynamicListKinds(),
		saZ, saA, ns,
	)

	client := kube.NewClientForInterfaces(typed, dynamicClient)

	first, err := client.Fetch(context.Background(), "default")
	if err != nil {
		t.Fatalf("first Fetch() error = %v", err)
	}
	second, err := client.Fetch(context.Background(), "default")
	if err != nil {
		t.Fatalf("second Fetch() error = %v", err)
	}

	order := func(dynamic []kube.DynamicObject) []string {
		out := make([]string, 0, len(dynamic))
		for i := range dynamic {
			obj := dynamic[i]
			out = append(out, obj.Resource.Resource+"/"+obj.Object.GetNamespace()+"/"+obj.Object.GetName())
		}
		return out
	}

	firstOrder := order(first.Dynamic)
	secondOrder := order(second.Dynamic)

	if !reflect.DeepEqual(firstOrder, secondOrder) {
		t.Fatalf("dynamic order differs between fetches:\nfirst=%v\nsecond=%v", firstOrder, secondOrder)
	}

	idxA, idxZ := -1, -1
	for i := range firstOrder {
		switch firstOrder[i] {
		case "serviceaccounts/default/a-builder":
			idxA = i
		case "serviceaccounts/default/z-builder":
			idxZ = i
		}
	}
	if idxA == -1 || idxZ == -1 || idxA > idxZ {
		t.Fatalf("serviceaccounts are not deterministically sorted by name: %v", firstOrder)
	}
}

func TestBuildGraphDynamicNodesAndOwnerEdges(t *testing.T) {
	t.Parallel()

	owner := unstructuredObject("v1", "ServiceAccount", "default", "owner")
	owner.SetUID("owner-uid")

	owned := unstructuredObject("v1", "ResourceQuota", "default", "owned")
	owned.SetUID("owned-uid")
	owned.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: "v1",
		Kind:       "ServiceAccount",
		Name:       "owner",
		UID:        "owner-uid",
	}})

	saEntry, _ := kube.LookupKind(resource.ReferenceKindServiceAccount)
	rqEntry, _ := kube.LookupKind(resource.ReferenceKindResourceQuota)

	snapshot := &kube.ResourceSnapshot{
		Namespace: "default",
		Dynamic: []kube.DynamicObject{
			{Resource: saEntry, Object: owner},
			{Resource: rqEntry, Object: owned},
		},
	}

	g := snapshot.BuildGraph()

	ownerRef := resource.NewReference(resource.ReferenceKindServiceAccount, "v1", "default", "owner", "owner-uid")
	if _, found := g.FindByRef(ownerRef); !found {
		t.Fatal("dynamic ServiceAccount node was not added to the graph")
	}

	if !edgeExists(g.GetEdges(), graph.EdgeOwns, "owner", "owned") {
		t.Error("expected owner edge ServiceAccount -> ResourceQuota from dynamic owner references")
	}
}
