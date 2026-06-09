package pod_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/pod"
	"github.com/gabor-boros/klue/pkg/resource"
)

func podNode(p *corev1.Pod) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindPod, "v1", p.Namespace, p.Name, string(p.UID)),
		Object: p,
	}
}

func TestCrashLoopRule(t *testing.T) {
	t.Parallel()

	crashing := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					RestartCount: 5,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff", Message: "back-off"},
					},
				},
			},
		},
	}

	findings := pod.CrashLoopRule{}.Evaluate(diagnose.RuleContext{}, podNode(crashing))
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if findings[0].Severity != diagnose.SeverityCritical {
		t.Errorf("Severity = %q, want %q", findings[0].Severity, diagnose.SeverityCritical)
	}

	healthy := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ok"}}
	if got := (pod.CrashLoopRule{}).Evaluate(diagnose.RuleContext{}, podNode(healthy)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for healthy pod", len(got))
	}
}

func TestConfigMissingRule(t *testing.T) {
	t.Parallel()

	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{Name: "cfg", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "missing-cfg"}}}},
			},
		},
	}

	builder := graph.NewBuilder()
	builder.AddNode(*podNode(p))
	g := builder.Build()

	findings := (pod.ConfigMissingRule{}).Evaluate(diagnose.RuleContext{Graph: g}, podNode(p))
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 (missing configmap)", len(findings))
	}

	// Now add the configmap; the finding should disappear.
	builder.AddNode(graph.Node{Ref: resource.NewReference(resource.ReferenceKindConfigMap, "v1", "default", "missing-cfg", "")})
	g = builder.Build()
	if got := (pod.ConfigMissingRule{}).Evaluate(diagnose.RuleContext{Graph: g}, podNode(p)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 once configmap exists", len(got))
	}
}
