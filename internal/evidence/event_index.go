// Package evidence provides read models over Kubernetes evidence (events,
// conditions, logs) used by diagnostic rules.
package evidence

import (
	"sort"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/pkg/resource"
)

// EventSet is an ordered collection of events that share the same involved
// object. The zero value is an empty, ready-to-use set.
type EventSet struct {
	events []corev1.Event
}

// Len returns the number of events in the set.
func (s *EventSet) Len() int {
	if s == nil {
		return 0
	}

	return len(s.events)
}

// Events returns a deterministically ordered copy of the events. Events are
// sorted by their effective timestamp (oldest first), with the event name as a
// stable tie-breaker.
func (s *EventSet) Events() []corev1.Event {
	if s == nil || len(s.events) == 0 {
		return nil
	}

	out := make([]corev1.Event, len(s.events))
	copy(out, s.events)

	sort.SliceStable(out, func(i, j int) bool {
		ti, tj := EffectiveEventTime(out[i]), EffectiveEventTime(out[j])
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}

		return out[i].Name < out[j].Name
	})

	return out
}

// Warnings returns the deterministically ordered subset of events whose type is
// Warning.
func (s *EventSet) Warnings() []corev1.Event {
	events := s.Events()

	warnings := make([]corev1.Event, 0, len(events))
	for _, event := range events {
		if event.Type == corev1.EventTypeWarning {
			warnings = append(warnings, event)
		}
	}

	if len(warnings) == 0 {
		return nil
	}

	return warnings
}

// Latest returns the most recent event in the set. The boolean is false when
// the set is empty.
func (s *EventSet) Latest() (corev1.Event, bool) {
	events := s.Events()
	if len(events) == 0 {
		return corev1.Event{}, false
	}

	return events[len(events)-1], true
}

// add appends an event to the set.
func (s *EventSet) add(event corev1.Event) {
	s.events = append(s.events, event)
}

// EventIndex groups Kubernetes events by the object they involve, allowing
// rules to look up the events relevant to a given resource.
type EventIndex struct {
	byUID     map[string]*EventSet
	byLogical map[string]*EventSet
}

// NewEventIndex builds an EventIndex from the given events. Each event is
// indexed by both the UID and the logical key of its involved object so it can
// be retrieved regardless of whether the caller knows the UID.
func NewEventIndex(events []corev1.Event) *EventIndex {
	index := &EventIndex{
		byUID:     make(map[string]*EventSet),
		byLogical: make(map[string]*EventSet),
	}

	for _, event := range events {
		ref := resource.ReferenceFromObjectReference(event.InvolvedObject)

		if uid := string(event.InvolvedObject.UID); uid != "" {
			index.setFor(index.byUID, ref.Key(), event)
		}

		index.setFor(index.byLogical, ref.LogicalKey(), event)
	}

	return index
}

// setFor appends an event to the set stored under key in the given map,
// creating the set if necessary.
func (i *EventIndex) setFor(m map[string]*EventSet, key string, event corev1.Event) {
	set, ok := m[key]
	if !ok {
		set = &EventSet{}
		m[key] = set
	}

	set.add(event)
}

// For returns the events involving the given resource. It prefers a UID match
// and falls back to the logical key. The returned set is never nil; callers
// receive an empty set when no events match.
func (i *EventIndex) For(ref resource.Reference) *EventSet {
	if i != nil && ref.UID != "" {
		if set, ok := i.byUID[ref.Key()]; ok {
			return set
		}
	}

	if i != nil {
		if set, ok := i.byLogical[ref.LogicalKey()]; ok {
			return set
		}
	}

	return &EventSet{}
}
