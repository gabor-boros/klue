package kube_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

// customResource builds an unstructured custom resource with the given
// status.conditions so the generic status mapping can be exercised.
func customResource(name, uid string, conditions []any) *unstructured.Unstructured {
	obj := unstructuredObject("cert-manager.io/v1", "Certificate", "default", name)
	obj.SetUID(typesUID(uid))
	if conditions != nil {
		_ = unstructured.SetNestedSlice(obj.Object, conditions, "status", "conditions")
	}
	return obj
}

func TestBuildGraphCustomResourceNodeAndStatus(t *testing.T) {
	t.Parallel()

	certEntry := kube.APIResource{
		Group: "cert-manager.io", Version: "v1", Resource: "certificates",
		Kind: "Certificate", Namespaced: true, Custom: true,
	}

	failing := customResource("web-cert", "cert-uid", []any{
		map[string]any{"type": "Ready", "status": "False", "reason": "Issuing"},
	})

	snapshot := &kube.ResourceSnapshot{
		Namespace: "default",
		Dynamic:   []kube.DynamicObject{{Resource: certEntry, Object: failing}},
	}

	g := snapshot.BuildGraph()

	ref := resource.NewReference("Certificate", "cert-manager.io/v1", "default", "web-cert", "cert-uid")
	node, found := g.FindByRef(ref)
	if !found {
		t.Fatal("custom resource Certificate was not added as a graph node")
	}
	if node.Status != resource.StatusDegraded {
		t.Errorf("Certificate status = %s, want %s (failing Ready condition)", node.Status, resource.StatusDegraded)
	}
}

func TestBuildGraphCustomResourceOwnsBuiltin(t *testing.T) {
	t.Parallel()

	certEntry := kube.APIResource{
		Group: "cert-manager.io", Version: "v1", Resource: "certificates",
		Kind: "Certificate", Namespaced: true, Custom: true,
	}

	owner := customResource("web-cert", "cert-uid", nil)

	// A built-in Secret owned by the custom resource, proving owner edges are
	// wired uniformly across typed built-ins and dynamic custom resources.
	ownedSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "web-cert-tls",
			UID:       typesUID("secret-uid"),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "cert-manager.io/v1",
				Kind:       "Certificate",
				Name:       "web-cert",
				UID:        typesUID("cert-uid"),
			}},
		},
	}

	snapshot := &kube.ResourceSnapshot{
		Namespace: "default",
		Secrets:   []corev1.Secret{ownedSecret},
		Dynamic:   []kube.DynamicObject{{Resource: certEntry, Object: owner}},
	}

	g := snapshot.BuildGraph()

	if !edgeExists(g.GetEdges(), graph.EdgeOwns, "web-cert", "web-cert-tls") {
		t.Error("expected owner edge Certificate -> Secret from cross-type owner references")
	}
}

func TestBuildGraphCertManagerRelationshipsWithMissingTargets(t *testing.T) {
	t.Parallel()

	certEntry := kube.APIResource{
		Group: "cert-manager.io", Version: "v1", Resource: "certificates",
		Kind: "Certificate", Namespaced: true, Custom: true,
	}
	issuerEntry := kube.APIResource{
		Group: "cert-manager.io", Version: "v1", Resource: "issuers",
		Kind: "Issuer", Namespaced: true, Custom: true,
	}

	certificate := customResource("web-cert", "cert-uid", nil)
	_ = unstructured.SetNestedField(certificate.Object, "web-tls", "spec", "secretName")
	_ = unstructured.SetNestedMap(certificate.Object, map[string]any{
		"name": "letsencrypt-staging",
		"kind": "Issuer",
	}, "spec", "issuerRef")

	issuer := unstructuredObject("cert-manager.io/v1", "Issuer", "default", "letsencrypt-staging")
	issuer.SetUID(typesUID("issuer-uid"))
	_ = unstructured.SetNestedMap(issuer.Object, map[string]any{"name": "acme-account-key"}, "spec", "acme", "privateKeySecretRef")

	ingress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web", UID: typesUID("ing-uid")},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{{SecretName: "web-tls"}},
		},
	}

	snapshot := &kube.ResourceSnapshot{
		Namespace: "default",
		Ingresses: []networkingv1.Ingress{ingress},
		Dynamic: []kube.DynamicObject{
			{Resource: certEntry, Object: certificate},
			{Resource: issuerEntry, Object: issuer},
		},
	}

	g := snapshot.BuildGraph()
	edges := g.GetEdges()

	if !edgeExists(edges, graph.EdgeUsesSecret, "web", "web-tls") {
		t.Fatal("expected Ingress -> Secret edge via spec.tls[].secretName")
	}
	if !edgeExists(edges, graph.EdgeProduces, "web-cert", "web-tls") {
		t.Fatal("expected Certificate -> Secret producer edge via spec.secretName")
	}
	if !edgeExists(edges, graph.EdgeReferences, "web-cert", "letsencrypt-staging") {
		t.Fatal("expected Certificate -> Issuer edge via spec.issuerRef")
	}
	if !edgeExists(edges, graph.EdgeUsesSecret, "letsencrypt-staging", "acme-account-key") {
		t.Fatal("expected Issuer -> Secret edge via privateKeySecretRef")
	}

	tlsRef := resource.NewReference(resource.ReferenceKindSecret, "v1", "default", "web-tls", "")
	tlsNode, found := g.FindByRef(tlsRef)
	if !found {
		t.Fatal("expected placeholder node for missing TLS secret")
	}
	if tlsNode.Status != resource.StatusMissing {
		t.Fatalf("TLS secret placeholder status = %s, want %s", tlsNode.Status, resource.StatusMissing)
	}
	if !tlsNode.IsPlaceholder() {
		t.Fatal("expected TLS secret node to be marked as placeholder")
	}
}
