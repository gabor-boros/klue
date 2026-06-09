package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/internal/output"
	"github.com/gabor-boros/klue/internal/rules"
	"github.com/gabor-boros/klue/pkg/resource"
)

// apiVersionFlag is the flag used to disambiguate a resource token that is
// served by multiple API groups or versions (common for custom resources).
const apiVersionFlag = "api-version"

// newWhyCommand builds the generic "why" command. It accepts any
// resource token (kind, plural name or alias), including custom resources
// discovered on the cluster, and resolves it against the live API server.
func newWhyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "why <resource> <name>",
		Short: "Explain why a resource is unhealthy",
		Long: `Explain why a resource is unhealthy by kind, plural name or alias.

The why command discovers the cluster's served API resources at runtime, so
custom resources such as cert-manager Certificates or external-dns DNSEndpoints
can be diagnosed. When a resource name is served by
multiple API groups or versions, use --api-version to disambiguate.`,
		Example: `  klue why pod web-7fdc4f4d74-jj6hb -n default
  klue why pod web-abc -n default --max-depth 2 --event-window 30m
  klue why deployment api -n prod --disable-rule builtin/warning-events
  klue why certificate my-cert -n cert-manager -o json
  klue why certificate my-cert --api-version cert-manager.io/v1 -n cert-manager`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiVersion, err := cmd.Flags().GetString(apiVersionFlag)
			if err != nil {
				return err
			}

			client, err := newDiagnoseClient()
			if err != nil {
				return err
			}

			resources, err := client.DiscoverResources()
			if err != nil {
				return err
			}

			entry, err := kube.ResolveResource(resources, args[0], apiVersion)
			if err != nil {
				return err
			}

			return diagnoseWithClient(cmd, client, resources, entry, args[1])
		},
	}

	cmd.Flags().String(apiVersionFlag, "", "The apiVersion (group/version) to disambiguate the resource")
	registerWhyFlags(cmd)

	return cmd
}

// newDiagnoseClient builds a Kubernetes client from the shared root flags.
func newDiagnoseClient() (*kube.Client, error) {
	kubeconfig, err := rootCmd.PersistentFlags().GetString("kubeconfig")
	if err != nil {
		return nil, err
	}
	kubeContext, err := rootCmd.PersistentFlags().GetString("context")
	if err != nil {
		return nil, err
	}
	fetchConcurrency, err := rootCmd.PersistentFlags().GetInt(fetchConcurrencyFlag)
	if err != nil {
		return nil, err
	}
	clientQPS, err := rootCmd.PersistentFlags().GetFloat32(clientQPSFlag)
	if err != nil {
		return nil, err
	}
	clientBurst, err := rootCmd.PersistentFlags().GetInt(clientBurstFlag)
	if err != nil {
		return nil, err
	}

	return kube.NewClient(kube.Options{
		Kubeconfig:       kubeconfig,
		Context:          kubeContext,
		FetchConcurrency: fetchConcurrency,
		QPS:              clientQPS,
		Burst:            clientBurst,
	})
}

// diagnoseWithClient fetches the resource graph for the target namespace and
// renders a diagnosis for the named object described by entry. The descriptor's
// scope determines whether the target is looked up namespaced or cluster-wide.
func diagnoseWithClient(cmd *cobra.Command, client *kube.Client, resources []kube.APIResource, entry kube.APIResource, name string) error {
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	timeout, err := rootCmd.PersistentFlags().GetDuration(timeoutFlag)
	if err != nil {
		return err
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	snapshot, err := client.FetchWithResources(ctx, namespace, resources)
	if err != nil {
		return err
	}

	resourceGraph := snapshot.BuildGraph()
	eventIndex := evidence.NewEventIndex(snapshot.Events)

	targetNamespace := namespace
	if !entry.Namespaced {
		targetNamespace = ""
	}

	target := resource.NewReference(entry.Kind, entry.APIVersion(), targetNamespace, name, "")
	if _, ok := resourceGraph.FindByRef(target); !ok {
		if !entry.Namespaced {
			return fmt.Errorf("%s not found", target.Display())
		}
		return fmt.Errorf("%s not found in namespace %q", target.Display(), namespace)
	}

	onlyRules, err := cmd.Flags().GetStringArray(onlyRuleFlag)
	if err != nil {
		return err
	}
	disableRules, err := cmd.Flags().GetStringArray(disableRuleFlag)
	if err != nil {
		return err
	}
	if len(onlyRules) > 0 && len(disableRules) > 0 {
		return fmt.Errorf("--%s and --%s are mutually exclusive", onlyRuleFlag, disableRuleFlag)
	}

	selectedRules, err := rules.Select(rules.All(), onlyRules, disableRules)
	if err != nil {
		return err
	}

	options, err := diagnoseOptionsFromFlags(cmd, namespace)
	if err != nil {
		return err
	}

	engine := diagnose.NewEngine(selectedRules...)
	result := engine.Diagnose(diagnose.RuleContext{
		Graph:   resourceGraph,
		Events:  eventIndex,
		Options: options,
	}, target)

	outputFormat, err := cmd.Flags().GetString(outputFlag)
	if err != nil {
		return err
	}

	return output.RenderDiagnosisFormat(cmd.OutOrStdout(), result, outputFormat)
}

func diagnoseOptionsFromFlags(cmd *cobra.Command, namespace string) (diagnose.DiagnoseOptions, error) {
	options := diagnose.DefaultDiagnoseOptions()
	options.Namespace = namespace
	options.Now = time.Now()

	maxDepth, err := cmd.Flags().GetInt(maxDepthFlag)
	if err != nil {
		return options, err
	}
	options.MaxDepth = maxDepth

	eventWindow, err := cmd.Flags().GetDuration(eventWindowFlag)
	if err != nil {
		return options, err
	}
	options.EventWindow = eventWindow

	terminatingGrace, err := cmd.Flags().GetDuration(terminatingGraceFlag)
	if err != nil {
		return options, err
	}
	options.TerminatingGracePeriod = terminatingGrace

	leaseStaleMultiplier, err := cmd.Flags().GetInt(leaseStaleMultiplierFlag)
	if err != nil {
		return options, err
	}
	options.LeaseStaleMultiplier = leaseStaleMultiplier

	noNamespaceScan, err := cmd.Flags().GetBool(noNamespaceScanFlag)
	if err != nil {
		return options, err
	}
	options.ScanNamespaceRemainder = !noNamespaceScan

	return options, nil
}
