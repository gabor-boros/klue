package evidence

import "github.com/gabor-boros/klue/pkg/resource"

// LogCandidate identifies a pod container whose logs should be fetched.
type LogCandidate struct {
	PodRef    resource.Reference
	Namespace string
	PodName   string
	Container string
	Previous  bool
	Reason    string
}

// LogEntry holds fetched log lines for a single pod container.
type LogEntry struct {
	PodRef     resource.Reference
	Container  string
	Previous   bool
	Lines      []string
	FetchError string
}
