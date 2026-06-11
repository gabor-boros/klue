package output_test

import (
	"strings"
	"testing"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/output"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestRenderDiagnosisFormat_Unsupported(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "")
	d := diagnose.Diagnosis{Target: target, Summary: "ok"}

	var unsupported strings.Builder
	if err := output.RenderDiagnosisFormat(&unsupported, d, "yaml"); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestRenderDiagnosisFormat_Text(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "")
	d := diagnose.Diagnosis{Target: target, Summary: "ok"}

	var text strings.Builder
	if err := output.RenderDiagnosisFormat(&text, d, "text"); err != nil {
		t.Fatalf("RenderDiagnosisFormat(text) error = %v", err)
	}
	if !strings.Contains(text.String(), "Summary: ok") {
		t.Fatalf("text output = %q", text.String())
	}
}

func TestRenderDiagnosisFormat_JSON(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "")
	d := diagnose.Diagnosis{Target: target, Summary: "ok"}

	var jsonOut strings.Builder
	if err := output.RenderDiagnosisFormat(&jsonOut, d, "json"); err != nil {
		t.Fatalf("RenderDiagnosisFormat(json) error = %v", err)
	}
	if !strings.Contains(jsonOut.String(), `"summary": "ok"`) {
		t.Fatalf("json output = %q", jsonOut.String())
	}

	var unsupported strings.Builder
	if err := output.RenderDiagnosisFormat(&unsupported, d, "yaml"); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}
