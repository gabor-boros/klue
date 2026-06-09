package kube_test

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

// certManagerResources is a discovery payload describing a namespaced custom
// resource (Certificate) and a cluster-scoped one (ClusterIssuer), plus a
// subresource and a non-listable resource that must be filtered out.
func certManagerResources() []*metav1.APIResourceList {
	return []*metav1.APIResourceList{
		{
			GroupVersion: "cert-manager.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "certificates", Kind: "Certificate", Namespaced: true, Verbs: metav1.Verbs{"list", "get", "watch"}},
				{Name: "certificates/status", Kind: "Certificate", Namespaced: true, Verbs: metav1.Verbs{"get"}},
				{Name: "clusterissuers", Kind: "ClusterIssuer", Namespaced: false, Verbs: metav1.Verbs{"list", "get"}},
				{Name: "challenges", Kind: "Challenge", Namespaced: true, Verbs: metav1.Verbs{"get"}},
			},
		},
	}
}

func TestDiscoverResourcesFiltersAndScopes(t *testing.T) {
	t.Parallel()

	typed := kubefake.NewClientset()
	typed.Resources = certManagerResources()

	client := kube.NewClientForInterfaces(typed, nil)

	resources, err := client.DiscoverResources()
	if err != nil {
		t.Fatalf("DiscoverResources() error = %v", err)
	}

	var certificate, clusterIssuer *kube.APIResource
	for i := range resources {
		switch resources[i].Kind {
		case "Certificate":
			certificate = &resources[i]
		case "ClusterIssuer":
			clusterIssuer = &resources[i]
		case "Challenge":
			t.Error("non-listable Challenge resource must not be discovered")
		}
		if kube.IsSubresource(resources[i].Resource) {
			t.Errorf("subresource %q must not be discovered", resources[i].Resource)
		}
	}

	if certificate == nil {
		t.Fatal("namespaced Certificate custom resource was not discovered")
	}
	if !certificate.Namespaced || !certificate.Custom || certificate.APIVersion() != "cert-manager.io/v1" {
		t.Errorf("Certificate descriptor = %+v, want namespaced custom cert-manager.io/v1", *certificate)
	}

	if clusterIssuer == nil {
		t.Fatal("cluster-scoped ClusterIssuer custom resource was not discovered")
	}
	if clusterIssuer.Namespaced {
		t.Error("ClusterIssuer should be cluster-scoped")
	}
}

func TestDiscoverResourcesSkipsBuiltinGroups(t *testing.T) {
	t.Parallel()

	typed := kubefake.NewClientset()
	typed.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Name: "deployments", Kind: "Deployment", Namespaced: true, Verbs: metav1.Verbs{"list"}},
			},
		},
	}

	client := kube.NewClientForInterfaces(typed, nil)

	resources, err := client.DiscoverResources()
	if err != nil {
		t.Fatalf("DiscoverResources() error = %v", err)
	}

	// The built-in catalog already contains exactly one Deployment entry; a
	// duplicate discovered from a built-in group would indicate the filter
	// failed.
	deployments := 0
	for _, r := range resources {
		if r.Kind == resource.ReferenceKindDeployment {
			deployments++
		}
	}
	if deployments != 1 {
		t.Errorf("Deployment entries = %d, want 1 (built-in group must not be re-discovered)", deployments)
	}
}

func TestResolveResource(t *testing.T) {
	t.Parallel()

	resources := []kube.APIResource{
		{Group: "cert-manager.io", Version: "v1", Resource: "certificates", Kind: "Certificate", Namespaced: true, Custom: true},
		{Group: "example.com", Version: "v1", Resource: "certificates", Kind: "Certificate", Namespaced: true, Custom: true},
		{Group: "networking.example.io", Version: "v1alpha1", Resource: "dnsendpoints", Kind: "DNSEndpoint", Namespaced: true, Custom: true, Aliases: []string{"dnsep"}},
	}

	t.Run("unambiguous by alias", func(t *testing.T) {
		t.Parallel()
		entry, err := kube.ResolveResource(resources, "dnsep", "")
		if err != nil {
			t.Fatalf("ResolveResource() error = %v", err)
		}
		if entry.Kind != "DNSEndpoint" {
			t.Errorf("Kind = %q, want DNSEndpoint", entry.Kind)
		}
	})

	t.Run("ambiguous requires api-version", func(t *testing.T) {
		t.Parallel()
		if _, err := kube.ResolveResource(resources, "certificate", ""); err == nil {
			t.Fatal("expected an ambiguity error for a kind served by two groups")
		}
	})

	t.Run("disambiguated by api-version", func(t *testing.T) {
		t.Parallel()
		entry, err := kube.ResolveResource(resources, "certificate", "cert-manager.io/v1")
		if err != nil {
			t.Fatalf("ResolveResource() error = %v", err)
		}
		if entry.Group != "cert-manager.io" {
			t.Errorf("Group = %q, want cert-manager.io", entry.Group)
		}
	})

	t.Run("unknown token", func(t *testing.T) {
		t.Parallel()
		if _, err := kube.ResolveResource(resources, "nope", ""); err == nil {
			t.Error("expected an error for an unknown resource token")
		}
	})
}

func TestClientFetchCustomResources(t *testing.T) {
	t.Parallel()

	typed := kubefake.NewClientset()
	typed.Resources = certManagerResources()

	certGVR := schema.GroupVersionResource{Group: "cert-manager.io", Version: "v1", Resource: "certificates"}
	issuerGVR := schema.GroupVersionResource{Group: "cert-manager.io", Version: "v1", Resource: "clusterissuers"}

	listKinds := dynamicListKinds()
	listKinds[certGVR] = "CertificateList"
	listKinds[issuerGVR] = "ClusterIssuerList"

	cert := unstructuredObject("cert-manager.io/v1", "Certificate", "default", "web-cert")
	otherCert := unstructuredObject("cert-manager.io/v1", "Certificate", "other", "api-cert")
	issuer := unstructuredObject("cert-manager.io/v1", "ClusterIssuer", "", "letsencrypt")

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(), listKinds, cert, otherCert, issuer,
	)

	client := kube.NewClientForInterfaces(typed, dynamicClient)

	snapshot, err := client.Fetch(context.Background(), "default")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	var sawCert, sawIssuer bool
	for _, obj := range snapshot.Dynamic {
		switch obj.Resource.Kind {
		case "Certificate":
			if obj.Object.GetNamespace() != "default" {
				t.Errorf("Certificate namespace = %q, want default (namespace scoping failed)", obj.Object.GetNamespace())
			}
			sawCert = true
		case "ClusterIssuer":
			sawIssuer = true
		}
	}

	if !sawCert {
		t.Error("custom resource fetch did not return the namespaced Certificate")
	}
	if !sawIssuer {
		t.Error("custom resource fetch did not return the cluster-scoped ClusterIssuer")
	}
}

func TestClientFetchWithResourcesUsesProvidedCatalog(t *testing.T) {
	t.Parallel()

	typed := kubefake.NewClientset()

	certEntry := kube.APIResource{
		Group:      "cert-manager.io",
		Version:    "v1",
		Resource:   "certificates",
		Kind:       "Certificate",
		Namespaced: true,
		Typed:      false,
		Custom:     true,
	}
	certGVR := certEntry.GroupVersionResource()

	listKinds := dynamicListKinds()
	listKinds[certGVR] = "CertificateList"

	cert := unstructuredObject("cert-manager.io/v1", "Certificate", "default", "web-cert")
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds, cert)

	client := kube.NewClientForInterfaces(typed, dynamicClient)

	resources := append(kube.BuiltinCatalog(), certEntry)
	snapshot, err := client.FetchWithResources(context.Background(), "default", resources)
	if err != nil {
		t.Fatalf("FetchWithResources() error = %v", err)
	}

	sawCert := false
	for _, obj := range snapshot.Dynamic {
		if obj.Resource.Kind == "Certificate" {
			sawCert = true
			if obj.Object.GetName() != "web-cert" {
				t.Errorf("Certificate name = %q, want web-cert", obj.Object.GetName())
			}
		}
	}
	if !sawCert {
		t.Fatal("FetchWithResources() did not list the provided custom resource type")
	}
}

func TestClientFetchCustomResourcesToleratesForbidden(t *testing.T) {
	t.Parallel()

	typed := kubefake.NewClientset()
	typed.Resources = certManagerResources()

	certGVR := schema.GroupVersionResource{Group: "cert-manager.io", Version: "v1", Resource: "certificates"}
	issuerGVR := schema.GroupVersionResource{Group: "cert-manager.io", Version: "v1", Resource: "clusterissuers"}

	listKinds := dynamicListKinds()
	listKinds[certGVR] = "CertificateList"
	listKinds[issuerGVR] = "ClusterIssuerList"

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds)
	dynamicClient.PrependReactor("list", "certificates", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "cert-manager.io", Resource: "certificates"}, "", nil)
	})

	client := kube.NewClientForInterfaces(typed, dynamicClient)

	if _, err := client.Fetch(context.Background(), "default"); err != nil {
		t.Fatalf("Fetch() error = %v, want nil (forbidden custom resource list must be tolerated)", err)
	}
}
