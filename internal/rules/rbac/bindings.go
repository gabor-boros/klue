// Package rbac contains diagnostic rules for RBAC bindings.
package rbac

import (
	rbacv1 "k8s.io/api/rbac/v1"
)

// bindingRoleRef extracts the role reference, namespace and name from a Role or
// ClusterRole binding object.
func bindingRoleRef(obj any) (rbacv1.RoleRef, string, bool) {
	switch binding := obj.(type) {
	case *rbacv1.RoleBinding:
		return binding.RoleRef, binding.Name, true
	case *rbacv1.ClusterRoleBinding:
		return binding.RoleRef, binding.Name, true
	default:
		return rbacv1.RoleRef{}, "", false
	}
}

// bindingHasSubjects reports whether the binding lists at least one subject.
func bindingHasSubjects(obj any) bool {
	switch binding := obj.(type) {
	case *rbacv1.RoleBinding:
		return len(binding.Subjects) > 0
	case *rbacv1.ClusterRoleBinding:
		return len(binding.Subjects) > 0
	default:
		return true
	}
}
