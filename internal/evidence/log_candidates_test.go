package evidence_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestSelectLogCandidatesCrashLoopPod(t *testing.T) {
	t.Parallel()

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web", UID: "pod-uid"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					RestartCount: 3,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
					},
				},
			},
		},
	}

	podRef := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "pod-uid")
	g := graph.NewGraph()
	g.AddNode(graph.Node{Ref: podRef, Object: &pod})

	candidates := evidence.SelectLogCandidates(g, podRef, []corev1.Pod{pod}, evidence.NewEventIndex(nil), 10)
	if len(candidates) != 1 {
		t.Fatalf("SelectLogCandidates() = %d candidates, want 1", len(candidates))
	}
	if candidates[0].Container != "app" || !candidates[0].Previous {
		t.Fatalf("candidate = %+v, want app previous logs", candidates[0])
	}
	if candidates[0].Reason == "" {
		t.Fatalf("candidate reason is empty, want selection reason")
	}
}

func TestSelectLogCandidatesRespectsReachability(t *testing.T) {
	t.Parallel()

	target := resource.NewReference(resource.ReferenceKindDeployment, "apps/v1", "default", "api", "deploy-uid")
	relatedPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "api-1", UID: "pod-1"},
		Status: corev1.PodStatus{
			Phase:             corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{{Name: "app"}},
		},
	}
	unrelatedPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "other-1", UID: "pod-2"},
		Status: corev1.PodStatus{
			Phase:             corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{{Name: "app"}},
		},
	}

	g := graph.NewGraph()
	deployNode := graph.Node{Ref: target}
	relatedRef := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "api-1", "pod-1")
	g.AddNode(deployNode)
	g.AddNode(graph.Node{Ref: relatedRef, Object: &relatedPod})
	g.AddEdge(graph.Edge{From: deployNode, To: graph.Node{Ref: relatedRef}, Kind: graph.EdgeOwns})

	candidates := evidence.SelectLogCandidates(g, target, []corev1.Pod{relatedPod, unrelatedPod}, evidence.NewEventIndex(nil), 10)
	if len(candidates) != 1 {
		t.Fatalf("SelectLogCandidates() = %d candidates, want 1 reachable pod", len(candidates))
	}
	if candidates[0].PodName != "api-1" {
		t.Fatalf("candidate pod = %q, want api-1", candidates[0].PodName)
	}
}

func TestSelectLogCandidatesWaitingContainerWithCorrelatedWarningEvent(t *testing.T) {
	t.Parallel()

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "api", UID: "pod-uid"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					RestartCount: 2,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CreateContainerConfigError"},
					},
				},
			},
		},
	}

	podRef := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "api", "pod-uid")
	g := graph.NewGraph()
	g.AddNode(graph.Node{Ref: podRef, Object: &pod})

	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-1"},
			Type:       corev1.EventTypeWarning,
			Reason:     "Failed",
			Message:    "Error: failed to generate container \"app\" config",
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "api",
				UID:        "pod-uid",
			},
		},
	}

	candidates := evidence.SelectLogCandidates(g, podRef, []corev1.Pod{pod}, evidence.NewEventIndex(events), 10)
	if len(candidates) != 1 {
		t.Fatalf("SelectLogCandidates() = %d candidates, want 1", len(candidates))
	}
	if candidates[0].Container != "app" || !candidates[0].Previous {
		t.Fatalf("candidate = %+v, want app previous logs", candidates[0])
	}
	if candidates[0].Reason == "" {
		t.Fatalf("candidate reason is empty, want selection reason")
	}
}

func TestSelectLogCandidatesImagePullBackoffWithoutRestartsSkipped(t *testing.T) {
	t.Parallel()

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "api", UID: "pod-uid"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}},
				},
			},
		},
	}

	podRef := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "api", "pod-uid")
	g := graph.NewGraph()
	g.AddNode(graph.Node{Ref: podRef, Object: &pod})

	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "ev-1"},
			Type:       corev1.EventTypeWarning,
			Reason:     "Failed",
			Message:    "Failed to pull image \"ghcr.io/acme/app:missing\" for container app",
			InvolvedObject: corev1.ObjectReference{
				Kind:       "Pod",
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "api",
				UID:        "pod-uid",
			},
		},
	}

	candidates := evidence.SelectLogCandidates(g, podRef, []corev1.Pod{pod}, evidence.NewEventIndex(events), 10)
	if len(candidates) != 0 {
		t.Fatalf("SelectLogCandidates() = %d candidates, want 0 for image pull with no restarts", len(candidates))
	}
}
