package rbac_test

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/rbac"
	"github.com/gabor-boros/klue/pkg/resource"
)

func bindingNode(rb *rbacv1.RoleBinding) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindRoleBinding, "rbac.authorization.k8s.io/v1", rb.Namespace, rb.Name, string(rb.UID)),
		Object: rb,
	}
}

func TestMissingRoleRule(t *testing.T) {
	t.Parallel()

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "read-binding"},
		RoleRef:    rbacv1.RoleRef{Kind: "Role", Name: "reader"},
		Subjects:   []rbacv1.Subject{{Kind: "User", Name: "alice"}},
	}

	builder := graph.NewBuilder()
	builder.AddNode(*bindingNode(rb))
	g := builder.Build()
	if got := (rbac.MissingRoleRule{}).Evaluate(diagnose.RuleContext{Graph: g}, bindingNode(rb)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for missing role", len(got))
	}

	builder.AddNode(graph.Node{Ref: resource.NewReference(resource.ReferenceKindRole, "rbac.authorization.k8s.io/v1", "default", "reader", "")})
	g = builder.Build()
	if got := (rbac.MissingRoleRule{}).Evaluate(diagnose.RuleContext{Graph: g}, bindingNode(rb)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 once the role exists", len(got))
	}
}

func TestNoSubjectsRule(t *testing.T) {
	t.Parallel()

	empty := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "read-binding"},
		RoleRef:    rbacv1.RoleRef{Kind: "Role", Name: "reader"},
	}
	if got := (rbac.NoSubjectsRule{}).Evaluate(diagnose.RuleContext{}, bindingNode(empty)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for binding without subjects", len(got))
	}

	withSubjects := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "read-binding"},
		RoleRef:    rbacv1.RoleRef{Kind: "Role", Name: "reader"},
		Subjects:   []rbacv1.Subject{{Kind: "User", Name: "alice"}},
	}
	if got := (rbac.NoSubjectsRule{}).Evaluate(diagnose.RuleContext{}, bindingNode(withSubjects)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 when subjects exist", len(got))
	}
}
