package node_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/node"
	"github.com/gabor-boros/klue/pkg/resource"
)

func nodeGraphNode(name string, nodeObj *corev1.Node) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindNode, "v1", "", name, string(nodeObj.UID)),
		Object: nodeObj,
	}
}

func TestNotReadyRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		condition corev1.ConditionStatus
		want      int
		severity  diagnose.Severity
	}{
		{name: "ready true", condition: corev1.ConditionTrue, want: 0},
		{name: "ready false", condition: corev1.ConditionFalse, want: 1, severity: diagnose.SeverityCritical},
		{name: "ready unknown", condition: corev1.ConditionUnknown, want: 1, severity: diagnose.SeverityError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nodeObj := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: tt.condition, Reason: "Test", Message: "ready status"},
					},
				},
			}

			findings := (node.NotReadyRule{}).Evaluate(diagnose.RuleContext{}, nodeGraphNode("worker-1", nodeObj))
			if len(findings) != tt.want {
				t.Fatalf("Evaluate() findings = %d, want %d", len(findings), tt.want)
			}
			if tt.want == 1 && findings[0].Severity != tt.severity {
				t.Fatalf("Evaluate() severity = %s, want %s", findings[0].Severity, tt.severity)
			}
		})
	}
}

func TestPressureRule(t *testing.T) {
	t.Parallel()

	nodeObj := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue, Message: "memory pressure"},
			},
		},
	}

	findings := (node.PressureRule{}).Evaluate(diagnose.RuleContext{}, nodeGraphNode("worker-1", nodeObj))
	if len(findings) != 1 {
		t.Fatalf("Evaluate() findings = %d, want 1", len(findings))
	}
}

func TestNetworkUnavailableRule(t *testing.T) {
	t.Parallel()

	nodeObj := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeNetworkUnavailable, Status: corev1.ConditionTrue, Message: "network not configured"},
			},
		},
	}

	findings := (node.NetworkUnavailableRule{}).Evaluate(diagnose.RuleContext{}, nodeGraphNode("worker-1", nodeObj))
	if len(findings) != 1 {
		t.Fatalf("Evaluate() findings = %d, want 1", len(findings))
	}
}

func TestUnschedulableRule(t *testing.T) {
	t.Parallel()

	nodeObj := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Spec:       corev1.NodeSpec{Unschedulable: true},
	}

	findings := (node.UnschedulableRule{}).Evaluate(diagnose.RuleContext{}, nodeGraphNode("worker-1", nodeObj))
	if len(findings) != 1 {
		t.Fatalf("Evaluate() findings = %d, want 1", len(findings))
	}
	if findings[0].Severity != diagnose.SeverityWarning {
		t.Fatalf("Evaluate() severity = %s, want %s", findings[0].Severity, diagnose.SeverityWarning)
	}
}
