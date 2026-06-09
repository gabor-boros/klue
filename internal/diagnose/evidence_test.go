package diagnose_test

import (
	"testing"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestNewEvidence(t *testing.T) {
	t.Parallel()

	source := resource.NewReference(resource.ReferenceKindPod, "v1", "default", "test-pod", "uid-1")

	ev := diagnose.NewEvidence(source, "Event", "container restarted", "raw payload")

	if ev.Source != source {
		t.Errorf("Source = %+v, want %+v", ev.Source, source)
	}

	if ev.Type != "Event" {
		t.Errorf("Type = %q, want %q", ev.Type, "Event")
	}

	if ev.Message != "container restarted" {
		t.Errorf("Message = %q, want %q", ev.Message, "container restarted")
	}

	if ev.Raw != "raw payload" {
		t.Errorf("Raw = %q, want %q", ev.Raw, "raw payload")
	}
}
