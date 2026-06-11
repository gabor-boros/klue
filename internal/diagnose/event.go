package diagnose

import (
	"slices"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/pkg/resource"
)

// WarningEvents returns the warning events recorded for the given resource,
// tolerating a nil event index.
func WarningEvents(ctx RuleContext, ref resource.Reference) []corev1.Event {
	if ctx.Events == nil {
		return nil
	}

	warnings := ctx.Events.For(ref).Warnings()
	filtered := make([]corev1.Event, 0, len(warnings))
	for _, event := range warnings {
		if evidence.WithinEventWindow(event, ctx.Options.Now, ctx.Options.EventWindow) {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// HasEventReason reports whether any warning event for the resource matches one
// of the given reasons.
func HasEventReason(ctx RuleContext, ref resource.Reference, reasons ...string) (corev1.Event, bool) {
	events := WarningEvents(ctx, ref)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if slices.Contains(reasons, event.Reason) {
			return event, true
		}
	}

	return corev1.Event{}, false
}
