package output

import (
	"encoding/json"
	"fmt"
	"io"

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
