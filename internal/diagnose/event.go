package diagnose

import (
	"slices"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/pkg/resource"
)

// WarningEvents returns the warning events recorded for the given resource,
// tolerating a nil event index.
func WarningEvents(ctx RuleContext, ref resource.Reference) []corev1.Event {
	if ctx.Events == nil {
		return nil
	}

	return ctx.Events.For(ref).Warnings()
}

// HasEventReason reports whether any warning event for the resource matches one
// of the given reasons.
func HasEventReason(ctx RuleContext, ref resource.Reference, reasons ...string) (corev1.Event, bool) {
	for _, event := range WarningEvents(ctx, ref) {
		if slices.Contains(reasons, event.Reason) {
			return event, true
		}
	}

	return corev1.Event{}, false
}
