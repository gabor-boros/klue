package diagnose

import "github.com/gabor-boros/klue/pkg/resource"

// Confidence expresses how strongly a finding is believed to explain the target
// problem, on a scale from 0.0 (least) to 1.0 (most) confident.
type Confidence float64

// Finding is a single diagnosed problem with the evidence and remediation
// suggestions that support it.
type Finding struct {
	ID            string             `json:"id"`
	Title         string             `json:"title"`
	Severity      Severity           `json:"severity"`
	Confidence    Confidence         `json:"confidence"`
	Corroboration int                `json:"corroboration,omitempty"`
	Resource      resource.Reference `json:"resource"`
	Evidence      []Evidence         `json:"evidence,omitempty"`
	Explanation   string             `json:"explanation,omitempty"`
	Suggestions   []Suggestion       `json:"suggestions,omitempty"`
}
