package builtin_test

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/evidence"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/builtin"
	"github.com/gabor-boros/klue/pkg/resource"
)

func podRef(name string) resource.Reference {
	return resource.NewReference(resource.ReferenceKindPod, "v1", "default", name, "")
}

func TestWarningEventsRule(t *testing.T) {
	t.Parallel()

	ref := podRef("web")
	events := evidence.NewEventIndex([]corev1.Event{
		{
			ObjectMeta:     metav1.ObjectMeta{Namespace: "default", Name: "e1"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "web", APIVersion: "v1"},
			Type:           corev1.EventTypeWarning,
			Reason:         "BackOff",
			Message:        "Back-off restarting failed container",
		},
	})

	node := &graph.Node{Ref: ref}
	ctx := diagnose.RuleContext{Events: events, Options: diagnose.DefaultDiagnoseOptions()}

	findings := builtin.WarningEventsRule{}.Evaluate(ctx, node)
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}
	if findings[0].Severity != diagnose.SeverityWarning {
		t.Errorf("Severity = %q, want warning", findings[0].Severity)
	}

	// No events -> no findings.
	empty := diagnose.RuleContext{Events: evidence.NewEventIndex(nil), Options: diagnose.DefaultDiagnoseOptions()}
	if got := (builtin.WarningEventsRule{}).Evaluate(empty, node); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 without events", len(got))
	}
}

func TestFailedConditionRule(t *testing.T) {
	t.Parallel()

	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiregistration.k8s.io/v1",
		"kind":       "APIService",
		"metadata":   map[string]any{"name": "v1beta1.metrics.k8s.io"},
		"status": map[string]any{
			"conditions": []any{
				map[string]any{"type": "Available", "status": "False", "reason": "FailedDiscoveryCheck", "message": "no response"},
			},
		},
	}}

	node := &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindAPIService, "apiregistration.k8s.io/v1", "", "v1beta1.metrics.k8s.io", ""),
		Object: obj,
	}

	findings := builtin.FailedConditionRule{}.Evaluate(diagnose.RuleContext{}, node)
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1", len(findings))
	}

	// A typed object is left to its dedicated rules.
	typedNode := &graph.Node{Ref: podRef("web"), Object: &corev1.Pod{}}
	if got := (builtin.FailedConditionRule{}).Evaluate(diagnose.RuleContext{}, typedNode); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for typed object", len(got))
	}
}

func TestTerminatingStuckRule(t *testing.T) {
	t.Parallel()

	now := time.Now()
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace:         "default",
		Name:              "stuck",
		DeletionTimestamp: &metav1.Time{Time: now.Add(-30 * time.Minute)},
		Finalizers:        []string{"example.com/protect"},
	}}
	node := &graph.Node{Ref: podRef("stuck"), Object: pod}

	ctx := diagnose.RuleContext{Options: diagnose.DiagnoseOptions{Now: now}}
	if got := (builtin.TerminatingStuckRule{}).Evaluate(ctx, node); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for stuck terminating pod", len(got))
	}

	// Without a reference clock the rule stays silent for determinism.
	if got := (builtin.TerminatingStuckRule{}).Evaluate(diagnose.RuleContext{}, node); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 without a reference time", len(got))
	}

	// Recently deleted pods are within the grace period.
	fresh := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace:         "default",
		Name:              "fresh",
		DeletionTimestamp: &metav1.Time{Time: now.Add(-10 * time.Second)},
	}}
	freshNode := &graph.Node{Ref: podRef("fresh"), Object: fresh}
	if got := (builtin.TerminatingStuckRule{}).Evaluate(ctx, freshNode); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 within grace period", len(got))
	}
}

func TestOrphanedOwnerRule(t *testing.T) {
	t.Parallel()

	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Namespace: "default",
		Name:      "orphan",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "ReplicaSet",
			Name:       "missing-rs",
			UID:        "missing-uid",
		}},
	}}
	node := &graph.Node{Ref: podRef("orphan"), Object: pod}

	builder := graph.NewBuilder()
	builder.AddNode(*node)
	g := builder.Build()

	if got := (builtin.OrphanedOwnerRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for missing owner", len(got))
	}

	// Add the owner; the finding disappears.
	builder.AddNode(graph.Node{Ref: resource.NewReference(resource.ReferenceKindReplicaSet, "apps/v1", "default", "missing-rs", "missing-uid")})
	g = builder.Build()
	if got := (builtin.OrphanedOwnerRule{}).Evaluate(diagnose.RuleContext{Graph: g}, node); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 once the owner exists", len(got))
	}
}

func TestMissingReferenceRule(t *testing.T) {
	t.Parallel()

	source := graph.Node{
		Ref: resource.NewReference(resource.ReferenceKindIngress, "networking.k8s.io/v1", "default", "web", ""),
	}
	missing := graph.Node{
		Ref:         resource.NewReference(resource.ReferenceKindSecret, "v1", "default", "web-tls", ""),
		Status:      resource.StatusMissing,
		Placeholder: &graph.Placeholder{},
	}

	builder := graph.NewBuilder()
	builder.AddNode(source)
	builder.AddNode(missing)
	builder.AddEdge(graph.Edge{Kind: graph.EdgeUsesSecret, From: source, To: missing, Reason: "tls", Data: graph.EdgeData{Path: "spec.tls[].secretName"}})

	findings := builtin.MissingReferenceRule{}.Evaluate(diagnose.RuleContext{Graph: builder.Build()}, &missing)
	if len(findings) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for unresolved placeholder", len(findings))
	}
	if findings[0].ID != "builtin/missing-reference" {
		t.Fatalf("ID = %q, want %q", findings[0].ID, "builtin/missing-reference")
	}
}

func TestMissingReferenceRuleSkipsProducerBackedPlaceholder(t *testing.T) {
	t.Parallel()

	consumer := graph.Node{Ref: resource.NewReference(resource.ReferenceKindIngress, "networking.k8s.io/v1", "default", "web", "")}
	producer := graph.Node{Ref: resource.NewReference("Certificate", "cert-manager.io/v1", "default", "web-cert", "")}
	missing := graph.Node{
		Ref:         resource.NewReference(resource.ReferenceKindSecret, "v1", "default", "web-tls", ""),
		Status:      resource.StatusMissing,
		Placeholder: &graph.Placeholder{},
	}

	builder := graph.NewBuilder()
	builder.AddNode(consumer)
	builder.AddNode(producer)
	builder.AddNode(missing)
	builder.AddEdge(graph.Edge{Kind: graph.EdgeUsesSecret, From: consumer, To: missing, Reason: "tls"})
	builder.AddEdge(graph.Edge{Kind: graph.EdgeProduces, From: producer, To: missing, Reason: "secretName"})

	if got := (builtin.MissingReferenceRule{}).Evaluate(diagnose.RuleContext{Graph: builder.Build()}, &missing); len(got) != 0 {
		t.Fatalf("Evaluate() = %d findings, want 0 when a producer edge exists", len(got))
	}
}
