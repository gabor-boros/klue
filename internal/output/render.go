package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/gabor-boros/klue/internal/diagnose"
)

// RenderDiagnosisFormat writes a diagnosis using the requested output format.
// Supported formats are "text" and "json".
func RenderDiagnosisFormat(w io.Writer, d diagnose.Diagnosis, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		return RenderDiagnosisText(w, d)
	case "json":
		return RenderDiagnosisJSON(w, d)
	case "markdown":
		return RenderDiagnosisMarkdown(w, d)
	default:
		return fmt.Errorf("unsupported output format %q (supported: text, json, markdown)", format)
	}
}

func statusOrDash(status string) string {
	if status == "" {
		return "-"
	}
	return status
}

func splitEvidenceLines(raw string) []string {
	if raw == "" {
		return nil
	}
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
