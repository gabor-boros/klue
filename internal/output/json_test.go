package output_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/output"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestRenderDiagnosisJSON(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "")
	rootCause := diagnose.Finding{
		ID:          "pod/crashloop",
		Title:       "Container is in CrashLoopBackOff",
		Severity:    diagnose.SeverityCritical,
		Confidence:  0.95,
		Resource:    target,
		Explanation: "It keeps crashing.",
	}

	d := diagnose.Diagnosis{
		Target:    target,
		Summary:   rootCause.Title,
		RootCause: &rootCause,
		Findings:  []diagnose.Finding{rootCause},
	}

	var b strings.Builder
	if err := output.RenderDiagnosisJSON(&b, d); err != nil {
		t.Fatalf("RenderDiagnosisJSON() error = %v", err)
	}

	var decoded diagnose.Diagnosis
	if err := json.Unmarshal([]byte(b.String()), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v\n---\n%s", err, b.String())
	}

	if decoded.Summary != d.Summary {
		t.Errorf("Summary = %q, want %q", decoded.Summary, d.Summary)
	}
	if decoded.RootCause == nil || decoded.RootCause.ID != "pod/crashloop" {
		t.Fatalf("decoded root cause = %#v, want pod/crashloop", decoded.RootCause)
	}
}

func TestRenderDiagnosisJSONWithDebug(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "")
	d := diagnose.Diagnosis{
		Target:  target,
		Summary: "ok",
		Debug: &diagnose.DebugInfo{
			EventWindow:        "1h",
			LogCandidatesTotal: 2,
		},
	}

	var b strings.Builder
	if err := output.RenderDiagnosisJSON(&b, d); err != nil {
		t.Fatalf("RenderDiagnosisJSON() error = %v", err)
	}

	if !strings.Contains(b.String(), `"debug":`) {
		t.Fatalf("json output missing debug payload: %s", b.String())
	}
}
