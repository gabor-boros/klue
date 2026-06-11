package ruleutil_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/rules/ruleutil"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestEventEvidenceReturnsLatestMatchingWarning(t *testing.T) {
	t.Parallel()

	ref := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "api", "pod-uid")
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-1"},
			Type:       corev1.EventTypeWarning,
			Reason:     "Failed",
			Message:    "failed to pull image",
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "api",
				UID:        "pod-uid",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-2"},
			Type:       corev1.EventTypeWarning,
			Reason:     "BackOff",
			Message:    "back-off pulling image",
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "api",
				UID:        "pod-uid",
			},
		},
	}

	ctx := diagnose.RuleContext{Events: evidence.NewEventIndex(events)}
	ev := ruleutil.EventEvidence(ctx, ref, "Failed", "BackOff")
	if len(ev) != 1 {
		t.Fatalf("EventEvidence() = %d items, want 1", len(ev))
	}
	if ev[0].Type != diagnose.EvidenceEvent {
		t.Fatalf("EventEvidence()[0].Type = %q, want %q", ev[0].Type, diagnose.EvidenceEvent)
	}
	if ev[0].Raw != "BackOff" {
		t.Fatalf("EventEvidence()[0].Raw = %q, want BackOff", ev[0].Raw)
	}
}

func TestEventEvidenceMatchingUsesPredicate(t *testing.T) {
	t.Parallel()

	ref := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "api", "pod-uid")
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-1"},
			Type:       corev1.EventTypeWarning,
			Reason:     "Failed",
			Message:    "failed to pull image for container sidecar",
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "api",
				UID:        "pod-uid",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-2"},
			Type:       corev1.EventTypeWarning,
			Reason:     "Failed",
			Message:    "failed to pull image for container app",
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "api",
				UID:        "pod-uid",
			},
		},
	}

	ctx := diagnose.RuleContext{Events: evidence.NewEventIndex(events)}
	ev := ruleutil.EventEvidenceMatching(
		ctx,
		ref,
		func(event corev1.Event) bool {
			return ruleutil.EventMessageContainsAny(event, "container app")
		},
		"Failed",
	)
	if len(ev) != 1 {
		t.Fatalf("EventEvidenceMatching() = %d items, want 1", len(ev))
	}
	if ev[0].Message != "failed to pull image for container app" {
		t.Fatalf("EventEvidenceMatching()[0].Message = %q, want container app message", ev[0].Message)
	}
}
