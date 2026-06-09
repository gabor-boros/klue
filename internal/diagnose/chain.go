package diagnose

import "github.com/gabor-boros/klue/pkg/resource"

// ChainStep is a single resource in the causal chain rendered for a diagnosis,
// together with its observed state and the explanation recorded for it.
type ChainStep struct {
	Resource    resource.Reference `json:"resource"`
	State       resource.Status    `json:"state,omitempty"`
	Explanation string             `json:"explanation,omitempty"`
}
