package pod_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/pod"
	"github.com/gabor-boros/klue/pkg/resource"
)

func podNode(p *corev1.Pod) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindPod, "v1", p.Namespace, p.Name, string(p.UID)),
		Object: p,
	}
}

func TestCrashLoopRule(t *testing.T) {
	t.Parallel()

	crashing := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					RestartCount: 5,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff", Message: "back-off"},
					},
				},
			},
		},
	}

	findings := pod.CrashLoopRule{}.Evaluate(diagnose.RuleContext{}, podNode(crashing))
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if findings[0].Severity != diagnose.SeverityCritical {
		t.Errorf("Severity = %q, want %q", findings[0].Severity, diagnose.SeverityCritical)
	}

	healthy := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ok"}}
	if got := (pod.CrashLoopRule{}).Evaluate(diagnose.RuleContext{}, podNode(healthy)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for healthy pod", len(got))
	}
}

func TestCrashLoopRuleWithLogs(t *testing.T) {
	t.Parallel()

	crashing := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web", UID: "pod-uid"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					RestartCount: 5,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff", Message: "back-off"},
					},
				},
			},
		},
	}

	node := podNode(crashing)
	logIndex := evidence.NewLogIndex([]evidence.LogEntry{
		{
			PodRef:    node.Ref,
			Container: "app",
			Previous:  true,
			Lines:     []string{"panic: intentional crash"},
		},
	})

	findings := pod.CrashLoopRule{}.Evaluate(diagnose.RuleContext{Logs: logIndex}, node)
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if len(findings[0].Evidence) < 2 {
		t.Fatalf("Evidence = %d items, want status and log evidence", len(findings[0].Evidence))
	}
	if findings[0].Evidence[1].Type != diagnose.EvidenceLog {
		t.Fatalf("second evidence type = %q, want Log", findings[0].Evidence[1].Type)
	}
	if findings[0].Explanation == "" || findings[0].Evidence[1].Raw == "" {
		t.Fatalf("expected log-enriched explanation and raw excerpt")
	}
}

func TestConfigMissingRule(t *testing.T) {
	t.Parallel()

	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{Name: "cfg", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "missing-cfg"}}}},
			},
		},
	}

	builder := graph.NewBuilder()
	builder.AddNode(*podNode(p))
	g := builder.Build()

	findings := (pod.ConfigMissingRule{}).Evaluate(diagnose.RuleContext{Graph: g}, podNode(p))
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 (missing configmap)", len(findings))
	}

	// Now add the configmap; the finding should disappear.
	builder.AddNode(graph.Node{Ref: resource.NewReference(resource.ReferenceKindConfigMap, "v1", "default", "missing-cfg", "")})
	g = builder.Build()
	if got := (pod.ConfigMissingRule{}).Evaluate(diagnose.RuleContext{Graph: g}, podNode(p)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 once configmap exists", len(got))
	}
}

func TestImagePullRuleStatusOnly(t *testing.T) {
	t.Parallel()

	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "imgpull", UID: "pod-uid"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Image: "ghcr.io/acme/app:missing",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ImagePullBackOff",
							Message: "Back-off pulling image \"ghcr.io/acme/app:missing\"",
						},
					},
				},
			},
		},
	}

	findings := pod.ImagePullRule{}.Evaluate(diagnose.RuleContext{}, podNode(p))
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if findings[0].Confidence != 0.9 {
		t.Fatalf("confidence = %v, want 0.9", findings[0].Confidence)
	}
	if len(findings[0].Evidence) != 1 {
		t.Fatalf("evidence = %d items, want status-only evidence", len(findings[0].Evidence))
	}
	if findings[0].Evidence[0].Type != diagnose.EvidenceStatus {
		t.Fatalf("evidence[0].Type = %q, want %q", findings[0].Evidence[0].Type, diagnose.EvidenceStatus)
	}
}

func TestImagePullRuleWithCorrelatedWarningEvent(t *testing.T) {
	t.Parallel()

	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "imgpull", UID: "pod-uid"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Image: "ghcr.io/acme/app:missing",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ErrImagePull",
							Message: "manifest unknown for ghcr.io/acme/app:missing",
						},
					},
				},
			},
		},
	}

	warnings := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-1"},
			Type:       corev1.EventTypeWarning,
			Reason:     "Failed",
			Message:    "Failed to pull image \"ghcr.io/acme/app:missing\" for container app: manifest unknown",
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "imgpull",
				UID:        "pod-uid",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-2"},
			Type:       corev1.EventTypeWarning,
			Reason:     "Failed",
			Message:    "Failed to pull image \"ghcr.io/acme/other:latest\" for container sidecar",
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "imgpull",
				UID:        "pod-uid",
			},
		},
	}

	findings := pod.ImagePullRule{}.Evaluate(
		diagnose.RuleContext{Events: evidence.NewEventIndex(warnings)},
		podNode(p),
	)
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if findings[0].Confidence != 0.95 {
		t.Fatalf("confidence = %v, want 0.95 with correlated event", findings[0].Confidence)
	}
	if len(findings[0].Evidence) != 2 {
		t.Fatalf("evidence = %d items, want status + event", len(findings[0].Evidence))
	}
	if findings[0].Evidence[1].Type != diagnose.EvidenceEvent {
		t.Fatalf("evidence[1].Type = %q, want %q", findings[0].Evidence[1].Type, diagnose.EvidenceEvent)
	}
}

func TestPendingRuleWithFailedSchedulingEvent(t *testing.T) {
	t.Parallel()

	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pending", UID: "pod-uid"},
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}
	node := podNode(p)
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-1"},
			Type:       corev1.EventTypeWarning,
			Reason:     "FailedScheduling",
			Message:    "0/3 nodes are available: 3 Insufficient cpu.",
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "pending",
				UID:        "pod-uid",
			},
		},
	}

	findings := pod.PendingRule{}.Evaluate(diagnose.RuleContext{
		Events:  evidence.NewEventIndex(events),
		Options: diagnose.DiagnoseOptions{},
	}, node)
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if findings[0].Evidence[0].Type != diagnose.EvidenceEvent {
		t.Fatalf("evidence[0].Type = %q, want %q", findings[0].Evidence[0].Type, diagnose.EvidenceEvent)
	}
}

func TestProbeRuleUsesStructuredUnhealthyEvent(t *testing.T) {
	t.Parallel()

	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "probe", UID: "pod-uid"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: false,
					State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				},
			},
		},
	}
	node := podNode(p)
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-1"},
			Type:       corev1.EventTypeWarning,
			Reason:     "Unhealthy",
			Message:    "Readiness probe failed: HTTP probe failed with statuscode: 503",
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "probe",
				UID:        "pod-uid",
			},
		},
	}

	findings := pod.ProbeRule{}.Evaluate(diagnose.RuleContext{
		Events: evidence.NewEventIndex(events),
	}, node)
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
}

func TestMountFailureRule(t *testing.T) {
	t.Parallel()

	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "mount", UID: "pod-uid"},
	}
	node := podNode(p)
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-1"},
			Type:       corev1.EventTypeWarning,
			Reason:     "FailedMount",
			Message:    `MountVolume.SetUp failed for volume "api-secret" : secret "api-secret" not found`,
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "mount",
				UID:        "pod-uid",
			},
		},
	}

	findings := pod.MountFailureRule{}.Evaluate(diagnose.RuleContext{
		Events: evidence.NewEventIndex(events),
	}, node)
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if findings[0].ID != "pod/mount-failure" {
		t.Fatalf("ID = %q, want pod/mount-failure", findings[0].ID)
	}
}
