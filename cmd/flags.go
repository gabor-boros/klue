package cmd

import (
	"github.com/spf13/cobra"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/kube"
)

const (
	fetchConcurrencyFlag     = "fetch-concurrency"
	clientQPSFlag            = "client-qps"
	clientBurstFlag          = "client-burst"
	timeoutFlag              = "timeout"
	maxDepthFlag             = "max-depth"
	eventWindowFlag          = "event-window"
	terminatingGraceFlag     = "terminating-grace"
	leaseStaleMultiplierFlag = "lease-stale-multiplier"
	noNamespaceScanFlag      = "no-namespace-scan"
	noFetchLogsFlag          = "no-fetch-logs"
	logTailLinesFlag         = "log-tail-lines"
	debugFlag                = "debug"
	disableRuleFlag          = "disable-rule"
	onlyRuleFlag             = "only-rule"
	outputFlag               = "output"
)

func registerRootFlags() {
	rootCmd.PersistentFlags().Int(fetchConcurrencyFlag, kube.DefaultFetchConcurrency(), "Maximum number of parallel Kubernetes list operations while fetching cluster state")
	rootCmd.PersistentFlags().Float32(clientQPSFlag, kube.DefaultClientQPS(), "Kubernetes API client rate limit in queries per second")
	rootCmd.PersistentFlags().Int(clientBurstFlag, kube.DefaultClientBurst(), "Kubernetes API client burst size")
	rootCmd.PersistentFlags().Duration(timeoutFlag, 0, "Maximum time to spend fetching cluster state (0 means no additional limit)")
}

func registerWhyFlags(cmd *cobra.Command) {
	cmd.Flags().Int(maxDepthFlag, diagnose.DefaultMaxDepth, "Maximum graph hops to traverse from the target (0 means unlimited)")
	cmd.Flags().Duration(eventWindowFlag, diagnose.DefaultEventWindow, "Maximum age of warning events to consider relevant")
	cmd.Flags().Duration(terminatingGraceFlag, diagnose.DefaultTerminatingGracePeriod, "How long a resource may remain terminating before it is reported as stuck")
	cmd.Flags().Int(leaseStaleMultiplierFlag, diagnose.DefaultLeaseStaleMultiplier, "How many lease durations may elapse before a lease holder is considered stale")
	cmd.Flags().Bool(noNamespaceScanFlag, false, "Do not scan unvisited resources in the target namespace when graph traversal finds no issues")
	cmd.Flags().Bool(noFetchLogsFlag, false, "Do not fetch container logs for unhealthy related pods")
	cmd.Flags().Int64(logTailLinesFlag, diagnose.DefaultLogTailLines, "Maximum number of trailing log lines to fetch per container")
	cmd.Flags().Bool(debugFlag, false, "Include debug metadata about evidence collection and rule correlation")
	cmd.Flags().StringArray(disableRuleFlag, nil, "Disable a diagnostic rule by ID (repeatable)")
	cmd.Flags().StringArray(onlyRuleFlag, nil, "Run only the listed diagnostic rule IDs (repeatable)")
	cmd.Flags().StringP(outputFlag, "o", "text", "Output format (text or json)")
}
