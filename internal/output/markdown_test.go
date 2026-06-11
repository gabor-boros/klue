package output_test

import (
	"strings"
	"testing"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/output"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestRenderDiagnosisMarkdown(t *testing.T) {
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
	if err := output.RenderDiagnosisMarkdown(&b, d); err != nil {
		t.Fatalf("RenderDiagnosisMarkdown() error = %v", err)
	}

	out := b.String()
	for _, want := range []string{
		"# Diagnosis",
		"**Target**: Pod/default/web",
		"**Summary**: Container is in CrashLoopBackOff",
		"CrashLoopBackOff",
		"## Root cause",
		"> **[critical] Container is in CrashLoopBackOff**",
		"> It keeps crashing\\.",
		"## Findings",
		"### 1. Container is in CrashLoopBackOff",
		"| Severity | `critical` |",
		"| Confidence | 95% |",
		"## Resource chain",
		"| :--- | :--- | :--- |",
		"| 1 | Pod/default/web | Running |",
		"## Suggestions",
		"### 1. Check logs",
		"```sh",
		"kubectl logs web",
		"```",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderDiagnosisMarkdownLogEvidence(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "")
	logFinding := diagnose.Finding{
		ID:       "pod/crashloop",
		Title:    "Container crashed",
		Severity: diagnose.SeverityCritical,
		Resource: target,
		Evidence: []diagnose.Evidence{
			diagnose.NewEvidence(
				target,
				diagnose.EvidenceLog,
				`container "app" (previous): panic`,
				"panic: boom\n\nignored blank line above",
			),
		},
	}

	var b strings.Builder
	if err := output.RenderDiagnosisMarkdown(&b, diagnose.Diagnosis{
		Target:   target,
		Summary:  logFinding.Title,
		Findings: []diagnose.Finding{logFinding},
	}); err != nil {
		t.Fatalf("RenderDiagnosisMarkdown() error = %v", err)
	}

	out := b.String()
	for _, want := range []string{
		"#### Evidence",
		`- container "app" \(previous\): panic`,
		"```text",
		"panic: boom",
		"ignored blank line above",
		"```",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderDiagnosisMarkdownWithDebugSection(t *testing.T) {
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
	if err := output.RenderDiagnosisMarkdown(&b, d); err != nil {
		t.Fatalf("RenderDiagnosisMarkdown() error = %v", err)
	}

	out := b.String()
	for _, want := range []string{
		"## Debug",
		"<details>",
		"<summary>Debug details</summary>",
		"| Event window | 1h |",
		"| Log candidates | 1 |",
		"| Logs fetched | 1, errors: 0 |",
		"| Correlation | 1 findings corroborated, 0 findings suppressed |",
		"### Candidate details",
		"| Pod/default/web | app | previous | crashloop\\-backoff |",
		"</details>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderDiagnosisMarkdownEmptyDebug(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "")
	d := diagnose.Diagnosis{
		Target:  target,
		Summary: "ok",
		Debug:   &diagnose.DebugInfo{},
	}

	var b strings.Builder
	if err := output.RenderDiagnosisMarkdown(&b, d); err != nil {
		t.Fatalf("RenderDiagnosisMarkdown() error = %v", err)
	}

	if !strings.Contains(b.String(), "No debug details were recorded.") {
		t.Fatalf("expected empty debug message, got:\n%s", b.String())
	}
}

func TestRenderDiagnosisMarkdownEscaping(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web-*", "")
	rootCause := diagnose.Finding{
		Title:       "Use `kubectl` & check <logs>",
		Severity:    diagnose.SeverityWarning,
		Explanation: "Line one\nLine | two",
	}
	finding := diagnose.Finding{
		Title:       "Pipe | in title",
		Severity:    "`weird`",
		Confidence:  0.5,
		Explanation: "backticks ``` inside",
	}

	d := diagnose.Diagnosis{
		Target:    target,
		Summary:   "Summary with *emphasis* and _underscore_",
		RootCause: &rootCause,
		Findings:  []diagnose.Finding{finding},
		Chain: []diagnose.ChainStep{
			{Resource: target, State: ""},
		},
		Suggestions: []diagnose.Suggestion{
			{
				Title:       "Run `echo`",
				Explanation: "Use care with # headings",
				Command:     "echo '`nested`' fence",
			},
		},
	}

	var b strings.Builder
	if err := output.RenderDiagnosisMarkdown(&b, d); err != nil {
		t.Fatalf("RenderDiagnosisMarkdown() error = %v", err)
	}

	out := b.String()
	for _, want := range []string{
		"**Summary**: Summary with \\*emphasis\\* and \\_underscore\\_",
		"> **[warning] Use \\`kubectl\\` & check <logs>**",
		"> Line one",
		"> Line \\| two",
		"### 1. Pipe \\| in title",
		"| Severity | `` `weird` `` |",
		"backticks \\`\\`\\` inside",
		"| 1 | Pod/default/web\\-\\* | \\- |",
		"### 1. Run \\`echo\\`",
		"Use care with \\# headings",
		"```sh",
		"echo '`nested`' fence",
		"```",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderDiagnosisMarkdownMinimal(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "")
	d := diagnose.Diagnosis{
		Target:  target,
		Summary: "All good",
	}

	var b strings.Builder
	if err := output.RenderDiagnosisMarkdown(&b, d); err != nil {
		t.Fatalf("RenderDiagnosisMarkdown() error = %v", err)
	}

	out := b.String()
	for _, unwanted := range []string{
		"## Root cause",
		"## Findings",
		"## Resource chain",
		"## Suggestions",
		"## Debug",
	} {
		if strings.Contains(out, unwanted) {
			t.Errorf("minimal output should not contain %q\n---\n%s", unwanted, out)
		}
	}

	if !strings.HasPrefix(out, "# Diagnosis\n\n**Target**:") {
		t.Fatalf("unexpected minimal output:\n%s", out)
	}
}
