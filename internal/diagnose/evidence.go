package diagnose

import "github.com/gabor-boros/klue/pkg/resource"

// EvidenceType is a short, human-readable category describing where a piece of
// evidence came from (for example a status condition, an event, or a missing
// reference). It is informational and not used for control flow.
type EvidenceType string

// Common evidence categories. Rules may use other descriptive values; these
// cover the recurring sources shared across multiple rules.
const (
	EvidenceCondition EvidenceType = "Condition"
	EvidenceEvent     EvidenceType = "Event"
	EvidenceStatus    EvidenceType = "Status"
	EvidenceReference EvidenceType = "Reference"
	EvidenceSpec      EvidenceType = "Spec"
	EvidenceLog       EvidenceType = "Log"
)

// Evidence captures a single observation that supports a finding. It records
// where the observation came from and a human-readable message, optionally
// preserving the raw source data.
type Evidence struct {
	Source  resource.Reference `json:"source"`
	Type    EvidenceType       `json:"type"`
	Message string             `json:"message,omitempty"`
	Raw     string             `json:"raw,omitempty"`
}

// NewEvidence creates a new Evidence from the given source, type, message and
// raw payload.
func NewEvidence(source resource.Reference, evidenceType EvidenceType, message, raw string) Evidence {
	return Evidence{
		Source:  source,
		Type:    evidenceType,
		Message: message,
		Raw:     raw,
	}
}
