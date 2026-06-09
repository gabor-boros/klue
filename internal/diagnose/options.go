package diagnose

import "time"

// Default values for DiagnoseOptions.
const (
	// DefaultMaxDepth configures exhaustive graph traversal from the target
	// resource. A value of zero means unlimited traversal depth.
	DefaultMaxDepth = 0

	// DefaultEventWindow is the default age threshold for events to be
	// considered relevant to a diagnosis.
	DefaultEventWindow = time.Hour

	// DefaultTerminatingGracePeriod is how long an object may carry a deletion
	// timestamp before it is considered stuck terminating.
	DefaultTerminatingGracePeriod = 5 * time.Minute

	// DefaultLeaseStaleMultiplier is how many lease durations may elapse past
	// the last renewal before the holder is considered stale.
	DefaultLeaseStaleMultiplier = 4
)

// DiagnoseOptions controls how a diagnosis is performed. Keeping these as
// explicit inputs (including the reference time) makes diagnoses deterministic
// and testable.
type DiagnoseOptions struct {
	// Namespace scopes the diagnosis to a single namespace.
	Namespace string

	// MaxDepth bounds how many graph hops are traversed from the target.
	// Zero or negative values mean unlimited traversal depth.
	MaxDepth int

	// EventWindow bounds how old an event may be to be considered relevant.
	EventWindow time.Duration

	// Now is the reference time used for any age or recency calculations.
	// Callers should set this explicitly to keep diagnoses deterministic.
	Now time.Time

	// TerminatingGracePeriod bounds how long an object may remain terminating
	// before the terminating-stuck rule reports it.
	TerminatingGracePeriod time.Duration

	// LeaseStaleMultiplier bounds how many lease durations may elapse past the
	// last renewal before the lease stale rule reports it. Zero or negative
	// values use DefaultLeaseStaleMultiplier.
	LeaseStaleMultiplier int

	// ScanNamespaceRemainder enables scanning unvisited nodes in the target
	// namespace when graph traversal finds no issues.
	ScanNamespaceRemainder bool
}

// DefaultDiagnoseOptions returns DiagnoseOptions populated with sensible
// defaults. Now is intentionally left as the zero value so callers can inject a
// deterministic reference time.
func DefaultDiagnoseOptions() DiagnoseOptions {
	return DiagnoseOptions{
		MaxDepth:               DefaultMaxDepth,
		EventWindow:            DefaultEventWindow,
		TerminatingGracePeriod: DefaultTerminatingGracePeriod,
		LeaseStaleMultiplier:   DefaultLeaseStaleMultiplier,
		ScanNamespaceRemainder: true,
	}
}
