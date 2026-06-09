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
