// Package builtin contains generic diagnostic rules that apply to every
// Kubernetes built-in resource. They reason about patterns common to all
// objects (warning events, status conditions, deletion timestamps, owner
// references) rather than a single resource type, complementing the typed rules
// in the sibling packages.
package builtin

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

// defaultTerminatingGracePeriod is the age beyond which an object still carrying
// a deletion timestamp is considered stuck terminating.
const defaultTerminatingGracePeriod = 5 * time.Minute

// withinWindow reports whether the event is recent enough to be relevant. When
// the reference time is zero (deterministic tests without a clock) all events
// are considered relevant.
func withinWindow(event corev1.Event, now time.Time, window time.Duration) bool {
	if now.IsZero() || window <= 0 {
		return true
	}

	stamp := eventTime(event)
	if stamp.IsZero() {
		return true
	}

	return now.Sub(stamp) <= window
}

// eventTime returns the most meaningful timestamp for an event.
func eventTime(event corev1.Event) time.Time {
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.EventTime.IsZero() {
		return event.EventTime.Time
	}
	return event.FirstTimestamp.Time
}

// describeKind returns a lowercase kubectl-friendly token for a resource kind.
func describeKind(kind resource.Kind) string {
	return ruleutil.KubectlKind(kind)
}
