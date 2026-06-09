package resource_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gabor-boros/klue/pkg/resource"
)

func TestNewReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName   string
		kind       resource.Kind
		apiVersion string
		namespace  string
		name       string
		uid        string
		expected   resource.Reference
	}{
		{
			testName:   "Full reference",
			kind:       resource.ReferenceKindPod,
			apiVersion: "v1",
			namespace:  "my-namespace",
			name:       "test-pod",
			uid:        "1234567890",
			expected: resource.Reference{
				Kind:       resource.ReferenceKindPod,
				APIVersion: "v1",
				Namespace:  "my-namespace",
				Name:       "test-pod",
				UID:        "1234567890",
			},
		},
		{
			testName:   "Missing namespace defaults to default",
			kind:       resource.ReferenceKindPod,
			apiVersion: "v1",
			namespace:  "",
			name:       "test-pod",
			uid:        "1234567890",
			expected: resource.Reference{
				Kind:       resource.ReferenceKindPod,
				APIVersion: "v1",
				Namespace:  "default",
				Name:       "test-pod",
				UID:        "1234567890",
			},
		},
		{
			testName:   "Missing UID",
			kind:       resource.ReferenceKindPod,
			apiVersion: "v1",
			namespace:  "my-namespace",
			name:       "test-pod",
			uid:        "",
			expected: resource.Reference{
				Kind:       resource.ReferenceKindPod,
				APIVersion: "v1",
				Namespace:  "my-namespace",
				Name:       "test-pod",
				UID:        "",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			got := resource.NewReference(tt.kind, tt.apiVersion, tt.namespace, tt.name, tt.uid)
			if got != tt.expected {
				t.Errorf("NewReference() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestReferenceFromObjectReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName string
		input    corev1.ObjectReference
		expected resource.Reference
	}{
		{
			testName: "Full reference",
			input: corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Pod",
				Namespace:  "my-namespace",
				Name:       "test-pod",
				UID:        types.UID("1234567890"),
			},
			expected: resource.Reference{
				APIVersion: "v1",
				Kind:       resource.ReferenceKindPod,
				Namespace:  "my-namespace",
				Name:       "test-pod",
				UID:        "1234567890",
			},
		},
		{
			testName: "Missing namespace defaults to default",
			input: corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       "test-pod",
			},
			expected: resource.Reference{
				APIVersion: "v1",
				Kind:       resource.ReferenceKindPod,
				Namespace:  "default",
				Name:       "test-pod",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			got := resource.ReferenceFromObjectReference(tt.input)
			if got != tt.expected {
				t.Errorf("ReferenceFromObjectReference() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestNewReference_Key(t *testing.T) {
	testCases := []struct {
		testName   string
		uid        string
		apiVersion string
		kind       resource.Kind
		namespace  string
		name       string
		expected   string
	}{
		{
			testName:   "With UID and namespace",
			uid:        "1234567890",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "default",
			name:       "test-pod",
			expected:   "uid/1234567890",
		},
		{
			testName:   "With UID and without namespace",
			uid:        "1234567890",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "",
			name:       "test-pod",
			expected:   "uid/1234567890",
		},
		{
			testName:   "Without UID and namespace",
			uid:        "",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "",
			name:       "test-pod",
			expected:   "v1/Pod/default/test-pod",
		},
		{
			testName:   "Without UID and with other namespace",
			uid:        "",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "my-namespace",
			name:       "test-pod",
			expected:   "v1/Pod/my-namespace/test-pod",
		},
		{
			testName:   "Without UID and without namespace",
			uid:        "",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "",
			name:       "test-pod",
			expected:   "v1/Pod/default/test-pod",
		},
	}
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			Reference := resource.NewReference(tt.kind, tt.apiVersion, tt.namespace, tt.name, tt.uid)
			if Reference.Key() != tt.expected {
				t.Errorf("Key() = %v, want Key %s", Reference.Key(), tt.expected)
			}
		})
	}
}

func TestNewReference_LogicalKey(t *testing.T) {
	tests := []struct {
		testName   string
		apiVersion string
		kind       resource.Kind
		namespace  string
		name       string
		expected   string
	}{
		{
			testName:   "With namespace",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "default",
			name:       "test-pod",
			expected:   "v1/Pod/default/test-pod",
		},
		{
			testName:   "Without namespace",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "",
			name:       "test-pod",
			expected:   "v1/Pod/default/test-pod",
		},
		{
			testName:   "With other namespace",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "my-namespace",
			name:       "test-pod",
			expected:   "v1/Pod/my-namespace/test-pod",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			Reference := resource.NewReference(tt.kind, tt.apiVersion, tt.namespace, tt.name, "")
			if Reference.LogicalKey() != tt.expected {
				t.Errorf("LogicalKey() = %v, want LogicalKey %s", Reference.LogicalKey(), tt.expected)
			}
		})
	}
}

func TestNewReference_Display(t *testing.T) {
	tests := []struct {
		testName   string
		apiVersion string
		kind       resource.Kind
		namespace  string
		name       string
		uid        string
		expected   string
	}{
		{
			testName:   "With UID and namespace",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "default",
			name:       "test-pod",
			uid:        "1234567890",
			expected:   "Pod/default/test-pod (uid: 1234567890)",
		},
		{
			testName:   "With UID and without namespace",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "",
			name:       "test-pod",
			uid:        "1234567890",
			expected:   "Pod/default/test-pod (uid: 1234567890)",
		},
		{
			testName:   "With UID and other namespace",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "my-namespace",
			name:       "test-pod",
			uid:        "1234567890",
			expected:   "Pod/my-namespace/test-pod (uid: 1234567890)",
		},
		{
			testName:   "Without UID and without namespace",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "my-namespace",
			name:       "test-pod",
			uid:        "",
			expected:   "Pod/my-namespace/test-pod",
		},
		{
			testName:   "Without UID and other namespace",
			apiVersion: "v1",
			kind:       resource.ReferenceKindPod,
			namespace:  "my-namespace",
			name:       "test-pod",
			uid:        "",
			expected:   "Pod/my-namespace/test-pod",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()

			Reference := resource.NewReference(tt.kind, tt.apiVersion, tt.namespace, tt.name, tt.uid)
			if Reference.Display() != tt.expected {
				t.Errorf("NewReference() = %v, want Display %s", Reference.Display(), tt.expected)
			}
		})
	}
}
