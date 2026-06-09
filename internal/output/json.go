package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gabor-boros/klue/internal/diagnose"
)

// RenderDiagnosisJSON writes a diagnosis to w as indented JSON.
func RenderDiagnosisJSON(w io.Writer, d diagnose.Diagnosis) error {
	encoded, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("encode diagnosis: %w", err)
	}
	encoded = append(encoded, '\n')
	_, err = w.Write(encoded)
	return err
}

// RenderDiagnosisFormat writes a diagnosis using the requested output format.
// Supported formats are "text" and "json".
func RenderDiagnosisFormat(w io.Writer, d diagnose.Diagnosis, format string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		return RenderDiagnosis(w, d)
	case "json":
		return RenderDiagnosisJSON(w, d)
	default:
		return fmt.Errorf("unsupported output format %q (supported: text, json)", format)
	}
}
