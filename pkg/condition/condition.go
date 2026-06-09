// Package condition provides a small, dependency-light model for reading and
// evaluating Kubernetes status conditions. It unifies the condition handling
// that status derivation and diagnostic rules would otherwise duplicate.
package condition

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Condition is the subset of a Kubernetes status condition that klue reasons
// about.
type Condition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// positiveTypes are condition types whose healthy state is "True"; a value of
// "False" therefore indicates a problem.
var positiveTypes = map[string]struct{}{
	"Ready":       {},
	"Available":   {},
	"Healthy":     {},
	"Initialized": {},
}

// negativeTypes are condition types whose healthy state is "False"; a value of
// "True" therefore indicates a problem.
var negativeTypes = map[string]struct{}{
	"Failed":   {},
	"Degraded": {},
}

// FromUnstructured extracts the status.conditions of an unstructured object. It
// returns nil when the object has no readable conditions.
func FromUnstructured(obj *unstructured.Unstructured) []Condition {
	raw, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return nil
	}

	conditions := make([]Condition, 0, len(raw))
	for _, entry := range raw {
		fields, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		condition := Condition{}
		condition.Type, _ = fields["type"].(string)
		condition.Status, _ = fields["status"].(string)
		condition.Reason, _ = fields["reason"].(string)
		condition.Message, _ = fields["message"].(string)
		conditions = append(conditions, condition)
	}

	return conditions
}

// IsFailing reports whether a condition represents a failing state: a positive
// condition (Ready/Available/Healthy/Initialized) reported as False, or a
// negative condition (Failed/Degraded) reported as True.
func IsFailing(c Condition) bool {
	if _, ok := positiveTypes[c.Type]; ok {
		return c.Status == "False"
	}
	if _, ok := negativeTypes[c.Type]; ok {
		return c.Status == "True"
	}
	return false
}

// AnyFailing reports whether any of the conditions is in a failing state.
func AnyFailing(conditions []Condition) bool {
	for i := range conditions {
		if IsFailing(conditions[i]) {
			return true
		}
	}
	return false
}
