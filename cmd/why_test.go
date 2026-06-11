package cmd

import (
	"testing"

	"github.com/gabor-boros/klue/internal/diagnose"
)

func TestNewDiagnoseCommand(t *testing.T) {
	t.Parallel()

	cmd := newWhyCommand()

	if cmd.Use != "why <resource> <name>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "why <resource> <name>")
	}

	for _, flag := range []string{
		apiVersionFlag,
		maxDepthFlag,
		eventWindowFlag,
		terminatingGraceFlag,
		leaseStaleMultiplierFlag,
		noNamespaceScanFlag,
		fetchCRDsFlag,
		fullSnapshotFlag,
		debugFlag,
		disableRuleFlag,
		onlyRuleFlag,
		outputFlag,
	} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("--%s flag is not registered on why", flag)
		}
	}

	if cmd.Args == nil {
		t.Fatal("why command should validate its arguments")
	}
	if err := cmd.Args(cmd, []string{"certificate"}); err == nil {
		t.Error("expected an error when only one argument is provided")
	}
	if err := cmd.Args(cmd, []string{"certificate", "web-cert"}); err != nil {
		t.Errorf("unexpected error for two arguments: %v", err)
	}
}

func TestDiagnoseCommandRegistered(t *testing.T) {
	t.Parallel()

	cmd, _, err := rootCmd.Find([]string{"why"})
	if err != nil {
		t.Fatalf("why command not registered: %v", err)
	}
	if cmd.Name() != "why" {
		t.Errorf("resolved command name = %q, want why", cmd.Name())
	}
}

func TestRootPersistentFlagsRegistered(t *testing.T) {
	t.Parallel()

	for _, flag := range []string{
		"namespace",
		"kubeconfig",
		"context",
		fetchConcurrencyFlag,
		clientQPSFlag,
		clientBurstFlag,
		timeoutFlag,
	} {
		if rootCmd.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("--%s persistent flag is not registered", flag)
		}
	}
}

func TestDiagnoseOptionsFromFlags_Debug(t *testing.T) {
	t.Parallel()

	cmd := newWhyCommand()
	if err := cmd.Flags().Set(debugFlag, "true"); err != nil {
		t.Fatalf("set --%s: %v", debugFlag, err)
	}

	options, err := diagnoseOptionsFromFlags(cmd, "default")
	if err != nil {
		t.Fatalf("diagnoseOptionsFromFlags() error = %v", err)
	}
	if !options.Debug {
		t.Fatalf("options.Debug = false, want true")
	}
}

func TestShouldFetchLogsForDiagnosis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		findings   []diagnose.Finding
		candidates int
		want       bool
	}{
		{
			name:       "no candidates disables log fetch",
			findings:   nil,
			candidates: 0,
			want:       false,
		},
		{
			name:       "no findings but candidates enables log fetch",
			findings:   nil,
			candidates: 1,
			want:       true,
		},
		{
			name: "crashloop finding enables log fetch",
			findings: []diagnose.Finding{
				{ID: "pod/crashloop"},
			},
			candidates: 1,
			want:       true,
		},
		{
			name: "non pod finding skips log fetch",
			findings: []diagnose.Finding{
				{ID: "service/selector-mismatch"},
			},
			candidates: 2,
			want:       false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldFetchLogsForDiagnosis(tt.findings, tt.candidates); got != tt.want {
				t.Fatalf("shouldFetchLogsForDiagnosis() = %t, want %t", got, tt.want)
			}
		})
	}
}
