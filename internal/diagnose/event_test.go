package diagnose_test

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestWarningEventsRespectsEventWindow(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ref := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "pod-uid")
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "old"},
			Type:       corev1.EventTypeWarning,
			Reason:     "BackOff",
			Message:    "old warning",
			LastTimestamp: metav1.Time{
				Time: now.Add(-2 * time.Hour),
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "web",
				UID:        "pod-uid",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "new"},
			Type:       corev1.EventTypeWarning,
			Reason:     "BackOff",
			Message:    "new warning",
			LastTimestamp: metav1.Time{
				Time: now.Add(-10 * time.Minute),
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "web",
				UID:        "pod-uid",
			},
		},
	}

	ctx := diagnose.RuleContext{
		Events: evidence.NewEventIndex(events),
		Options: diagnose.DiagnoseOptions{
			Now:         now,
			EventWindow: time.Hour,
		},
	}

	warnings := diagnose.WarningEvents(ctx, ref)
	if len(warnings) != 1 {
		t.Fatalf("WarningEvents() = %d, want 1 in-window event", len(warnings))
	}
	if warnings[0].Message != "new warning" {
		t.Fatalf("WarningEvents()[0].Message = %q, want new warning", warnings[0].Message)
	}
}

func TestHasEventReasonReturnsLatestMatch(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ref := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "pod-uid")
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "old"},
			Type:       corev1.EventTypeWarning,
			Reason:     "FailedScheduling",
			Message:    "old failed scheduling",
			LastTimestamp: metav1.Time{
				Time: now.Add(-30 * time.Minute),
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "web",
				UID:        "pod-uid",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "new"},
			Type:       corev1.EventTypeWarning,
			Reason:     "FailedScheduling",
			Message:    "new failed scheduling",
			LastTimestamp: metav1.Time{
				Time: now.Add(-5 * time.Minute),
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "web",
				UID:        "pod-uid",
			},
		},
	}

	ctx := diagnose.RuleContext{
		Events: evidence.NewEventIndex(events),
		Options: diagnose.DiagnoseOptions{
			Now:         now,
			EventWindow: time.Hour,
		},
	}

	event, ok := diagnose.HasEventReason(ctx, ref, "FailedScheduling")
	if !ok {
		t.Fatal("HasEventReason() = false, want true")
	}
	if event.Message != "new failed scheduling" {
		t.Fatalf("HasEventReason().Message = %q, want latest matching event", event.Message)
	}
}
