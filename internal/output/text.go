// Package output renders diagnoses and graphs as human-readable text.
package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/gabor-boros/klue/internal/diagnose"
)

// RenderDiagnosisText writes a diagnosis to w as plain text.
func RenderDiagnosisText(w io.Writer, d diagnose.Diagnosis) error {
	var b strings.Builder

	fmt.Fprintf(&b, "Target:  %s\n", d.Target.Display())
	fmt.Fprintf(&b, "Summary: %s\n", d.Summary)

	if d.RootCause != nil {
		fmt.Fprintf(&b, "\nRoot cause: [%s] %s\n", d.RootCause.Severity, d.RootCause.Title)
		if d.RootCause.Explanation != "" {
			fmt.Fprintf(&b, "  %s\n", d.RootCause.Explanation)
		}
	}

	if len(d.Findings) > 0 {
		b.WriteString("\nFindings:\n")
		for i := range d.Findings {
			writeFindingText(&b, &d.Findings[i])
		}
	}

	if len(d.Chain) > 0 {
		b.WriteString("\nResource chain:\n")
		for _, step := range d.Chain {
			fmt.Fprintf(&b, "  - %s (%s)\n", step.Resource.Display(), statusOrDash(string(step.State)))
		}
	}

	if len(d.Suggestions) > 0 {
		b.WriteString("\nSuggestions:\n")
		for _, suggestion := range d.Suggestions {
			writeSuggestionText(&b, suggestion)
		}
	}

	if d.Debug != nil {
		b.WriteString("\nDebug:\n")
		if d.Debug.EventWindow != "" {
			fmt.Fprintf(&b, "  event window: %s\n", d.Debug.EventWindow)
		}
		if d.Debug.LogCandidatesTotal > 0 {
			fmt.Fprintf(&b, "  log candidates: %d\n", d.Debug.LogCandidatesTotal)
		}
		if d.Debug.LogEntriesFetched > 0 || d.Debug.LogFetchErrors > 0 {
			fmt.Fprintf(&b, "  logs fetched: %d (errors: %d)\n", d.Debug.LogEntriesFetched, d.Debug.LogFetchErrors)
		}
		if d.Debug.CorrelatedFindings > 0 || d.Debug.SuppressedFindings > 0 {
			fmt.Fprintf(&b, "  correlation: %d findings corroborated, %d findings suppressed\n", d.Debug.CorrelatedFindings, d.Debug.SuppressedFindings)
		}
		if len(d.Debug.LogCandidates) > 0 {
			b.WriteString("  candidate details:\n")
			for _, candidate := range d.Debug.LogCandidates {
				run := "current"
				if candidate.Previous {
					run = "previous"
				}
				fmt.Fprintf(&b, "    - %s container=%s (%s)", candidate.Pod, candidate.Container, run)
				if candidate.Reason != "" {
					fmt.Fprintf(&b, " reason=%s", candidate.Reason)
				}
				b.WriteByte('\n')
			}
		}
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func writeEvidenceText(b *strings.Builder, ev diagnose.Evidence) {
	if ev.Message == "" && ev.Raw == "" {
		return
	}

	if ev.Type == diagnose.EvidenceLog {
		if ev.Message != "" {
			fmt.Fprintf(b, "      evidence: %s\n", ev.Message)
		}
		for _, line := range splitEvidenceLines(ev.Raw) {
			fmt.Fprintf(b, "      log: %s\n", line)
		}
		return
	}

	if ev.Message != "" {
		fmt.Fprintf(b, "      evidence: %s\n", ev.Message)
	}
}

func writeFindingText(b *strings.Builder, finding *diagnose.Finding) {
	fmt.Fprintf(b, "  - [%s] %s (confidence %.0f%%)\n", finding.Severity, finding.Title, finding.Confidence*100)
	if finding.Explanation != "" {
		fmt.Fprintf(b, "      %s\n", finding.Explanation)
	}
	for _, ev := range finding.Evidence {
		writeEvidenceText(b, ev)
	}
}

func writeSuggestionText(b *strings.Builder, suggestion diagnose.Suggestion) {
	fmt.Fprintf(b, "  - %s\n", suggestion.Title)
	if suggestion.Command != "" {
		fmt.Fprintf(b, "      $ %s\n", suggestion.Command)
	}
	if suggestion.Explanation != "" {
		fmt.Fprintf(b, "      %s\n", suggestion.Explanation)
	}
}
