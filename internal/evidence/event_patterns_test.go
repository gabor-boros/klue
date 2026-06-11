package evidence_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/evidence"
)

func TestParseWarningEventSignal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		event     corev1.Event
		wantKind  string
		wantCause string
	}{
		{
			name: "image pull not found",
			event: corev1.Event{
				Type:    corev1.EventTypeWarning,
				Reason:  "Failed",
				Message: `Failed to pull image "ghcr.io/acme/api:missing" for container "app": manifest unknown`,
			},
			wantKind:  "image-pull",
			wantCause: "image-not-found",
		},
		{
			name: "probe unhealthy timeout",
			event: corev1.Event{
				Type:    corev1.EventTypeWarning,
				Reason:  "Unhealthy",
				Message: `Readiness probe failed: Get "http://10.0.0.1:8080/healthz": context deadline exceeded`,
			},
			wantKind:  "probe",
			wantCause: "timeout",
		},
		{
			name: "scheduling resource pressure",
			event: corev1.Event{
				Type:    corev1.EventTypeWarning,
				Reason:  "FailedScheduling",
				Message: "0/4 nodes are available: 2 Insufficient cpu.",
			},
			wantKind:  "scheduling",
			wantCause: "insufficient-cpu",
		},
		{
			name: "mount missing secret",
			event: corev1.Event{
				Type:    corev1.EventTypeWarning,
				Reason:  "FailedMount",
				Message: `MountVolume.SetUp failed for volume "api-secret" : secret "api-secret" not found`,
			},
			wantKind:  "mount",
			wantCause: "missing-secret",
		},
		{
			name: "provisioning permission denied",
			event: corev1.Event{
				Type:    corev1.EventTypeWarning,
				Reason:  "ProvisioningFailed",
				Message: "failed to provision volume with StorageClass fast: permission denied",
			},
			wantKind:  "provisioning",
			wantCause: "permission-denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := evidence.ParseWarningEventSignal(tt.event)
			if !ok {
				t.Fatalf("ParseWarningEventSignal() = not matched, want %s", tt.wantKind)
			}
			if got.Category != tt.wantKind {
				t.Fatalf("Category = %q, want %q", got.Category, tt.wantKind)
			}
			if got.Cause != tt.wantCause {
				t.Fatalf("Cause = %q, want %q", got.Cause, tt.wantCause)
			}
		})
	}
}

func TestParseWarningEventSignalUnmatchedWarning(t *testing.T) {
	t.Parallel()

	got, ok := evidence.ParseWarningEventSignal(corev1.Event{
		Type:    corev1.EventTypeWarning,
		Reason:  "BackOff",
		Message: "back-off restarting failed container",
	})
	if !ok {
		t.Fatal("ParseWarningEventSignal() = no match, want generic warning signal")
	}
	if got.Category != "generic" {
		t.Fatalf("Category = %q, want generic", got.Category)
	}
}

func TestParseWarningEventSignalSkipsNonWarnings(t *testing.T) {
	t.Parallel()

	if _, ok := evidence.ParseWarningEventSignal(corev1.Event{
		Type:    corev1.EventTypeNormal,
		Reason:  "Scheduled",
		Message: "successfully assigned",
	}); ok {
		t.Fatal("ParseWarningEventSignal() matched non-warning event, want false")
	}
}

func TestMatchHelpers(t *testing.T) {
	t.Parallel()

	event := corev1.Event{
		Type:    corev1.EventTypeWarning,
		Reason:  "Failed",
		Message: `Failed to pull image "ghcr.io/acme/api:missing" for container "app": manifest unknown`,
	}
	if !evidence.MatchImagePullEvent(event, "app", "ghcr.io/acme/api:missing") {
		t.Fatal("MatchImagePullEvent() = false, want true")
	}
	if !evidence.MatchWaitingContainerEvent(event, "app", "ErrImagePull") {
		t.Fatal("MatchWaitingContainerEvent() = false, want true")
	}
}
