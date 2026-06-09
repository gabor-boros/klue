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
