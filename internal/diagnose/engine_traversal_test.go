package diagnose_test

import (
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/internal/rules/builtin"
	"github.com/gabor-boros/klue/internal/rules/ingress"
	"github.com/gabor-boros/klue/internal/rules/service"
	"github.com/gabor-boros/klue/pkg/resource"
)

type recordingRule struct {
	id                 string
	kinds              []resource.Kind
	findingsByResource map[string][]diagnose.Finding
	evaluatedKeys      []string
}

func (r *recordingRule) ID() string                 { return r.id }
func (r *recordingRule) Description() string        { return r.id }
func (r *recordingRule) AppliesTo() []resource.Kind { return r.kinds }
func (r *recordingRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	r.evaluatedKeys = append(r.evaluatedKeys, node.Ref.Key())
	return r.findingsByResource[node.Ref.Key()]
}

func (r *recordingRule) evaluated(ref resource.Reference) bool {
	for _, key := range r.evaluatedKeys {
		if key == ref.Key() {
			return true
		}
	}
	return false
}

func serviceRef(name string) resource.Reference {
	return resource.NewReference(resource.ReferenceKindService, "v1", "default", name, "uid-"+name)
}

func serviceRefInNamespace(namespace, name, uid string) resource.Reference {
	return resource.NewReference(resource.ReferenceKindService, "v1", namespace, name, uid)
}

func configMapRef(name string) resource.Reference {
	return resource.NewReference(resource.ReferenceKindConfigMap, "v1", "default", name, "uid-"+name)
}

func certRequestRef(name string) resource.Reference {
	return resource.NewReference("CertificateRequest", "cert-manager.io/v1", "default", name, "uid-"+name)
}

func TestEngineDiagnose_StopsAfterFirstFindingLayer(t *testing.T) {
	t.Parallel()

	target := podRef("web")
	svc := serviceRef("web-svc")
	cfg := configMapRef("web-config")

	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{Ref: target, Status: resource.StatusRunning})
	builder.AddNode(graph.Node{Ref: svc, Status: resource.StatusHealthy})
	builder.AddNode(graph.Node{Ref: cfg, Status: resource.StatusHealthy})
	builder.AddEdge(graph.Edge{Kind: graph.EdgeSelectedBy, From: graph.Node{Ref: svc}, To: graph.Node{Ref: target}})
	builder.AddEdge(graph.Edge{Kind: graph.EdgeReferences, From: graph.Node{Ref: svc}, To: graph.Node{Ref: cfg}})

	rule := &recordingRule{
		id:    "recording",
		kinds: []resource.Kind{resource.KindAny},
		findingsByResource: map[string][]diagnose.Finding{
			svc.Key(): {
				{ID: "service/problem", Title: "service issue", Severity: diagnose.SeverityError, Confidence: 0.8, Resource: svc},
			},
			cfg.Key(): {
				{ID: "config/problem", Title: "config issue", Severity: diagnose.SeverityCritical, Confidence: 0.9, Resource: cfg},
			},
		},
	}

	engine := diagnose.NewEngine(rule)
	d := engine.Diagnose(diagnose.RuleContext{Graph: builder.Build(), Options: diagnose.DefaultDiagnoseOptions()}, target)

	if d.RootCause == nil {
		t.Fatal("RootCause = nil, want finding from first related layer")
	}
	if d.RootCause.ID != "service/problem" {
		t.Fatalf("RootCause.ID = %q, want %q", d.RootCause.ID, "service/problem")
	}
	if rule.evaluated(cfg) {
		t.Fatalf("config map node %q was evaluated, want traversal to stop before deeper layer", cfg.Display())
	}
	if len(d.Chain) != 2 {
		t.Fatalf("Chain length = %d, want 2 (target + first related layer)", len(d.Chain))
	}
}

func TestEngineDiagnose_HealthyAfterGraphExhausted(t *testing.T) {
	t.Parallel()

	target := podRef("web")
	svc := serviceRef("web-svc")
	cfg := configMapRef("web-config")

	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{Ref: target, Status: resource.StatusRunning})
	builder.AddNode(graph.Node{Ref: svc, Status: resource.StatusHealthy})
	builder.AddNode(graph.Node{Ref: cfg, Status: resource.StatusHealthy})
	builder.AddEdge(graph.Edge{Kind: graph.EdgeSelectedBy, From: graph.Node{Ref: svc}, To: graph.Node{Ref: target}})
	builder.AddEdge(graph.Edge{Kind: graph.EdgeReferences, From: graph.Node{Ref: svc}, To: graph.Node{Ref: cfg}})

	d := diagnose.NewEngine().Diagnose(diagnose.RuleContext{
		Graph:   builder.Build(),
		Options: diagnose.DefaultDiagnoseOptions(),
	}, target)

	if d.RootCause != nil {
		t.Fatalf("RootCause = %+v, want nil", d.RootCause)
	}
	if len(d.Chain) != 3 {
		t.Fatalf("Chain length = %d, want 3 when graph is exhausted", len(d.Chain))
	}

	wantSummary := fmt.Sprintf("%s appears healthy", target.Display())
	if d.Summary != wantSummary {
		t.Fatalf("Summary = %q, want %q", d.Summary, wantSummary)
	}
}

func TestEngineDiagnose_FallbackFindsDisconnectedNamespaceNode(t *testing.T) {
	t.Parallel()

	target := podRef("web")
	disconnectedSvc := serviceRef("mismatch-svc")

	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{Ref: target, Status: resource.StatusReady})
	builder.AddNode(graph.Node{Ref: disconnectedSvc, Status: resource.StatusHealthy})

	rule := &recordingRule{
		id:    "fallback",
		kinds: []resource.Kind{resource.KindAny},
		findingsByResource: map[string][]diagnose.Finding{
			disconnectedSvc.Key(): {
				{ID: "service/disconnected", Title: "disconnected service issue", Severity: diagnose.SeverityError, Confidence: 0.85, Resource: disconnectedSvc},
			},
		},
	}

	d := diagnose.NewEngine(rule).Diagnose(diagnose.RuleContext{
		Graph:   builder.Build(),
		Options: diagnose.DefaultDiagnoseOptions(),
	}, target)

	if d.RootCause == nil {
		t.Fatal("RootCause = nil, want fallback finding on disconnected service")
	}
	if d.RootCause.ID != "service/disconnected" {
		t.Fatalf("RootCause.ID = %q, want %q", d.RootCause.ID, "service/disconnected")
	}
	if len(d.Chain) != 2 {
		t.Fatalf("Chain length = %d, want 2 (target + disconnected finding node)", len(d.Chain))
	}
	if d.Chain[1].Resource.Key() != disconnectedSvc.Key() {
		t.Fatalf("Chain[1] = %s, want %s", d.Chain[1].Resource.Display(), disconnectedSvc.Display())
	}
}

func TestEngineDiagnose_FallbackRespectsNamespace(t *testing.T) {
	t.Parallel()

	target := podRef("web")
	otherNamespaceSvc := serviceRefInNamespace("other", "mismatch-svc", "uid-other-mismatch-svc")

	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{Ref: target, Status: resource.StatusReady})
	builder.AddNode(graph.Node{Ref: otherNamespaceSvc, Status: resource.StatusHealthy})

	rule := &recordingRule{
		id:    "namespace-scope",
		kinds: []resource.Kind{resource.KindAny},
		findingsByResource: map[string][]diagnose.Finding{
			otherNamespaceSvc.Key(): {
				{ID: "service/other-ns", Title: "other namespace issue", Severity: diagnose.SeverityError, Confidence: 0.8, Resource: otherNamespaceSvc},
			},
		},
	}

	d := diagnose.NewEngine(rule).Diagnose(diagnose.RuleContext{
		Graph:   builder.Build(),
		Options: diagnose.DefaultDiagnoseOptions(),
	}, target)

	if d.RootCause != nil {
		t.Fatalf("RootCause = %+v, want nil (fallback must stay in target namespace)", d.RootCause)
	}
	wantSummary := fmt.Sprintf("%s appears healthy", target.Display())
	if d.Summary != wantSummary {
		t.Fatalf("Summary = %q, want %q", d.Summary, wantSummary)
	}
	if rule.evaluated(otherNamespaceSvc) {
		t.Fatalf("other namespace node %q was evaluated, want namespace-scoped fallback", otherNamespaceSvc.Display())
	}
}

func TestEngineDiagnose_NoFallbackWhenConnectedComponentHasFindings(t *testing.T) {
	t.Parallel()

	target := podRef("web")
	connectedSvc := serviceRef("web-svc")
	disconnectedCfg := configMapRef("disconnected-config")

	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{Ref: target, Status: resource.StatusReady})
	builder.AddNode(graph.Node{Ref: connectedSvc, Status: resource.StatusHealthy})
	builder.AddNode(graph.Node{Ref: disconnectedCfg, Status: resource.StatusHealthy})
	builder.AddEdge(graph.Edge{Kind: graph.EdgeSelectedBy, From: graph.Node{Ref: connectedSvc}, To: graph.Node{Ref: target}})

	rule := &recordingRule{
		id:    "no-fallback-after-findings",
		kinds: []resource.Kind{resource.KindAny},
		findingsByResource: map[string][]diagnose.Finding{
			connectedSvc.Key(): {
				{ID: "connected/problem", Title: "connected issue", Severity: diagnose.SeverityWarning, Confidence: 0.6, Resource: connectedSvc},
			},
			disconnectedCfg.Key(): {
				{ID: "disconnected/problem", Title: "disconnected issue", Severity: diagnose.SeverityCritical, Confidence: 0.95, Resource: disconnectedCfg},
			},
		},
	}

	d := diagnose.NewEngine(rule).Diagnose(diagnose.RuleContext{
		Graph:   builder.Build(),
		Options: diagnose.DefaultDiagnoseOptions(),
	}, target)

	if d.RootCause == nil {
		t.Fatal("RootCause = nil, want connected-component finding")
	}
	if d.RootCause.ID != "connected/problem" {
		t.Fatalf("RootCause.ID = %q, want %q (fallback must not run once connected findings exist)", d.RootCause.ID, "connected/problem")
	}
	if rule.evaluated(disconnectedCfg) {
		t.Fatalf("disconnected node %q was evaluated, want no fallback when connected findings already exist", disconnectedCfg.Display())
	}
}

func TestEngineDiagnose_MaxDepthBoundsTraversal(t *testing.T) {
	t.Parallel()

	target := podRef("web")
	svc := serviceRef("web-svc")
	certReq := certRequestRef("web-certreq")

	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{Ref: target, Status: resource.StatusRunning})
	builder.AddNode(graph.Node{Ref: svc, Status: resource.StatusHealthy})
	builder.AddNode(graph.Node{Ref: certReq, Status: resource.StatusDegraded})
	builder.AddEdge(graph.Edge{Kind: graph.EdgeSelectedBy, From: graph.Node{Ref: svc}, To: graph.Node{Ref: target}})
	builder.AddEdge(graph.Edge{Kind: graph.EdgeReferences, From: graph.Node{Ref: svc}, To: graph.Node{Ref: certReq}})

	rule := &recordingRule{
		id:    "depth-check",
		kinds: []resource.Kind{resource.KindAny},
		findingsByResource: map[string][]diagnose.Finding{
			certReq.Key(): {
				{ID: "cert/problem", Title: "certificate request pending", Severity: diagnose.SeverityError, Confidence: 0.9, Resource: certReq},
			},
		},
	}

	engine := diagnose.NewEngine(rule)

	limited := diagnose.DefaultDiagnoseOptions()
	limited.MaxDepth = 1
	dLimited := engine.Diagnose(diagnose.RuleContext{Graph: builder.Build(), Options: limited}, target)
	if len(dLimited.Findings) != 0 {
		t.Fatalf("Findings = %d, want 0 when depth cap excludes depth-2 resource", len(dLimited.Findings))
	}
	if rule.evaluated(certReq) {
		t.Fatalf("depth-2 node %q was evaluated, want no fallback when traversal is not exhausted", certReq.Display())
	}
	if strings.Contains(dLimited.Summary, "appears healthy") {
		t.Fatalf("Summary = %q, want non-healthy summary for truncated traversal", dLimited.Summary)
	}

	unlimited := diagnose.DefaultDiagnoseOptions()
	unlimited.MaxDepth = 0
	dUnlimited := engine.Diagnose(diagnose.RuleContext{Graph: builder.Build(), Options: unlimited}, target)
	if dUnlimited.RootCause == nil {
		t.Fatal("RootCause = nil, want finding with unlimited traversal")
	}
	if dUnlimited.RootCause.ID != "cert/problem" {
		t.Fatalf("RootCause.ID = %q, want %q", dUnlimited.RootCause.ID, "cert/problem")
	}
}

func TestEngineDiagnose_PodSurfacesRelatedServiceTargetPortMismatch(t *testing.T) {
	t.Parallel()

	podObj := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "web-pod",
			UID:       "pod-uid",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "web",
					Ports: []corev1.ContainerPort{
						{ContainerPort: 8080},
					},
				},
			},
		},
	}
	pod := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web-pod", "pod-uid")

	svcObj := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "web-svc",
			UID:       "svc-uid",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "web"},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(9090),
				},
			},
		},
	}
	svc := resource.NewReference(resource.ReferenceKindService, "v1", "default", "web-svc", "svc-uid")

	podNode := graph.Node{Ref: pod, Object: podObj, Status: resource.StatusReady}
	svcNode := graph.Node{Ref: svc, Object: svcObj, Status: resource.StatusHealthy}

	builder := graph.NewBuilder()
	builder.AddNode(podNode)
	builder.AddNode(svcNode)
	builder.AddEdge(graph.Edge{Kind: graph.EdgeSelectedBy, From: svcNode, To: podNode})

	engine := diagnose.NewEngine(service.TargetPortMismatchRule{})
	d := engine.Diagnose(diagnose.RuleContext{Graph: builder.Build(), Options: diagnose.DefaultDiagnoseOptions()}, pod)

	if d.RootCause == nil {
		t.Fatal("RootCause = nil, want service target port mismatch finding")
	}
	if d.RootCause.ID != "service/target-port-mismatch" {
		t.Fatalf("RootCause.ID = %q, want %q", d.RootCause.ID, "service/target-port-mismatch")
	}
	if d.RootCause.Resource.Kind != resource.ReferenceKindService {
		t.Fatalf("RootCause.Resource.Kind = %q, want %q", d.RootCause.Resource.Kind, resource.ReferenceKindService)
	}
}

func TestEngineDiagnose_PodSurfacesDisconnectedServiceSelectorMismatch(t *testing.T) {
	t.Parallel()

	podObj := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "backend-demo",
			UID:       "pod-backend-demo",
			Labels:    map[string]string{"app": "backend"},
		},
	}
	pod := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "backend-demo", "pod-backend-demo")

	svcObj := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "mismatch-demo",
			UID:       "svc-mismatch-demo",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "frontend"},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
		},
	}
	svc := resource.NewReference(resource.ReferenceKindService, "v1", "default", "mismatch-demo", "svc-mismatch-demo")

	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{Ref: pod, Object: podObj, Status: resource.StatusReady})
	builder.AddNode(graph.Node{Ref: svc, Object: svcObj, Status: resource.StatusHealthy})
	// Intentionally no Service->Pod selector edge: this simulates selector mismatch.

	d := diagnose.NewEngine(service.SelectorMismatchRule{}).Diagnose(diagnose.RuleContext{
		Graph:   builder.Build(),
		Options: diagnose.DefaultDiagnoseOptions(),
	}, pod)

	if d.RootCause == nil {
		t.Fatal("RootCause = nil, want selector mismatch from disconnected service via fallback")
	}
	if d.RootCause.ID != "service/selector-mismatch" {
		t.Fatalf("RootCause.ID = %q, want %q", d.RootCause.ID, "service/selector-mismatch")
	}
	if d.RootCause.Resource.Kind != resource.ReferenceKindService {
		t.Fatalf("RootCause.Resource.Kind = %q, want %q", d.RootCause.Resource.Kind, resource.ReferenceKindService)
	}
	if len(d.Chain) != 2 {
		t.Fatalf("Chain length = %d, want 2 (target + disconnected service)", len(d.Chain))
	}
}

func TestEngineDiagnose_PodSurfacesRelatedCertificateRequestCondition(t *testing.T) {
	t.Parallel()

	pod := podRef("web")
	certReq := certRequestRef("web-certreq")

	certReqObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "cert-manager.io/v1",
			"kind":       "CertificateRequest",
			"metadata": map[string]any{
				"name":      "web-certreq",
				"namespace": "default",
				"uid":       "uid-web-certreq",
			},
			"status": map[string]any{
				"conditions": []any{
					map[string]any{
						"type":    "Ready",
						"status":  "False",
						"reason":  "Pending",
						"message": "Waiting for issuer to sign the request",
					},
				},
			},
		},
	}

	builder := graph.NewBuilder()
	podNode := graph.Node{Ref: pod, Status: resource.StatusRunning}
	certReqNode := graph.Node{Ref: certReq, Object: certReqObj, Status: resource.StatusDegraded}
	builder.AddNode(podNode)
	builder.AddNode(certReqNode)
	builder.AddEdge(graph.Edge{Kind: graph.EdgeReferences, From: podNode, To: certReqNode})

	engine := diagnose.NewEngine(builtin.FailedConditionRule{})
	d := engine.Diagnose(diagnose.RuleContext{Graph: builder.Build(), Options: diagnose.DefaultDiagnoseOptions()}, pod)

	if d.RootCause == nil {
		t.Fatal("RootCause = nil, want related certificate request condition finding")
	}
	if d.RootCause.ID != "builtin/failed-condition" {
		t.Fatalf("RootCause.ID = %q, want %q", d.RootCause.ID, "builtin/failed-condition")
	}
	if d.RootCause.Resource.Kind != "CertificateRequest" {
		t.Fatalf("RootCause.Resource.Kind = %q, want %q", d.RootCause.Resource.Kind, "CertificateRequest")
	}
}

func TestEngineDiagnose_IngressTraversesMissingTLSToCertManagerIssuer(t *testing.T) {
	t.Parallel()

	ingObj := networkingIngressWithTLS("default", "web", "web-tls")
	certObj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "cert-manager.io/v1",
		"kind":       "Certificate",
		"metadata": map[string]any{
			"name":      "web-cert",
			"namespace": "default",
			"uid":       "cert-uid",
		},
		"spec": map[string]any{
			"secretName": "web-tls",
			"issuerRef": map[string]any{
				"name": "letsencrypt-staging",
				"kind": "Issuer",
			},
		},
	}}
	issuerObj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "cert-manager.io/v1",
		"kind":       "Issuer",
		"metadata": map[string]any{
			"name":      "letsencrypt-staging",
			"namespace": "default",
			"uid":       "issuer-uid",
		},
		"status": map[string]any{
			"conditions": []any{
				map[string]any{
					"type":    "Ready",
					"status":  "False",
					"reason":  "InvalidConfig",
					"message": "missing DNS provider credentials",
				},
			},
		},
	}}

	certEntry := kube.APIResource{
		Group: "cert-manager.io", Version: "v1", Resource: "certificates",
		Kind: "Certificate", Namespaced: true, Custom: true,
	}
	issuerEntry := kube.APIResource{
		Group: "cert-manager.io", Version: "v1", Resource: "issuers",
		Kind: "Issuer", Namespaced: true, Custom: true,
	}

	snapshot := &kube.ResourceSnapshot{
		Namespace: "default",
		Ingresses: []networkingv1.Ingress{ingObj},
		Dynamic: []kube.DynamicObject{
			{Resource: certEntry, Object: certObj},
			{Resource: issuerEntry, Object: issuerObj},
		},
	}

	target := resource.NewReference(resource.ReferenceKindIngress, "networking.k8s.io/v1", "default", "web", "")
	engine := diagnose.NewEngine(
		ingress.TLSSecretMissingRule{},
		builtin.MissingReferenceRule{},
		builtin.FailedConditionRule{},
	)

	d := engine.Diagnose(diagnose.RuleContext{
		Graph:   snapshot.BuildGraph(),
		Options: diagnose.DefaultDiagnoseOptions(),
	}, target)

	if d.RootCause == nil {
		t.Fatal("RootCause = nil, want cert-manager issuer finding through placeholder traversal")
	}
	if d.RootCause.ID != "builtin/failed-condition" {
		t.Fatalf("RootCause.ID = %q, want %q", d.RootCause.ID, "builtin/failed-condition")
	}
	if d.RootCause.Resource.Kind != "Issuer" {
		t.Fatalf("RootCause.Resource.Kind = %q, want %q", d.RootCause.Resource.Kind, "Issuer")
	}
}

func TestEngineDiagnose_NoDuplicateMissingTLSFinding(t *testing.T) {
	t.Parallel()

	// An ingress references a TLS secret that no resource produces. The typed
	// ingress rule and the generic placeholder rule both describe the same
	// missing object, but traversal must stop at the ingress layer so the user
	// sees a single, rich finding rather than a duplicate.
	snapshot := &kube.ResourceSnapshot{
		Namespace: "default",
		Ingresses: []networkingv1.Ingress{networkingIngressWithTLS("default", "web", "web-tls")},
	}

	target := resource.NewReference(resource.ReferenceKindIngress, "networking.k8s.io/v1", "default", "web", "")
	engine := diagnose.NewEngine(
		ingress.TLSSecretMissingRule{},
		builtin.MissingReferenceRule{},
	)

	d := engine.Diagnose(diagnose.RuleContext{
		Graph:   snapshot.BuildGraph(),
		Options: diagnose.DefaultDiagnoseOptions(),
	}, target)

	if len(d.Findings) != 1 {
		t.Fatalf("Findings = %d, want exactly 1 for a single missing TLS secret", len(d.Findings))
	}
	if d.Findings[0].ID != "ingress/tls-secret-missing" {
		t.Fatalf("Findings[0].ID = %q, want the typed ingress rule", d.Findings[0].ID)
	}
	for _, finding := range d.Findings {
		if finding.ID == "builtin/missing-reference" {
			t.Fatal("found duplicate builtin/missing-reference finding for the same missing secret")
		}
	}
}

func networkingIngressWithTLS(namespace, name, secret string) networkingv1.Ingress {
	return networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{{SecretName: secret}},
		},
	}
}
