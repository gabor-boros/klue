package diagnose

import "github.com/gabor-boros/klue/pkg/resource"

type Diagnosis struct {
	Target      resource.Reference `json:"target"`
	Summary     string             `json:"summary"`
	RootCause   *Finding           `json:"rootCause,omitempty"`
	Findings    []Finding          `json:"findings,omitempty"`
	Chain       []ChainStep        `json:"chain,omitempty"`
	Suggestions []Suggestion       `json:"suggestions,omitempty"`
}
