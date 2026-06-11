package ruleutil

import (
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/pkg/resource"
)

// EventEvidence returns one Event evidence item for the latest warning event
// that matches the given reasons.
func EventEvidence(ctx diagnose.RuleContext, ref resource.Reference, reasons ...string) []diagnose.Evidence {
	event, ok := LatestWarningEvent(ctx, ref, nil, reasons...)
	if !ok {
		return nil
	}

	return []diagnose.Evidence{
		NewEventEvidence(ref, event),
	}
}

// EventEvidenceMatching returns one Event evidence item for the latest warning
// event that matches the given reasons and predicate.
func EventEvidenceMatching(
	ctx diagnose.RuleContext,
	ref resource.Reference,
	match func(corev1.Event) bool,
	reasons ...string,
) []diagnose.Evidence {
	event, ok := LatestWarningEvent(ctx, ref, match, reasons...)
	if !ok {
		return nil
	}

	return []diagnose.Evidence{
		NewEventEvidence(ref, event),
	}
}

// LatestWarningEvent returns the newest warning event that matches the given
// reasons and optional predicate.
func LatestWarningEvent(
	ctx diagnose.RuleContext,
	ref resource.Reference,
	match func(corev1.Event) bool,
	reasons ...string,
) (corev1.Event, bool) {
	events := diagnose.WarningEvents(ctx, ref)
	if len(events) == 0 {
		return corev1.Event{}, false
	}

	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if len(reasons) > 0 && !slices.Contains(reasons, event.Reason) {
			continue
		}
		if match != nil && !match(event) {
			continue
		}
		return event, true
	}

	return corev1.Event{}, false
}

// NewEventEvidence converts a Kubernetes event into normalized finding evidence.
func NewEventEvidence(source resource.Reference, event corev1.Event) diagnose.Evidence {
	return diagnose.NewEvidence(source, diagnose.EvidenceEvent, event.Message, event.Reason)
}

// EventMessageContainsAny reports whether the event message contains at least
// one non-empty token, case-insensitively.
func EventMessageContainsAny(event corev1.Event, tokens ...string) bool {
	if len(tokens) == 0 {
		return false
	}

	message := strings.ToLower(event.Message)
	for _, token := range tokens {
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" {
			continue
		}
		if strings.Contains(message, token) {
			return true
		}
	}

	return false
}

// MatchImagePullEvent delegates to structured warning-event parsing.
func MatchImagePullEvent(event corev1.Event, container, image string) bool {
	return evidence.MatchImagePullEvent(event, container, image)
}

// MatchProbeEvent delegates to structured warning-event parsing.
func MatchProbeEvent(event corev1.Event, container string) bool {
	return evidence.MatchProbeEvent(event, container)
}
