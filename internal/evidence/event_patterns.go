package evidence

import (
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// EventSignal is a structured interpretation of a warning event message.
type EventSignal struct {
	Category  string
	Cause     string
	Container string
	Image     string
	Volume    string
	ProbeType string
}

var (
	containerTokenRE = regexp.MustCompile(`(?i)container\s+"?([a-z0-9\-._]+)"?`)
	imageTokenRE     = regexp.MustCompile(`(?i)image\s+"([^"]+)"`)
	quotedWordRE     = regexp.MustCompile(`"([^"]+)"`)
	volumeNameRE     = regexp.MustCompile(`(?i)volume\s+"?([a-z0-9\-._]+)"?`)
)

// ParseWarningEventSignal parses warning events into a conservative structured
// signal used by rules and candidate selection.
func ParseWarningEventSignal(event corev1.Event) (EventSignal, bool) {
	if event.Type != corev1.EventTypeWarning {
		return EventSignal{}, false
	}

	msg := strings.ToLower(event.Message)
	signal := EventSignal{}

	switch {
	case isImagePullEvent(event, msg):
		signal.Category = "image-pull"
		signal.Cause = imagePullCause(msg)
		signal.Container = parseContainerToken(event.Message)
		signal.Image = parseImageToken(event.Message)
		return signal, true
	case isProbeEvent(event, msg):
		signal.Category = "probe"
		signal.Cause = probeCause(msg)
		signal.ProbeType = detectProbeType(msg)
		signal.Container = parseContainerToken(event.Message)
		return signal, true
	case event.Reason == "FailedScheduling":
		signal.Category = "scheduling"
		signal.Cause = schedulingCause(msg)
		return signal, true
	case isMountEvent(event, msg):
		signal.Category = "mount"
		signal.Cause = mountCause(msg)
		signal.Volume = parseVolumeToken(event.Message)
		return signal, true
	case event.Reason == "ProvisioningFailed":
		signal.Category = "provisioning"
		signal.Cause = provisioningCause(msg)
		return signal, true
	default:
		return EventSignal{Category: "generic", Cause: "warning"}, true
	}
}

// MatchImagePullEvent reports whether the warning event matches the given
// container/image context for image pull failures.
func MatchImagePullEvent(event corev1.Event, container, image string) bool {
	signal, ok := ParseWarningEventSignal(event)
	if !ok || signal.Category != "image-pull" {
		return false
	}

	if container != "" && signal.Container != "" && !strings.EqualFold(signal.Container, container) {
		return false
	}
	if image != "" && signal.Image != "" && !strings.EqualFold(signal.Image, image) {
		return false
	}
	return true
}

// MatchWaitingContainerEvent reports whether the warning event supports a
// waiting-state failure for the given container.
func MatchWaitingContainerEvent(event corev1.Event, container, waitingReason string) bool {
	signal, ok := ParseWarningEventSignal(event)
	if !ok {
		return false
	}
	if container != "" && signal.Container != "" && !strings.EqualFold(signal.Container, container) {
		return false
	}

	switch waitingReason {
	case "ImagePullBackOff", "ErrImagePull":
		return signal.Category == "image-pull"
	case "CreateContainerConfigError", "CreateContainerError", "RunContainerError":
		return signal.Category == "mount" || signal.Category == "generic" || signal.Category == "provisioning"
	default:
		return signal.Category == "generic"
	}
}

// MatchProbeEvent reports whether the warning event indicates a probe failure
// and, when known, points to the provided container.
func MatchProbeEvent(event corev1.Event, container string) bool {
	signal, ok := ParseWarningEventSignal(event)
	if !ok || signal.Category != "probe" {
		return false
	}
	if container != "" && signal.Container != "" && !strings.EqualFold(signal.Container, container) {
		return false
	}
	return true
}

func isImagePullEvent(event corev1.Event, msg string) bool {
	if event.Reason == "FailedToRetrieveImagePullSecret" {
		return true
	}
	return strings.Contains(msg, "pull image") || strings.Contains(msg, "imagepullbackoff") || strings.Contains(msg, "errimagepull")
}

func imagePullCause(msg string) string {
	switch {
	case strings.Contains(msg, "secret"), strings.Contains(msg, "pull secret"):
		return "registry-auth"
	case strings.Contains(msg, "manifest unknown"), strings.Contains(msg, "not found"), strings.Contains(msg, "repository does not exist"):
		return "image-not-found"
	case strings.Contains(msg, "x509"), strings.Contains(msg, "tls"), strings.Contains(msg, "certificate"):
		return "registry-tls"
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "dial tcp"), strings.Contains(msg, "connection refused"), strings.Contains(msg, "i/o timeout"):
		return "registry-network"
	default:
		return "pull-failed"
	}
}

func isProbeEvent(event corev1.Event, msg string) bool {
	if event.Reason != "Unhealthy" {
		return false
	}
	return strings.Contains(msg, "probe") || strings.Contains(msg, "health")
}

func probeCause(msg string) string {
	switch {
	case strings.Contains(msg, "connection refused"):
		return "connection-refused"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline"):
		return "timeout"
	default:
		return "probe-failed"
	}
}

func detectProbeType(msg string) string {
	switch {
	case strings.Contains(msg, "readiness"):
		return "readiness"
	case strings.Contains(msg, "liveness"):
		return "liveness"
	case strings.Contains(msg, "startup"):
		return "startup"
	default:
		return ""
	}
}

func schedulingCause(msg string) string {
	switch {
	case strings.Contains(msg, "insufficient cpu"):
		return "insufficient-cpu"
	case strings.Contains(msg, "insufficient memory"):
		return "insufficient-memory"
	case strings.Contains(msg, "didn't match node selector"), strings.Contains(msg, "node(s) didn't match pod affinity"), strings.Contains(msg, "node affinity"):
		return "selector-or-affinity"
	case strings.Contains(msg, "had taint"), strings.Contains(msg, "taint"), strings.Contains(msg, "toleration"):
		return "taints-or-tolerations"
	case strings.Contains(msg, "persistentvolumeclaim"), strings.Contains(msg, "unbound immediate persistentvolumeclaims"):
		return "pvc-binding"
	case strings.Contains(msg, "topology spread"), strings.Contains(msg, "match pod topology spread constraints"):
		return "topology-spread"
	default:
		return "scheduling-failed"
	}
}

func isMountEvent(event corev1.Event, msg string) bool {
	if event.Reason == "FailedMount" || event.Reason == "FailedAttachVolume" || event.Reason == "FailedMapVolume" {
		return true
	}
	return strings.Contains(msg, "mountvolume") || strings.Contains(msg, "failed to mount")
}

func mountCause(msg string) string {
	switch {
	case strings.Contains(msg, "timed out"), strings.Contains(msg, "timeout"):
		return "mount-timeout"
	case strings.Contains(msg, "secret") && strings.Contains(msg, "not found"):
		return "missing-secret"
	case strings.Contains(msg, "configmap") && strings.Contains(msg, "not found"):
		return "missing-configmap"
	case strings.Contains(msg, "persistentvolumeclaim") && strings.Contains(msg, "not found"):
		return "missing-pvc"
	default:
		return "mount-failed"
	}
}

func provisioningCause(msg string) string {
	switch {
	case strings.Contains(msg, "quota"), strings.Contains(msg, "exceeded"):
		return "quota-exceeded"
	case strings.Contains(msg, "permission"), strings.Contains(msg, "forbidden"), strings.Contains(msg, "denied"):
		return "permission-denied"
	case strings.Contains(msg, "topology"), strings.Contains(msg, "zone"), strings.Contains(msg, "region"):
		return "topology-constraint"
	default:
		return "provisioning-failed"
	}
}

func parseContainerToken(message string) string {
	match := containerTokenRE.FindStringSubmatch(message)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func parseImageToken(message string) string {
	match := imageTokenRE.FindStringSubmatch(message)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func parseVolumeToken(message string) string {
	match := volumeNameRE.FindStringSubmatch(message)
	if len(match) == 2 {
		return match[1]
	}

	quoted := quotedWordRE.FindStringSubmatch(message)
	if len(quoted) == 2 {
		return quoted[1]
	}
	return ""
}
