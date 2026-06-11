package diagnose_test

import (
	"testing"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// stubRule is a configurable rule used to exercise the engine.
type stubRule struct {
	id       string
	kinds    []resource.Kind
	findings []diagnose.Finding
}

func (r stubRule) ID() string                 { return r.id }
func (r stubRule) Description() string        { return r.id }
func (r stubRule) AppliesTo() []resource.Kind { return r.kinds }
func (r stubRule) Evaluate(_ diagnose.RuleContext, _ *graph.Node) []diagnose.Finding {
	return r.findings
}

func podRef(name string) resource.Reference {
	return resource.NewReference(resource.ReferenceKindPod, "v1", "default", name, "uid-"+name)
}

func newGraphWithPod(name string) *graph.Graph {
	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{Ref: podRef(name), Status: resource.StatusRunning})
	return builder.Build()
}

func TestEngineDiagnose_RootCauseBySeverity(t *testing.T) {
	t.Parallel()

	target := podRef("web")

	warning := diagnose.Finding{ID: "a/warn", Title: "warn", Severity: diagnose.SeverityWarning, Confidence: 0.9, Resource: target}
	critical := diagnose.Finding{ID: "b/crit", Title: "crit", Severity: diagnose.SeverityCritical, Confidence: 0.5, Resource: target}

	engine := diagnose.NewEngine(
		stubRule{id: "a", kinds: []resource.Kind{resource.ReferenceKindPod}, findings: []diagnose.Finding{warning}},
		stubRule{id: "b", kinds: []resource.Kind{resource.ReferenceKindPod}, findings: []diagnose.Finding{critical}},
	)

	d := engine.Diagnose(diagnose.RuleContext{Graph: newGraphWithPod("web"), Options: diagnose.DefaultDiagnoseOptions()}, target)

	if d.RootCause == nil {
		t.Fatal("RootCause = nil, want critical finding")
	}
	if d.RootCause.ID != "b/crit" {
		t.Errorf("RootCause.ID = %q, want %q (severity must win)", d.RootCause.ID, "b/crit")
	}
	if len(d.Findings) != 2 {
		t.Errorf("Findings = %d, want 2", len(d.Findings))
	}
}

func TestEngineDiagnose_KindDispatch(t *testing.T) {
	t.Parallel()

	target := podRef("web")
	deployFinding := diagnose.Finding{ID: "d/x", Title: "deploy", Severity: diagnose.SeverityError, Resource: target}

	// Rule applies only to Deployments, so it must not fire for a Pod target.
	engine := diagnose.NewEngine(
		stubRule{id: "d", kinds: []resource.Kind{resource.ReferenceKindDeployment}, findings: []diagnose.Finding{deployFinding}},
	)

	d := engine.Diagnose(diagnose.RuleContext{Graph: newGraphWithPod("web"), Options: diagnose.DefaultDiagnoseOptions()}, target)

	if len(d.Findings) != 0 {
		t.Errorf("Findings = %d, want 0 (rule should not apply to pods)", len(d.Findings))
	}
}

func TestEngineDiagnose_Healthy(t *testing.T) {
	t.Parallel()

	target := podRef("web")
	engine := diagnose.NewEngine(
		stubRule{id: "a", kinds: []resource.Kind{resource.ReferenceKindPod}, findings: nil},
	)

	d := engine.Diagnose(diagnose.RuleContext{Graph: newGraphWithPod("web"), Options: diagnose.DefaultDiagnoseOptions()}, target)

	if d.RootCause != nil {
		t.Errorf("RootCause = %+v, want nil", d.RootCause)
	}
	if len(d.Chain) == 0 {
		t.Error("Chain is empty, want at least the target node")
	}
}

func TestEngineDiagnose_TargetNotFound(t *testing.T) {
	t.Parallel()

	engine := diagnose.NewEngine()
	d := engine.Diagnose(diagnose.RuleContext{Graph: graph.NewBuilder().Build(), Options: diagnose.DefaultDiagnoseOptions()}, podRef("missing"))

	if d.RootCause != nil {
		t.Errorf("RootCause = %+v, want nil", d.RootCause)
	}
	if d.Summary == "" {
		t.Error("Summary is empty, want a not-found message")
	}
}

func TestEngineDiagnose_RootCauseByCorroboration(t *testing.T) {
	t.Parallel()

	target := podRef("web")
	singleEvidence := diagnose.Finding{
		ID:         "a/warn",
		Title:      "single evidence",
		Severity:   diagnose.SeverityWarning,
		Confidence: 0.95,
		Resource:   target,
		Evidence: []diagnose.Evidence{
			diagnose.NewEvidence(target, diagnose.EvidenceStatus, "status", ""),
		},
	}
	corroborated := diagnose.Finding{
		ID:         "b/warn",
		Title:      "corroborated",
		Severity:   diagnose.SeverityWarning,
		Confidence: 0.8,
		Resource:   target,
		Evidence: []diagnose.Evidence{
			diagnose.NewEvidence(target, diagnose.EvidenceStatus, "status", ""),
			diagnose.NewEvidence(target, diagnose.EvidenceEvent, "event", "Failed"),
			diagnose.NewEvidence(target, diagnose.EvidenceLog, "log", "panic: boom"),
		},
	}

	engine := diagnose.NewEngine(
		stubRule{id: "a", kinds: []resource.Kind{resource.ReferenceKindPod}, findings: []diagnose.Finding{singleEvidence}},
		stubRule{id: "b", kinds: []resource.Kind{resource.ReferenceKindPod}, findings: []diagnose.Finding{corroborated}},
	)

	d := engine.Diagnose(diagnose.RuleContext{Graph: newGraphWithPod("web"), Options: diagnose.DefaultDiagnoseOptions()}, target)
	if d.RootCause == nil {
		t.Fatal("RootCause = nil, want corroborated finding")
	}
	if d.RootCause.ID != "b/warn" {
		t.Fatalf("RootCause.ID = %q, want corroborated finding to rank first", d.RootCause.ID)
	}
}

func TestEngineDiagnose_SuppressesRedundantBuiltinWarningEvents(t *testing.T) {
	t.Parallel()

	target := podRef("web")
	typed := diagnose.Finding{
		ID:         "pod/probe-failure",
		Title:      "typed",
		Severity:   diagnose.SeverityWarning,
		Confidence: 0.7,
		Resource:   target,
		Evidence: []diagnose.Evidence{
			diagnose.NewEvidence(target, diagnose.EvidenceEvent, "Readiness probe failed", "Unhealthy"),
		},
	}
	builtinWarning := diagnose.Finding{
		ID:         "builtin/warning-events",
		Title:      "warning event",
		Severity:   diagnose.SeverityWarning,
		Confidence: 0.4,
		Resource:   target,
		Evidence: []diagnose.Evidence{
			diagnose.NewEvidence(target, diagnose.EvidenceEvent, "Readiness probe failed", "Unhealthy"),
		},
	}

	engine := diagnose.NewEngine(
		stubRule{id: "typed", kinds: []resource.Kind{resource.ReferenceKindPod}, findings: []diagnose.Finding{typed}},
		stubRule{id: "builtin", kinds: []resource.Kind{resource.ReferenceKindPod}, findings: []diagnose.Finding{builtinWarning}},
	)

	d := engine.Diagnose(diagnose.RuleContext{
		Graph: newGraphWithPod("web"),
		Options: diagnose.DiagnoseOptions{
			Debug: true,
		},
	}, target)
	if len(d.Findings) != 1 {
		t.Fatalf("Findings = %d, want 1 after dedupe", len(d.Findings))
	}
	if d.Findings[0].ID != "pod/probe-failure" {
		t.Fatalf("Findings[0].ID = %q, want typed finding", d.Findings[0].ID)
	}
	if d.Debug == nil || d.Debug.SuppressedFindings == 0 {
		t.Fatalf("Debug = %+v, want suppressed findings metadata", d.Debug)
	}
}

func TestEngineDiagnose_PrefersUpstreamRelatedFindingOnTie(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindService, "v1", "default", "web", "uid-svc")
	pod := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web-pod", "uid-pod")

	builder := graph.NewBuilder()
	builder.AddNode(graph.Node{Ref: target, Status: resource.StatusDegraded})
	builder.AddNode(graph.Node{Ref: pod, Status: resource.StatusDegraded})
	builder.AddEdge(graph.Edge{
		Kind: graph.EdgeSelectedBy,
		From: graph.Node{Ref: target},
		To:   graph.Node{Ref: pod},
	})
	g := builder.Build()

	serviceSymptom := diagnose.Finding{
		ID:         "service/no-endpoints",
		Title:      "service symptom",
		Severity:   diagnose.SeverityWarning,
		Confidence: 0.5,
		Resource:   target,
		Evidence: []diagnose.Evidence{
			diagnose.NewEvidence(target, diagnose.EvidenceStatus, "status", ""),
		},
	}
	podCause := diagnose.Finding{
		ID:         "pod/cause",
		Title:      "pod cause",
		Severity:   diagnose.SeverityWarning,
		Confidence: 0.5,
		Resource:   pod,
		Evidence: []diagnose.Evidence{
			diagnose.NewEvidence(pod, diagnose.EvidenceStatus, "status", ""),
		},
	}

	engine := diagnose.NewEngine(
		stubRule{
			id:    "service",
			kinds: []resource.Kind{resource.ReferenceKindService},
			findings: []diagnose.Finding{
				serviceSymptom,
				podCause,
			},
		},
	)
	d := engine.Diagnose(diagnose.RuleContext{Graph: g, Options: diagnose.DefaultDiagnoseOptions()}, target)
	if d.RootCause == nil {
		t.Fatal("RootCause = nil, want pod-related finding")
	}
	if d.RootCause.ID != "pod/cause" {
		t.Fatalf("RootCause.ID = %q, want %q", d.RootCause.ID, "pod/cause")
	}
}
