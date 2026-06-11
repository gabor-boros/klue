package diagnose

import "github.com/gabor-boros/klue/pkg/resource"

type DebugLogCandidate struct {
	Pod       string `json:"pod"`
	Container string `json:"container"`
	Previous  bool   `json:"previous,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type DebugInfo struct {
	EventWindow        string              `json:"eventWindow,omitempty"`
	RulesSelected      []string            `json:"rulesSelected,omitempty"`
	LogCandidates      []DebugLogCandidate `json:"logCandidates,omitempty"`
	LogCandidatesTotal int                 `json:"logCandidatesTotal,omitempty"`
	LogEntriesFetched  int                 `json:"logEntriesFetched,omitempty"`
	LogFetchErrors     int                 `json:"logFetchErrors,omitempty"`
	CorrelatedFindings int                 `json:"correlatedFindings,omitempty"`
	SuppressedFindings int                 `json:"suppressedFindings,omitempty"`
}

type Diagnosis struct {
	Target      resource.Reference `json:"target"`
	Summary     string             `json:"summary"`
	RootCause   *Finding           `json:"rootCause,omitempty"`
	Findings    []Finding          `json:"findings,omitempty"`
	Chain       []ChainStep        `json:"chain,omitempty"`
	Suggestions []Suggestion       `json:"suggestions,omitempty"`
	Debug       *DebugInfo         `json:"debug,omitempty"`
}
