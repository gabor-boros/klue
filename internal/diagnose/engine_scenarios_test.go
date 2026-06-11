package diagnose_test

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/internal/rules/builtin"
	"github.com/gabor-boros/klue/internal/rules/pod"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestEngineScenario_ProbeEventAndLogsSuppressGenericWarning(t *testing.T) {
	t.Parallel()

	now := time.Now()
	podObj := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "api",
			UID:       "pod-uid",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: false,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-1"},
			Type:       corev1.EventTypeWarning,
			Reason:     "Unhealthy",
			Message:    "Readiness probe failed: HTTP probe failed with statuscode: 503",
			LastTimestamp: metav1.Time{
				Time: now.Add(-5 * time.Minute),
			},
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "api",
				UID:        "pod-uid",
			},
		},
	}

	snapshot := kube.ResourceSnapshot{
		Namespace: "default",
		Pods:      []corev1.Pod{podObj},
		Events:    events,
	}
	graph := snapshot.BuildGraph()
	target := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "api", "pod-uid")
	logs := evidence.NewLogIndex([]evidence.LogEntry{
		{
			PodRef:    target,
			Container: "app",
			Lines:     []string{"readiness endpoint timeout"},
		},
	})

	engine := diagnose.NewEngine(
		pod.ProbeRule{},
		builtin.WarningEventsRule{},
		builtin.LogSignalRule{},
	)
	result := engine.Diagnose(diagnose.RuleContext{
		Graph:  graph,
		Events: evidence.NewEventIndex(events),
		Logs:   logs,
		Options: diagnose.DiagnoseOptions{
			Now:         now,
			EventWindow: time.Hour,
			Debug:       true,
		},
	}, target)

	if result.RootCause == nil {
		t.Fatal("RootCause = nil, want pod/probe-failure")
	}
	if result.RootCause.ID != "pod/probe-failure" {
		t.Fatalf("RootCause.ID = %q, want pod/probe-failure", result.RootCause.ID)
	}
	for _, finding := range result.Findings {
		if finding.ID == "builtin/warning-events" {
			t.Fatalf("unexpected redundant builtin/warning-events finding: %+v", finding)
		}
	}
	if result.Debug == nil || result.Debug.SuppressedFindings == 0 {
		t.Fatalf("Debug metadata missing suppression details: %+v", result.Debug)
	}
}
