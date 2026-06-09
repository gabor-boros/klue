// Package node contains diagnostic rules for Kubernetes Nodes.
package node

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// NotReadyRule reports nodes whose Ready condition is false or unknown.
type NotReadyRule struct{}

func (NotReadyRule) ID() string { return "node/not-ready" }

func (NotReadyRule) Description() string {
	return "Detects nodes that are not ready"
}

func (NotReadyRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindNode}
}

func (r NotReadyRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	k8sNode, ok := graph.As[*corev1.Node](node)
	if !ok {
		return nil
	}

	condition, found := findNodeCondition(k8sNode.Status.Conditions, corev1.NodeReady)
	if !found || condition.Status == corev1.ConditionTrue {
		return nil
	}

	severity := diagnose.SeverityError
	if condition.Status == corev1.ConditionFalse {
		severity = diagnose.SeverityCritical
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Node is not ready",
			Severity:   severity,
			Confidence: 0.9,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Condition", condition.Message, condition.Reason),
			},
			Explanation: "The kubelet reports the node as not ready, so workloads may fail to schedule or run reliably.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect node conditions and events",
					Command: fmt.Sprintf("kubectl describe node %s", k8sNode.Name),
				},
			},
		},
	}
}

// PressureRule reports nodes under memory, disk, or PID pressure.
type PressureRule struct{}

func (PressureRule) ID() string { return "node/pressure" }

func (PressureRule) Description() string {
	return "Detects node resource pressure conditions"
}

func (PressureRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindNode}
}

func (r PressureRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	k8sNode, ok := graph.As[*corev1.Node](node)
	if !ok {
		return nil
	}

	pressureTypes := []corev1.NodeConditionType{
		corev1.NodeMemoryPressure,
		corev1.NodeDiskPressure,
		corev1.NodePIDPressure,
	}

	var active []corev1.NodeConditionType
	var evidences []diagnose.Evidence
	for _, pressureType := range pressureTypes {
		condition, found := findNodeCondition(k8sNode.Status.Conditions, pressureType)
		if !found || condition.Status != corev1.ConditionTrue {
			continue
		}

		active = append(active, pressureType)
		evidences = append(evidences, diagnose.NewEvidence(node.Ref, "Condition", condition.Message, condition.Reason))
	}

	if len(active) == 0 {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:          r.ID(),
			Title:       fmt.Sprintf("Node reports pressure conditions: %s", joinNodeConditionTypes(active)),
			Severity:    diagnose.SeverityError,
			Confidence:  0.85,
			Resource:    node.Ref,
			Evidence:    evidences,
			Explanation: "The node is under resource pressure and may evict pods or reject new scheduling requests.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect node resource usage and pressure conditions",
					Command: fmt.Sprintf("kubectl describe node %s", k8sNode.Name),
				},
			},
		},
	}
}

// NetworkUnavailableRule reports nodes with unavailable networking.
type NetworkUnavailableRule struct{}

func (NetworkUnavailableRule) ID() string { return "node/network-unavailable" }

func (NetworkUnavailableRule) Description() string {
	return "Detects nodes with network unavailable condition"
}

func (NetworkUnavailableRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindNode}
}

func (r NetworkUnavailableRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	k8sNode, ok := graph.As[*corev1.Node](node)
	if !ok {
		return nil
	}

	condition, found := findNodeCondition(k8sNode.Status.Conditions, corev1.NodeNetworkUnavailable)
	if !found || condition.Status != corev1.ConditionTrue {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Node network is unavailable",
			Severity:   diagnose.SeverityError,
			Confidence: 0.8,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Condition", condition.Message, condition.Reason),
			},
			Explanation: "Pod networking on this node is unavailable, which can break service connectivity.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Inspect CNI/network daemon state on the node",
					Command: fmt.Sprintf("kubectl describe node %s", k8sNode.Name),
				},
			},
		},
	}
}

// UnschedulableRule reports nodes that are cordoned.
type UnschedulableRule struct{}

func (UnschedulableRule) ID() string { return "node/unschedulable" }

func (UnschedulableRule) Description() string {
	return "Detects nodes marked unschedulable"
}

func (UnschedulableRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindNode}
}

func (r UnschedulableRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	k8sNode, ok := graph.As[*corev1.Node](node)
	if !ok || !k8sNode.Spec.Unschedulable {
		return nil
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "Node is marked unschedulable",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.95,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Spec", "node.spec.unschedulable=true", "unschedulable"),
			},
			Explanation: "New pods will not be scheduled to this node while it is cordoned.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Uncordon the node if it should accept new workloads",
					Command: fmt.Sprintf("kubectl uncordon %s", k8sNode.Name),
				},
			},
		},
	}
}

func findNodeCondition(conditions []corev1.NodeCondition, conditionType corev1.NodeConditionType) (corev1.NodeCondition, bool) {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition, true
		}
	}
	return corev1.NodeCondition{}, false
}

func joinNodeConditionTypes(types []corev1.NodeConditionType) string {
	parts := make([]string, 0, len(types))
	for _, conditionType := range types {
		parts = append(parts, string(conditionType))
	}
	return strings.Join(parts, ", ")
}
