package hpa_test

import (
	"testing"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/hpa"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(h *autoscalingv2.HorizontalPodAutoscaler) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindHorizontalPodAutoscaler, "autoscaling/v2", h.Namespace, h.Name, string(h.UID)),
		Object: h,
	}
}

func TestScalingDisabledRule(t *testing.T) {
	t.Parallel()

	disabled := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{
				{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionFalse, Reason: "FailedGetResourceMetric", Message: "no metrics"},
			},
		},
	}
	if got := (hpa.ScalingDisabledRule{}).Evaluate(diagnose.RuleContext{}, node(disabled)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for scaling disabled", len(got))
	}

	active := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{
				{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionTrue},
			},
		},
	}
	if got := (hpa.ScalingDisabledRule{}).Evaluate(diagnose.RuleContext{}, node(active)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 when scaling active", len(got))
	}
}

func TestMissingScaleTargetRule(t *testing.T) {
	t.Parallel()

	h := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "web"},
		},
	}

	builder := graph.NewBuilder()
	builder.AddNode(*node(h))
	g := builder.Build()
	if got := (hpa.MissingScaleTargetRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node(h)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for missing target", len(got))
	}

	builder.AddNode(graph.Node{Ref: resource.NewReference(resource.ReferenceKindDeployment, "apps/v1", "default", "web", "")})
	g = builder.Build()
	if got := (hpa.MissingScaleTargetRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node(h)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 once the target exists", len(got))
	}
}
