package deployment_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/deployment"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(deploy *appsv1.Deployment) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindDeployment, "apps/v1", deploy.Namespace, deploy.Name, string(deploy.UID)),
		Object: deploy,
	}
}

func replicas(n int32) *int32 { return &n }

func TestUnavailableRule(t *testing.T) {
	t.Parallel()

	degraded := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec:       appsv1.DeploymentSpec{Replicas: replicas(3)},
		Status:     appsv1.DeploymentStatus{AvailableReplicas: 1},
	}
	if got := (deployment.UnavailableRule{}).Evaluate(diagnose.RuleContext{}, node(degraded)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 when replicas unavailable", len(got))
	}

	healthy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec:       appsv1.DeploymentSpec{Replicas: replicas(3)},
		Status:     appsv1.DeploymentStatus{AvailableReplicas: 3},
	}
	if got := (deployment.UnavailableRule{}).Evaluate(diagnose.RuleContext{}, node(healthy)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 when fully available", len(got))
	}
}

func TestRolloutStuckRule(t *testing.T) {
	t.Parallel()

	stuck := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec:       appsv1.DeploymentSpec{Replicas: replicas(3)},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:    appsv1.DeploymentProgressing,
					Reason:  "ProgressDeadlineExceeded",
					Message: "exceeded its progress deadline",
				},
			},
		},
	}
	if got := (deployment.RolloutStuckRule{}).Evaluate(diagnose.RuleContext{}, node(stuck)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for stalled rollout", len(got))
	}

	progressing := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec:       appsv1.DeploymentSpec{Replicas: replicas(3)},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentProgressing,
					Reason: "NewReplicaSetAvailable",
				},
			},
		},
	}
	if got := (deployment.RolloutStuckRule{}).Evaluate(diagnose.RuleContext{}, node(progressing)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 when progressing normally", len(got))
	}
}
