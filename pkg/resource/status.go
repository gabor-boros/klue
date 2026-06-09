package resource

// Status is the status of the resource.
type Status string

const (
	StatusCompleted   Status = "Completed"
	StatusDegraded    Status = "Degraded"
	StatusFailed      Status = "Failed"
	StatusHealthy     Status = "Healthy"
	StatusMissing     Status = "Missing"
	StatusNotReady    Status = "NotReady"
	StatusPending     Status = "Pending"
	StatusProgressing Status = "Progressing"
	StatusReady       Status = "Ready"
	StatusRunning     Status = "Running"
	StatusSucceeded   Status = "Succeeded"
	StatusSuspended   Status = "Suspended"
	StatusTerminated  Status = "Terminated"
	StatusTerminating Status = "Terminating"
	StatusUnhealthy   Status = "Unhealthy"
	StatusUnknown     Status = "Unknown"
)

// healthyStatuses are the statuses that represent a resource working as
// intended.
var healthyStatuses = map[Status]struct{}{
	StatusCompleted: {},
	StatusHealthy:   {},
	StatusReady:     {},
	StatusRunning:   {},
	StatusSucceeded: {},
}

// terminalStatuses are the statuses that represent a resource that has finished
// its lifecycle, whether successfully or not.
var terminalStatuses = map[Status]struct{}{
	StatusCompleted:  {},
	StatusFailed:     {},
	StatusSucceeded:  {},
	StatusTerminated: {},
}

// IsHealthy reports whether the status represents a resource working as
// intended.
func (s Status) IsHealthy() bool {
	_, ok := healthyStatuses[s]
	return ok
}

// IsTerminal reports whether the status represents a resource that has reached
// the end of its lifecycle and will not progress further on its own.
func (s Status) IsTerminal() bool {
	_, ok := terminalStatuses[s]
	return ok
}
