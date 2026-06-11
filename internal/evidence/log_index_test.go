package evidence_test

import (
	"strings"
	"testing"

	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestLogIndexRelevantLines(t *testing.T) {
	t.Parallel()

	podRef := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "web", "uid-1")
	index := evidence.NewLogIndex([]evidence.LogEntry{
		{
			PodRef:    podRef,
			Container: "app",
			Lines: []string{
				"starting app",
				"error: connection refused",
				"panic: fatal",
			},
		},
	})

	logs := index.ForPodContainer(podRef, "app")
	ranked := logs.RelevantLines(2)
	if len(ranked) != 2 {
		t.Fatalf("RelevantLines() = %d, want 2", len(ranked))
	}
	if ranked[0].PatternID != "panic" {
		t.Fatalf("top line pattern = %q, want panic", ranked[0].PatternID)
	}

	message := logs.SummaryMessage("app", true)
	if !strings.Contains(message, "app") || !strings.Contains(message, "previous") {
		t.Fatalf("SummaryMessage() = %q, want container and previous", message)
	}

	raw := logs.RawExcerpt(2)
	if !strings.Contains(raw, "panic") {
		t.Fatalf("RawExcerpt() = %q, want panic line", raw)
	}
}
