package evidence

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

// EffectiveEventTime returns the most meaningful timestamp for an event.
func EffectiveEventTime(event corev1.Event) time.Time {
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.EventTime.IsZero() {
		return event.EventTime.Time
	}
	return event.FirstTimestamp.Time
}

// WithinEventWindow reports whether the event is recent enough to be considered
// relevant. When the reference time is zero all events are considered relevant.
func WithinEventWindow(event corev1.Event, now time.Time, window time.Duration) bool {
	if now.IsZero() || window <= 0 {
		return true
	}

	stamp := EffectiveEventTime(event)
	if stamp.IsZero() {
		return true
	}

	return now.Sub(stamp) <= window
}
