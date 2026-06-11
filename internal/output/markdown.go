// Package output renders diagnoses and graphs as human-readable text.
package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/gabor-boros/klue/internal/diagnose"
)

// RenderDiagnosisMarkdown writes a diagnosis to w as Markdown.
func RenderDiagnosisMarkdown(w io.Writer, d diagnose.Diagnosis) error {
	var b strings.Builder

	b.WriteString("# Diagnosis\n\n")

	fmt.Fprintf(&b, "**Target**: %s\n\n", mdTableCell(d.Target.Display()))
	fmt.Fprintf(&b, "**Summary**: %s\n\n", mdTableCell(d.Summary))

	if d.RootCause != nil {
		b.WriteString("\n## Root cause\n\n")
		fmt.Fprintf(
			&b,
			"> **[%s] %s**\n",
			mdInline(string(d.RootCause.Severity)),
			mdInline(d.RootCause.Title),
		)

		if d.RootCause.Explanation != "" {
			b.WriteString(">\n")
			writeBlockquoteMarkdown(&b, d.RootCause.Explanation)
		}
	}

	if len(d.Findings) > 0 {
		b.WriteString("\n## Findings\n\n")
		for i := range d.Findings {
			writeFindingMarkdown(&b, i+1, &d.Findings[i])
		}
	}

	if len(d.Chain) > 0 {
		b.WriteString("\n## Resource chain\n\n")
		b.WriteString("| # | Resource | State |\n")
		b.WriteString("| :--- | :--- | :--- |\n")

		for i, step := range d.Chain {
			fmt.Fprintf(
				&b,
				"| %d | %s | %s |\n",
				i+1,
				mdTableCell(step.Resource.Display()),
				mdTableCell(statusOrDash(string(step.State))),
			)
		}
	}

	if len(d.Suggestions) > 0 {
		b.WriteString("\n## Suggestions\n\n")
		for i, suggestion := range d.Suggestions {
			writeSuggestionMarkdown(&b, i+1, suggestion)
		}
	}

	if d.Debug != nil {
		writeDebugMarkdown(&b, d.Debug)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func writeFindingMarkdown(b *strings.Builder, index int, finding *diagnose.Finding) {
	fmt.Fprintf(b, "### %d. %s\n\n", index, mdInline(finding.Title))

	b.WriteString("| Field | Value |\n")
	b.WriteString("| :--- | :--- |\n")
	fmt.Fprintf(b, "| Severity | %s |\n", mdCodeSpan(string(finding.Severity)))
	fmt.Fprintf(b, "| Confidence | %.0f%% |\n", finding.Confidence*100)

	if finding.Explanation != "" {
		b.WriteString("\n")
		fmt.Fprintf(b, "%s\n\n", mdInline(finding.Explanation))
	}

	if len(finding.Evidence) > 0 {
		b.WriteString("#### Evidence\n\n")
		for _, ev := range finding.Evidence {
			writeEvidenceMarkdown(b, ev)
		}
	}
}

func writeEvidenceMarkdown(b *strings.Builder, ev diagnose.Evidence) {
	if ev.Message == "" && ev.Raw == "" {
		return
	}

	if ev.Message != "" {
		fmt.Fprintf(b, "- %s\n", mdInline(ev.Message))
	}

	if ev.Type == diagnose.EvidenceLog && ev.Raw != "" {
		lines := splitEvidenceLines(ev.Raw)
		if len(lines) == 0 {
			return
		}

		b.WriteString("\n")
		writeFencedBlockMarkdown(b, "text", strings.Join(lines, "\n"))
		b.WriteString("\n")
	}
}

func writeSuggestionMarkdown(b *strings.Builder, index int, suggestion diagnose.Suggestion) {
	fmt.Fprintf(b, "### %d. %s\n\n", index, mdInline(suggestion.Title))

	if suggestion.Explanation != "" {
		fmt.Fprintf(b, "%s\n\n", mdInline(suggestion.Explanation))
	}

	if suggestion.Command != "" {
		writeFencedBlockMarkdown(b, "sh", suggestion.Command)
		b.WriteString("\n")
	}
}

func writeDebugMarkdown(b *strings.Builder, debug *diagnose.DebugInfo) {
	b.WriteString("\n## Debug\n\n")
	b.WriteString("<details>\n")
	b.WriteString("<summary>Debug details</summary>\n\n")

	hasSummaryRows := debug.EventWindow != "" ||
		debug.LogCandidatesTotal > 0 ||
		debug.LogEntriesFetched > 0 ||
		debug.LogFetchErrors > 0 ||
		debug.CorrelatedFindings > 0 ||
		debug.SuppressedFindings > 0

	if hasSummaryRows {
		b.WriteString("| Field | Value |\n")
		b.WriteString("| :--- | :--- |\n")

		if debug.EventWindow != "" {
			fmt.Fprintf(b, "| Event window | %s |\n", mdTableCell(debug.EventWindow))
		}
		if debug.LogCandidatesTotal > 0 {
			fmt.Fprintf(b, "| Log candidates | %d |\n", debug.LogCandidatesTotal)
		}
		if debug.LogEntriesFetched > 0 || debug.LogFetchErrors > 0 {
			fmt.Fprintf(
				b,
				"| Logs fetched | %d, errors: %d |\n",
				debug.LogEntriesFetched,
				debug.LogFetchErrors,
			)
		}
		if debug.CorrelatedFindings > 0 || debug.SuppressedFindings > 0 {
			fmt.Fprintf(
				b,
				"| Correlation | %d findings corroborated, %d findings suppressed |\n",
				debug.CorrelatedFindings,
				debug.SuppressedFindings,
			)
		}
	}

	if len(debug.LogCandidates) > 0 {
		if hasSummaryRows {
			b.WriteString("\n")
		}

		b.WriteString("### Candidate details\n\n")
		b.WriteString("| Pod | Container | Run | Reason |\n")
		b.WriteString("| :--- | :--- | :--- | :--- |\n")

		for _, candidate := range debug.LogCandidates {
			run := "current"
			if candidate.Previous {
				run = "previous"
			}

			fmt.Fprintf(
				b,
				"| %s | %s | %s | %s |\n",
				mdTableCell(candidate.Pod),
				mdTableCell(candidate.Container),
				mdTableCell(run),
				mdTableCell(candidate.Reason),
			)
		}
	}

	if !hasSummaryRows && len(debug.LogCandidates) == 0 {
		b.WriteString("No debug details were recorded.\n")
	}

	b.WriteString("\n</details>\n")
}

func writeBlockquoteMarkdown(b *strings.Builder, s string) {
	for _, line := range strings.Split(s, "\n") {
		fmt.Fprintf(b, "> %s\n", mdInline(line))
	}
	b.WriteString("\n")
}

func writeFencedBlockMarkdown(b *strings.Builder, language string, body string) {
	fence := strings.Repeat("`", maxBacktickRun(body)+1)
	if len(fence) < 3 {
		fence = "```"
	}

	if language != "" {
		fmt.Fprintf(b, "%s%s\n", fence, language)
	} else {
		fmt.Fprintf(b, "%s\n", fence)
	}

	b.WriteString(body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		b.WriteByte('\n')
	}

	fmt.Fprintf(b, "%s\n", fence)
}

func mdTableCell(s string) string {
	s = mdInline(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}

func mdInline(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"{", "\\{",
		"}", "\\}",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		".", "\\.",
		"!", "\\!",
		"|", "\\|",
	)

	return replacer.Replace(s)
}

func mdCodeSpan(s string) string {
	fence := strings.Repeat("`", maxBacktickRun(s)+1)

	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return fence + " " + s + " " + fence
	}

	return fence + s + fence
}

func maxBacktickRun(s string) int {
	maxRun := 0
	currentRun := 0

	for _, r := range s {
		if r == '`' {
			currentRun++
			if currentRun > maxRun {
				maxRun = currentRun
			}
			continue
		}

		currentRun = 0
	}

	return maxRun
}
