package cmd

import (
	"testing"
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
