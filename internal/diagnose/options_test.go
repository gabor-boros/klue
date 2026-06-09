package diagnose_test

import (
	"testing"

	"github.com/gabor-boros/klue/internal/diagnose"
)

func TestDefaultDiagnoseOptions(t *testing.T) {
	t.Parallel()

	opts := diagnose.DefaultDiagnoseOptions()

	if opts.MaxDepth != diagnose.DefaultMaxDepth {
		t.Errorf("MaxDepth = %d, want %d", opts.MaxDepth, diagnose.DefaultMaxDepth)
	}

	if opts.EventWindow != diagnose.DefaultEventWindow {
		t.Errorf("EventWindow = %v, want %v", opts.EventWindow, diagnose.DefaultEventWindow)
	}

	if opts.TerminatingGracePeriod != diagnose.DefaultTerminatingGracePeriod {
		t.Errorf("TerminatingGracePeriod = %v, want %v", opts.TerminatingGracePeriod, diagnose.DefaultTerminatingGracePeriod)
	}

	if opts.LeaseStaleMultiplier != diagnose.DefaultLeaseStaleMultiplier {
		t.Errorf("LeaseStaleMultiplier = %d, want %d", opts.LeaseStaleMultiplier, diagnose.DefaultLeaseStaleMultiplier)
	}

	if !opts.ScanNamespaceRemainder {
		t.Error("ScanNamespaceRemainder = false, want true")
	}

	if opts.Namespace != "" {
		t.Errorf("Namespace = %q, want empty", opts.Namespace)
	}

	if !opts.Now.IsZero() {
		t.Errorf("Now = %v, want zero value", opts.Now)
	}
}
