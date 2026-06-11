package kube_test

import (
	"context"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

func newDynamicClient(extraKinds map[schema.GroupVersionResource]string, objs ...runtime.Object) *dynamicfake.FakeDynamicClient {
	listKinds := dynamicListKinds()
	for gvr, kind := range extraKinds {
		listKinds[gvr] = kind
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds, objs...)
}

func TestParseCRDFetchMode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in      string
		want    kube.CRDFetchMode
		wantErr bool
	}{
		{in: "", want: kube.CRDFetchRelated},
		{in: "all", want: kube.CRDFetchAll},
		{in: "related", want: kube.CRDFetchRelated},
		{in: "none", want: kube.CRDFetchNone},
		{in: "NoNe", want: kube.CRDFetchNone},
		{in: "invalid", wantErr: true},
	} {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got, err := kube.ParseCRDFetchMode(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseCRDFetchMode(%q) error = nil, want error", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCRDFetchMode(%q) error = %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParseCRDFetchMode(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFetchSnapshot_TargetPrefetchNotFound(t *testing.T) {
	t.Parallel()

	podEntry, ok := kube.LookupKind(resource.ReferenceKindPod)
	if !ok {
		t.Fatal("pod entry not found in catalog")
	}

	client := kube.NewClientForInterfaces(
		kubefake.NewClientset(),
		newDynamicClient(nil),
	)

	_, err := client.FetchSnapshot(context.Background(), "default", kube.SnapshotFetchOptions{
		Resources:      kube.BuiltinCatalog(),
		TargetResource: podEntry,
		TargetName:     "missing",
		CRDFetchMode:   kube.CRDFetchRelated,
	})
	if err == nil {
		t.Fatal("FetchSnapshot() error = nil, want not-found from target prefetch")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("FetchSnapshot() error = %q, want not-found message", err)
	}
}

func TestFetchSnapshot_RelatedCRDsForCustomTarget(t *testing.T) {
	t.Parallel()

	certEntry := kube.APIResource{
		Group:      "cert-manager.io",
		Version:    "v1",
		Resource:   "certificates",
		Kind:       resource.Kind("Certificate"),
		Namespaced: true,
		Custom:     true,
	}
	issuerEntry := kube.APIResource{
		Group:      "cert-manager.io",
		Version:    "v1",
		Resource:   "issuers",
		Kind:       resource.Kind("Issuer"),
		Namespaced: true,
		Custom:     true,
	}
	widgetEntry := kube.APIResource{
		Group:      "demo.klue.io",
		Version:    "v1",
		Resource:   "widgets",
		Kind:       resource.Kind("Widget"),
		Namespaced: true,
		Custom:     true,
	}

	dyn := newDynamicClient(map[schema.GroupVersionResource]string{
		certEntry.GroupVersionResource():   "CertificateList",
		issuerEntry.GroupVersionResource(): "IssuerList",
		widgetEntry.GroupVersionResource(): "WidgetList",
	},
		unstructuredObject("cert-manager.io/v1", "Certificate", "default", "web-cert"),
		unstructuredObject("cert-manager.io/v1", "Issuer", "default", "letsencrypt"),
		unstructuredObject("demo.klue.io/v1", "Widget", "default", "widget-a"),
	)

	client := kube.NewClientForInterfaces(kubefake.NewClientset(), dyn)
	snapshot, err := client.FetchSnapshot(context.Background(), "default", kube.SnapshotFetchOptions{
		Resources:      append(kube.BuiltinCatalog(), certEntry, issuerEntry, widgetEntry),
		TargetResource: certEntry,
		TargetName:     "web-cert",
		CRDFetchMode:   kube.CRDFetchRelated,
		FullSnapshot:   true,
	})
	if err != nil {
		t.Fatalf("FetchSnapshot() error = %v", err)
	}

	sawCert := false
	sawIssuer := false
	sawWidget := false
	for i := range snapshot.Dynamic {
		switch snapshot.Dynamic[i].Resource.Kind {
		case resource.Kind("Certificate"):
			sawCert = true
		case resource.Kind("Issuer"):
			sawIssuer = true
		case resource.Kind("Widget"):
			sawWidget = true
		}
	}
	if !sawCert || !sawIssuer {
		t.Fatalf("related CRD mode should include cert-manager resources, sawCert=%t sawIssuer=%t", sawCert, sawIssuer)
	}
	if sawWidget {
		t.Fatal("related CRD mode should not include unrelated custom groups")
	}
}

func TestFetchSnapshot_CRDFetchNoneSkipsCustom(t *testing.T) {
	t.Parallel()

	certEntry := kube.APIResource{
		Group:      "cert-manager.io",
		Version:    "v1",
		Resource:   "certificates",
		Kind:       resource.Kind("Certificate"),
		Namespaced: true,
		Custom:     true,
	}
	dyn := newDynamicClient(map[schema.GroupVersionResource]string{
		certEntry.GroupVersionResource(): "CertificateList",
	}, unstructuredObject("cert-manager.io/v1", "Certificate", "default", "web-cert"))

	client := kube.NewClientForInterfaces(kubefake.NewClientset(), dyn)
	snapshot, err := client.FetchSnapshot(context.Background(), "default", kube.SnapshotFetchOptions{
		Resources:      append(kube.BuiltinCatalog(), certEntry),
		TargetResource: certEntry,
		TargetName:     "web-cert",
		CRDFetchMode:   kube.CRDFetchNone,
		FullSnapshot:   true,
	})
	if err != nil {
		t.Fatalf("FetchSnapshot() error = %v", err)
	}
	for i := range snapshot.Dynamic {
		if snapshot.Dynamic[i].Resource.Custom {
			t.Fatalf("CRDFetchNone should skip custom resources, got %s", snapshot.Dynamic[i].Resource.Kind)
		}
	}
}

func TestFetchSnapshot_RelatedModeBuiltinTargetSkipsCustom(t *testing.T) {
	t.Parallel()

	certEntry := kube.APIResource{
		Group:      "cert-manager.io",
		Version:    "v1",
		Resource:   "certificates",
		Kind:       resource.Kind("Certificate"),
		Namespaced: true,
		Custom:     true,
	}
	dyn := newDynamicClient(map[schema.GroupVersionResource]string{
		certEntry.GroupVersionResource(): "CertificateList",
	}, unstructuredObject("cert-manager.io/v1", "Certificate", "default", "web-cert"))

	serviceEntry, ok := kube.LookupKind(resource.ReferenceKindService)
	if !ok {
		t.Fatal("service entry not found in catalog")
	}

	client := kube.NewClientForInterfaces(kubefake.NewClientset(), dyn)
	snapshot, err := client.FetchSnapshot(context.Background(), "default", kube.SnapshotFetchOptions{
		Resources:      append(kube.BuiltinCatalog(), certEntry),
		TargetResource: serviceEntry,
		TargetName:     "web",
		CRDFetchMode:   kube.CRDFetchRelated,
		FullSnapshot:   true,
	})
	if err != nil {
		t.Fatalf("FetchSnapshot() error = %v", err)
	}

	for i := range snapshot.Dynamic {
		if snapshot.Dynamic[i].Resource.Kind == resource.Kind("Certificate") {
			t.Fatal("builtin target should not include cert-manager resources in related mode")
		}
	}
}

func TestFetchSnapshot_PreservesLegacyMethods(t *testing.T) {
	t.Parallel()

	client := kube.NewClientForInterface(kubefake.NewClientset())
	if _, err := client.Fetch(context.Background(), "default"); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if _, err := client.FetchWithResources(context.Background(), "default", kube.BuiltinCatalog()); err != nil {
		t.Fatalf("FetchWithResources() error = %v", err)
	}
}
