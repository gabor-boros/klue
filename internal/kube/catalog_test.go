package kube_test

import (
	"testing"

	"github.com/gabor-boros/klue/internal/kube"
	"github.com/gabor-boros/klue/pkg/resource"
)

func TestBuiltinCatalogIsConsistent(t *testing.T) {
	t.Parallel()

	catalog := kube.BuiltinCatalog()
	if len(catalog) == 0 {
		t.Fatal("BuiltinCatalog() is empty")
	}

	tokens := make(map[string]string)
	for _, entry := range catalog {
		if entry.Version == "" || entry.Resource == "" || entry.Kind == "" {
			t.Errorf("incomplete catalog entry: %+v", entry)
		}

		token := entry.CommandToken()
		if existing, ok := tokens[token]; ok {
			t.Errorf("duplicate command token %q used by %q and %q", token, existing, entry.Kind)
		}
		tokens[token] = string(entry.Kind)
	}

	// Aliases must not collide with primary tokens or each other.
	for _, entry := range catalog {
		for _, alias := range entry.Aliases {
			if owner, ok := tokens[alias]; ok && owner != string(entry.Kind) {
				t.Errorf("alias %q for %q collides with command token of %q", alias, entry.Kind, owner)
			}
			if _, ok := tokens[alias]; !ok {
				tokens[alias] = string(entry.Kind)
			}
		}
	}
}

func TestIsBuiltinGroup(t *testing.T) {
	t.Parallel()

	builtin := []string{"", "apps", "batch", "rbac.authorization.k8s.io", "apiregistration.k8s.io"}
	for _, group := range builtin {
		if !kube.IsBuiltinGroup(group) {
			t.Errorf("IsBuiltinGroup(%q) = false, want true", group)
		}
	}

	if kube.IsBuiltinGroup("example.com") {
		t.Error("IsBuiltinGroup(\"example.com\") = true, want false (CRD group)")
	}
}

func TestIsSubresource(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"pods":              false,
		"pods/status":       true,
		"deployments/scale": true,
		"nodes":             false,
	}
	for name, want := range cases {
		if got := kube.IsSubresource(name); got != want {
			t.Errorf("IsSubresource(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestLookupCommandTokenAndKind(t *testing.T) {
	t.Parallel()

	entry, ok := kube.LookupCommandToken("po")
	if !ok || entry.Kind != resource.ReferenceKindPod {
		t.Fatalf("LookupCommandToken(\"po\") = %+v, %v; want Pod entry", entry, ok)
	}

	entry, ok = kube.LookupKind(resource.ReferenceKindHorizontalPodAutoscaler)
	if !ok || entry.Resource != "horizontalpodautoscalers" || entry.APIVersion() != "autoscaling/v2" {
		t.Fatalf("LookupKind(HPA) = %+v, %v; want autoscaling/v2 hpa entry", entry, ok)
	}
	if entry.Namespaced != true {
		t.Errorf("HPA Namespaced = %v, want true", entry.Namespaced)
	}

	if _, ok := kube.LookupCommandToken("definitely-not-a-resource"); ok {
		t.Error("LookupCommandToken returned ok for unknown token")
	}
}

func TestAPIVersionFormatting(t *testing.T) {
	t.Parallel()

	pod, _ := kube.LookupKind(resource.ReferenceKindPod)
	if pod.APIVersion() != "v1" {
		t.Errorf("Pod APIVersion() = %q, want v1", pod.APIVersion())
	}

	deploy, _ := kube.LookupKind(resource.ReferenceKindDeployment)
	if deploy.APIVersion() != "apps/v1" {
		t.Errorf("Deployment APIVersion() = %q, want apps/v1", deploy.APIVersion())
	}
}
