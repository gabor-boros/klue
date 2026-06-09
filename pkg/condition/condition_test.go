package condition_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/gabor-boros/klue/pkg/condition"
)

func TestIsFailing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cond condition.Condition
		want bool
	}{
		{"ready false is failing", condition.Condition{Type: "Ready", Status: "False"}, true},
		{"ready true is healthy", condition.Condition{Type: "Ready", Status: "True"}, false},
		{"available false is failing", condition.Condition{Type: "Available", Status: "False"}, true},
		{"degraded true is failing", condition.Condition{Type: "Degraded", Status: "True"}, true},
		{"degraded false is healthy", condition.Condition{Type: "Degraded", Status: "False"}, false},
		{"unknown type is healthy", condition.Condition{Type: "Custom", Status: "False"}, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := condition.IsFailing(tc.cond); got != tc.want {
				t.Errorf("IsFailing(%+v) = %v, want %v", tc.cond, got, tc.want)
			}
		})
	}
}

func TestFromUnstructured(t *testing.T) {
	t.Parallel()

	obj := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"conditions": []any{
				map[string]any{
					"type":    "Ready",
					"status":  "False",
					"reason":  "Stalled",
					"message": "not ready yet",
				},
				map[string]any{
					"type":   "Synced",
					"status": "True",
				},
			},
		},
	}}

	conditions := condition.FromUnstructured(obj)
	if len(conditions) != 2 {
		t.Fatalf("FromUnstructured() returned %d conditions, want 2", len(conditions))
	}
	if conditions[0].Reason != "Stalled" || conditions[0].Message != "not ready yet" {
		t.Errorf("first condition = %+v, want reason/message preserved", conditions[0])
	}
	if !condition.AnyFailing(conditions) {
		t.Error("AnyFailing() = false, want true for Ready=False")
	}
}

func TestFromUnstructuredNoConditions(t *testing.T) {
	t.Parallel()

	obj := &unstructured.Unstructured{Object: map[string]any{"status": map[string]any{}}}
	if got := condition.FromUnstructured(obj); got != nil {
		t.Errorf("FromUnstructured() = %v, want nil when no conditions present", got)
	}
}
