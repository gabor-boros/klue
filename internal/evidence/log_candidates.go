package evidence

import (
	"sort"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

const defaultMaxLogCandidates = 10

// SelectLogCandidates returns pod containers whose logs should be fetched for
// the given diagnosis target. Candidates are limited to graph-reachable pods
// (or the target pod when diagnosing a Pod) and unhealthy signals.
func SelectLogCandidates(g *graph.Graph, target resource.Reference, pods []corev1.Pod, events *EventIndex, maxCandidates int) []LogCandidate {
	if maxCandidates <= 0 {
		maxCandidates = defaultMaxLogCandidates
	}

	allowedUIDs := reachablePodUIDs(g, target)
	candidates := make([]LogCandidate, 0)

	for i := range pods {
		pod := &pods[i]
		if len(allowedUIDs) > 0 {
			if _, ok := allowedUIDs[string(pod.UID)]; !ok {
				continue
			}
		}

		podRef := resource.NewReference(resource.ReferenceKindPod, "v1", pod.Namespace, pod.Name, string(pod.UID))
		candidates = append(candidates, podContainerCandidates(podRef, pod, events)...)
	}

	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i].PodRef.Key() + "|" + candidates[i].Container
		right := candidates[j].PodRef.Key() + "|" + candidates[j].Container
		if left != right {
			return left < right
		}
		if candidates[i].Previous != candidates[j].Previous {
			return candidates[i].Previous
		}
		return candidates[i].PodName < candidates[j].PodName
	})

	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}
	return candidates
}

func reachablePodUIDs(g *graph.Graph, target resource.Reference) map[string]struct{} {
	if target.Kind == resource.ReferenceKindPod {
		if target.UID != "" {
			return map[string]struct{}{target.UID: {}}
		}
		return nil
	}
	if g == nil {
		return nil
	}

	start, ok := g.FindByRef(target)
	if !ok {
		return nil
	}

	visited := map[string]struct{}{start.Ref.Key(): {}}
	frontier := []graph.Node{start}
	allowed := make(map[string]struct{})

	for len(frontier) > 0 {
		next := make([]graph.Node, 0)
		for _, node := range frontier {
			if node.Ref.Kind == resource.ReferenceKindPod && node.Ref.UID != "" {
				allowed[node.Ref.UID] = struct{}{}
			}

			related := g.GetRelatedNodes(node)
			sort.Slice(related, func(i, j int) bool {
				return related[i].Ref.Key() < related[j].Ref.Key()
			})

			for _, relatedNode := range related {
				key := relatedNode.Ref.Key()
				if _, seen := visited[key]; seen {
					continue
				}
				visited[key] = struct{}{}
				next = append(next, relatedNode)
			}
		}
		frontier = next
	}

	return allowed
}

func podContainerCandidates(podRef resource.Reference, pod *corev1.Pod, events *EventIndex) []LogCandidate {
	var candidates []LogCandidate
	seen := make(map[string]struct{})
	podWarnings := warningEventsForPod(events, podRef)

	addWithReason := func(container string, previous bool, reason string) {
		key := container
		if previous {
			key += "|previous"
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, LogCandidate{
			PodRef:    podRef,
			Namespace: pod.Namespace,
			PodName:   pod.Name,
			Container: container,
			Previous:  previous,
			Reason:    reason,
		})
	}
	for _, status := range pod.Status.ContainerStatuses {
		if waiting := status.State.Waiting; waiting != nil && waiting.Reason == "CrashLoopBackOff" {
			addWithReason(status.Name, true, "crashloop-backoff")
		}
		if waiting := status.State.Waiting; waiting != nil && waitingLogCandidate(waiting.Reason) {
			if imagePullWaitingReason(waiting.Reason) && status.RestartCount == 0 {
				continue
			}
			if hasContainerWarningEvent(podWarnings, status.Name, waiting.Reason) {
				addWithReason(status.Name, status.RestartCount > 0, "waiting-state-with-correlated-warning")
			}
		}
		if terminated := status.State.Terminated; terminated != nil {
			switch terminated.Reason {
			case "Error", "OOMKilled":
				addWithReason(status.Name, status.RestartCount > 0, "terminated-"+terminated.Reason)
			}
		}
	}

	for _, status := range pod.Status.InitContainerStatuses {
		if waiting := status.State.Waiting; waiting != nil && waiting.Reason == "CrashLoopBackOff" {
			addWithReason(status.Name, true, "init-crashloop-backoff")
		}
		if waiting := status.State.Waiting; waiting != nil && waitingLogCandidate(waiting.Reason) {
			if imagePullWaitingReason(waiting.Reason) && status.RestartCount == 0 {
				continue
			}
			if hasContainerWarningEvent(podWarnings, status.Name, waiting.Reason) {
				addWithReason(status.Name, status.RestartCount > 0, "init-waiting-state-with-correlated-warning")
			}
		}
		if terminated := status.State.Terminated; terminated != nil {
			switch terminated.Reason {
			case "Error", "OOMKilled":
				addWithReason(status.Name, status.RestartCount > 0, "init-terminated-"+terminated.Reason)
			}
		}
	}

	if pod.Status.Phase == corev1.PodFailed {
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name != "" {
				addWithReason(status.Name, false, "pod-failed-phase")
			}
		}
	}

	if probeFailureCandidate(pod, events) {
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Running != nil && !status.Ready {
				addWithReason(status.Name, false, "running-not-ready-with-probe-warning")
			}
		}
	}

	return candidates
}

func probeFailureCandidate(pod *corev1.Pod, events *EventIndex) bool {
	if pod.Status.Phase != corev1.PodRunning || events == nil {
		return false
	}

	podRef := resource.NewReference(resource.ReferenceKindPod, "v1", pod.Namespace, pod.Name, string(pod.UID))
	for _, event := range events.For(podRef).Events() {
		if event.Reason == "Unhealthy" {
			return true
		}
	}
	return false
}

func warningEventsForPod(events *EventIndex, podRef resource.Reference) []corev1.Event {
	if events == nil {
		return nil
	}

	return events.For(podRef).Warnings()
}

func waitingLogCandidate(reason string) bool {
	switch reason {
	case "CreateContainerConfigError", "CreateContainerError", "RunContainerError", "ImagePullBackOff", "ErrImagePull":
		return true
	default:
		return false
	}
}

func imagePullWaitingReason(reason string) bool {
	return reason == "ImagePullBackOff" || reason == "ErrImagePull"
}

func hasContainerWarningEvent(events []corev1.Event, container, waitingReason string) bool {
	for _, event := range events {
		if event.Type != corev1.EventTypeWarning {
			continue
		}
		if event.Reason != "Failed" && event.Reason != "BackOff" && event.Reason != "FailedToRetrieveImagePullSecret" {
			continue
		}

		if MatchWaitingContainerEvent(event, container, waitingReason) {
			return true
		}
	}

	return false
}
