package evidence_test

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/pkg/resource"
)

func newEvent(name, evType string, last time.Time, obj corev1.ObjectReference) corev1.Event {
	return corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: name},
		InvolvedObject: obj,
		Type:           evType,
		LastTimestamp:  metav1.NewTime(last),
	}
}

func TestEventIndex_ForByUID(t *testing.T) {
	t.Parallel()

	obj := corev1.ObjectReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Namespace:  "default",
		Name:       "test-pod",
		UID:        types.UID("uid-1"),
	}
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	index := evidence.NewEventIndex([]corev1.Event{
		newEvent("e1", corev1.EventTypeWarning, base, obj),
		newEvent("e2", corev1.EventTypeNormal, base.Add(time.Minute), obj),
	})

	ref := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "test-pod", "uid-1")
	set := index.For(ref)

	if got := set.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2", got)
	}
}

func TestEventIndex_ForLogicalFallback(t *testing.T) {
	t.Parallel()

	obj := corev1.ObjectReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Namespace:  "default",
		Name:       "test-pod",
		UID:        types.UID("uid-1"),
	}
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	index := evidence.NewEventIndex([]corev1.Event{
		newEvent("e1", corev1.EventTypeWarning, base, obj),
	})

	// Look up without a UID: must fall back to the logical key.
	ref := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "test-pod", "")
	set := index.For(ref)

	if got := set.Len(); got != 1 {
		t.Fatalf("Len() = %d, want 1", got)
	}
}

func TestEventIndex_ForMissReturnsEmptySet(t *testing.T) {
	t.Parallel()

	index := evidence.NewEventIndex(nil)

	ref := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "missing", "")
	set := index.For(ref)

	if set == nil {
		t.Fatal("For() returned nil, want non-nil empty set")
	}

	if got := set.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}
}

func TestEventSet_EventsOrderingAndWarnings(t *testing.T) {
	t.Parallel()

	obj := corev1.ObjectReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Namespace:  "default",
		Name:       "test-pod",
		UID:        types.UID("uid-1"),
	}
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	// Provide events out of order to verify deterministic sorting.
	index := evidence.NewEventIndex([]corev1.Event{
		newEvent("newer", corev1.EventTypeNormal, base.Add(2*time.Minute), obj),
		newEvent("older", corev1.EventTypeWarning, base, obj),
		newEvent("middle", corev1.EventTypeWarning, base.Add(time.Minute), obj),
	})

	ref := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "test-pod", "uid-1")
	set := index.For(ref)

	events := set.Events()
	wantOrder := []string{"older", "middle", "newer"}
	if len(events) != len(wantOrder) {
		t.Fatalf("Events() len = %d, want %d", len(events), len(wantOrder))
	}

	for i, want := range wantOrder {
		if events[i].Name != want {
			t.Errorf("Events()[%d].Name = %q, want %q", i, events[i].Name, want)
		}
	}

	warnings := set.Warnings()
	if len(warnings) != 2 {
		t.Fatalf("Warnings() len = %d, want 2", len(warnings))
	}

	latest, ok := set.Latest()
	if !ok {
		t.Fatal("Latest() ok = false, want true")
	}

	if latest.Name != "newer" {
		t.Errorf("Latest().Name = %q, want %q", latest.Name, "newer")
	}
}

func TestEventSet_ZeroValue(t *testing.T) {
	t.Parallel()

	var set evidence.EventSet

	if got := set.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0", got)
	}

	if got := set.Events(); got != nil {
		t.Fatalf("Events() = %v, want nil", got)
	}

	if _, ok := set.Latest(); ok {
		t.Fatal("Latest() ok = true, want false")
	}
}
