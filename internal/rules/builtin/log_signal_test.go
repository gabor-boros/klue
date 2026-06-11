package builtin_test

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/builtin"
)

func TestLogSignalRule(t *testing.T) {
	t.Parallel()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web", UID: "pod-uid"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
			},
		},
	}

	ref := podRef("web")
	ref.UID = "pod-uid"
	node := &graph.Node{Ref: ref, Object: pod}
	logIndex := evidence.NewLogIndex([]evidence.LogEntry{
		{PodRef: ref, Container: "app", Lines: []string{"dial tcp: connection refused"}},
	})

	findings := (builtin.LogSignalRule{}).Evaluate(diagnose.RuleContext{Logs: logIndex}, node)
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if !strings.Contains(findings[0].Title, "connection refused") {
		t.Fatalf("title = %q, want connection refused signal", findings[0].Title)
	}
}

func TestLogSignalRuleSkipsCrashLoop(t *testing.T) {
	t.Parallel()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web", UID: "pod-uid"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
				},
			},
		},
	}

	ref := podRef("web")
	ref.UID = "pod-uid"
	node := &graph.Node{Ref: ref, Object: pod}
	logIndex := evidence.NewLogIndex([]evidence.LogEntry{
		{PodRef: ref, Container: "app", Lines: []string{"panic: boom"}},
	})

	if got := (builtin.LogSignalRule{}).Evaluate(diagnose.RuleContext{Logs: logIndex}, node); len(got) != 0 {
		t.Fatalf("Evaluate() = %d findings, want 0 when status already explains failure", len(got))
	}
}
