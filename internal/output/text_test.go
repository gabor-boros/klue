package output_test

import (
	"strings"
	"testing"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/output"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestRenderDiagnosis(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "")
	rootCause := diagnose.Finding{
		ID:          "pod/crashloop",
		Title:       "Container is in CrashLoopBackOff",
		Severity:    diagnose.SeverityCritical,
		Confidence:  0.95,
		Resource:    target,
		Explanation: "It keeps crashing.",
		Suggestions: []diagnose.Suggestion{{Title: "Check logs", Command: "kubectl logs web"}},
	}

	d := diagnose.Diagnosis{
		Target:      target,
		Summary:     rootCause.Title,
		RootCause:   &rootCause,
		Findings:    []diagnose.Finding{rootCause},
		Chain:       []diagnose.ChainStep{{Resource: target, State: "Running"}},
		Suggestions: rootCause.Suggestions,
	}

	var b strings.Builder
	if err := output.RenderDiagnosis(&b, d); err != nil {
		t.Fatalf("RenderDiagnosis() error = %v", err)
	}

	out := b.String()
	logFinding := diagnose.Finding{
		ID:       "pod/crashloop",
		Title:    "Container crashed",
		Severity: diagnose.SeverityCritical,
		Resource: target,
		Evidence: []diagnose.Evidence{
			diagnose.NewEvidence(target, diagnose.EvidenceLog, `container "app" (previous): panic`, "panic: boom"),
		},
	}

	var logBuilder strings.Builder
	if err := output.RenderDiagnosis(&logBuilder, diagnose.Diagnosis{
		Target:   target,
		Summary:  logFinding.Title,
		Findings: []diagnose.Finding{logFinding},
	}); err != nil {
		t.Fatalf("RenderDiagnosis() log evidence error = %v", err)
	}
	if !strings.Contains(logBuilder.String(), "log: panic: boom") {
		t.Fatalf("log evidence output missing log line:\n%s", logBuilder.String())
	}

	for _, want := range []string{
		"Pod/default/web",
		"Root cause:",
		"CrashLoopBackOff",
		"Suggestions:",
		"kubectl logs web",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderDiagnosisWithDebugSection(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "")
	d := diagnose.Diagnosis{
		Target:  target,
		Summary: "debug",
		Debug: &diagnose.DebugInfo{
			EventWindow:        "1h",
			LogCandidatesTotal: 1,
			LogEntriesFetched:  1,
			LogFetchErrors:     0,
			CorrelatedFindings: 1,
			LogCandidates: []diagnose.DebugLogCandidate{
				{Pod: target.Display(), Container: "app", Previous: true, Reason: "crashloop-backoff"},
			},
		},
	}

	var b strings.Builder
	if err := output.RenderDiagnosis(&b, d); err != nil {
		t.Fatalf("RenderDiagnosis() error = %v", err)
	}

	out := b.String()
	for _, want := range []string{
		"Debug:",
		"event window: 1h",
		"log candidates: 1",
		"candidate details:",
		"reason=crashloop-backoff",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\n---\n%s", want, out)
		}
	}
}
