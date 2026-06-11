package ruleutil

import (
	"fmt"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/pkg/resource"
)

// LogEvidence returns log-based evidence for a pod container when logs were
// fetched during diagnosis.
func LogEvidence(ctx diagnose.RuleContext, podRef resource.Reference, container string, previous bool) []diagnose.Evidence {
	if ctx.Logs == nil {
		return nil
	}

	logs := ctx.Logs.ForPodContainer(podRef, container)
	message := logs.SummaryMessage(container, previous)
	if message == "" {
		return nil
	}

	return []diagnose.Evidence{
		diagnose.NewEvidence(podRef, diagnose.EvidenceLog, message, logs.RawExcerpt(3)),
	}
}

// LogExplanation returns an explanation suffix derived from log signals.
func LogExplanation(ctx diagnose.RuleContext, podRef resource.Reference, container string) string {
	if ctx.Logs == nil {
		return ""
	}

	logs := ctx.Logs.ForPodContainer(podRef, container)
	signal, line, ok := logs.BestSignal()
	if !ok {
		return ""
	}

	if signal.ID == "error-keyword" {
		return fmt.Sprintf(" Logs indicate: %s.", truncateForExplanation(line))
	}
	return fmt.Sprintf(" Logs indicate %s.", signal.Summary)
}

func truncateForExplanation(line string) string {
	const max = 100
	if len(line) <= max {
		return line
	}
	return line[:max-3] + "..."
}
