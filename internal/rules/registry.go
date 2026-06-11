// Package rules aggregates all diagnostic rules so callers can obtain the full
// set without importing each rule subpackage individually.
package rules

import (
	"fmt"
	"strings"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/rules/builtin"
	"github.com/gabor-boros/klue/internal/rules/cronjob"
	"github.com/gabor-boros/klue/internal/rules/csr"
	"github.com/gabor-boros/klue/internal/rules/daemonset"
	"github.com/gabor-boros/klue/internal/rules/deployment"
	"github.com/gabor-boros/klue/internal/rules/hpa"
	"github.com/gabor-boros/klue/internal/rules/ingress"
	"github.com/gabor-boros/klue/internal/rules/job"
	"github.com/gabor-boros/klue/internal/rules/lease"
	"github.com/gabor-boros/klue/internal/rules/networkpolicy"
	"github.com/gabor-boros/klue/internal/rules/node"
	"github.com/gabor-boros/klue/internal/rules/pdb"
	"github.com/gabor-boros/klue/internal/rules/pod"
	"github.com/gabor-boros/klue/internal/rules/pv"
	"github.com/gabor-boros/klue/internal/rules/pvc"
	"github.com/gabor-boros/klue/internal/rules/rbac"
	"github.com/gabor-boros/klue/internal/rules/replicaset"
	"github.com/gabor-boros/klue/internal/rules/service"
	"github.com/gabor-boros/klue/internal/rules/statefulset"
	"github.com/gabor-boros/klue/internal/rules/storageclass"
)

// All returns every diagnostic rule known to klue.
func All() []diagnose.Rule {
	return []diagnose.Rule{
		pod.CrashLoopRule{},
		pod.ImagePullRule{},
		pod.ConfigMissingRule{},
		pod.PendingRule{},
		pod.ProbeRule{},
		pod.MountFailureRule{},

		deployment.RolloutStuckRule{},
		deployment.UnavailableRule{},

		statefulset.UnavailableRule{},
		statefulset.RolloutStuckRule{},

		replicaset.UnavailableRule{},
		replicaset.ReplicaFailureRule{},

		daemonset.UnavailableRule{},
		daemonset.MisscheduledRule{},

		job.FailedRule{},

		cronjob.SuspendedRule{},
		cronjob.JobFailuresRule{},

		node.NotReadyRule{},
		node.PressureRule{},
		node.NetworkUnavailableRule{},
		node.UnschedulableRule{},

		service.NoEndpointsRule{},
		service.SelectorMismatchRule{},
		service.TargetPortMismatchRule{},

		pvc.UnboundRule{},
		pvc.MissingStorageClassRule{},
		pvc.ProvisionerStuckRule{},

		pv.FailedRule{},
		pv.ReleasedRetainedRule{},

		storageclass.NoProvisionerRule{},
		storageclass.WaitForFirstConsumerRule{},

		ingress.BackendMissingRule{},
		ingress.TLSSecretMissingRule{},

		hpa.ScalingDisabledRule{},
		hpa.MissingScaleTargetRule{},

		pdb.DisruptionsBlockedRule{},
		pdb.NoMatchingPodsRule{},

		networkpolicy.NoMatchingPodsRule{},

		rbac.MissingRoleRule{},
		rbac.NoSubjectsRule{},

		csr.DeniedRule{},
		csr.PendingRule{},

		lease.StaleRule{},

		builtin.WarningEventsRule{},
		builtin.LogSignalRule{},
		builtin.FailedConditionRule{},
		builtin.TerminatingStuckRule{},
		builtin.MissingReferenceRule{},
		builtin.OrphanedOwnerRule{},
	}
}

// Select returns a filtered subset of rules. When only is non-empty, only those
// rule IDs are returned. Otherwise, when disable is non-empty, all rules except
// the disabled IDs are returned. Unknown IDs produce an error.
func Select(all []diagnose.Rule, only, disable []string) ([]diagnose.Rule, error) {
	byID := make(map[string]diagnose.Rule, len(all))
	for _, rule := range all {
		byID[rule.ID()] = rule
	}

	if len(only) > 0 {
		selected := make([]diagnose.Rule, 0, len(only))
		var unknown []string
		for _, id := range only {
			rule, ok := byID[id]
			if !ok {
				unknown = append(unknown, id)
				continue
			}
			selected = append(selected, rule)
		}
		if len(unknown) > 0 {
			return nil, fmt.Errorf("unknown rule ID(s): %s", strings.Join(unknown, ", "))
		}
		return selected, nil
	}

	if len(disable) == 0 {
		return all, nil
	}

	disabled := make(map[string]struct{}, len(disable))
	var unknown []string
	for _, id := range disable {
		if _, ok := byID[id]; !ok {
			unknown = append(unknown, id)
			continue
		}
		disabled[id] = struct{}{}
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("unknown rule ID(s): %s", strings.Join(unknown, ", "))
	}

	selected := make([]diagnose.Rule, 0, len(all)-len(disabled))
	for _, rule := range all {
		if _, skip := disabled[rule.ID()]; skip {
			continue
		}
		selected = append(selected, rule)
	}
	return selected, nil
}
