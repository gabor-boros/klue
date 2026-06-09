package kube

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// DynamicRelationships extracts reusable references from unstructured objects.
// It combines conservative generic patterns with a small data-driven registry
// for controller-specific producer/reference semantics.
func DynamicRelationships(entry APIResource, obj *unstructured.Unstructured, resolver RelationshipScopeResolver) []Relationship {
	relationships := make([]Relationship, 0, 8)
	relationships = append(relationships, certManagerRelationships(entry, obj, resolver)...)
	relationships = append(relationships, genericSpecRelationships(entry, obj, resolver)...)

	if len(relationships) == 0 {
		return nil
	}

	sortRelationships(relationships)
	return dedupeRelationships(relationships)
}

func certManagerRelationships(entry APIResource, obj *unstructured.Unstructured, resolver RelationshipScopeResolver) []Relationship {
	if entry.APIVersion() != "cert-manager.io/v1" {
		return nil
	}

	relationships := make([]Relationship, 0, 3)

	if entry.Kind == resource.Kind("Certificate") {
		if name, found, err := unstructured.NestedString(obj.Object, "spec", "secretName"); err == nil && found && name != "" {
			relationships = append(relationships, Relationship{
				EdgeKind: graph.EdgeProduces,
				Target: resource.NewReference(
					resource.ReferenceKindSecret,
					apiVersionCore,
					namespaceForTarget(resource.ReferenceKindSecret, apiVersionCore, obj.GetNamespace(), "", resolver),
					name,
					"",
				),
				Reason: "secretName",
				Path:   "spec.secretName",
				Role:   RelationshipRoleProduces,
			})
		}

		if ref, found, err := unstructured.NestedMap(obj.Object, "spec", "issuerRef"); err == nil && found {
			name, _ := ref["name"].(string)
			if name != "" {
				kind := resource.Kind("Issuer")
				if rawKind, ok := ref["kind"].(string); ok && rawKind != "" {
					kind = resource.Kind(rawKind)
				}
				group := "cert-manager.io"
				if rawGroup, ok := ref["group"].(string); ok && rawGroup != "" {
					group = rawGroup
				}
				targetAPIVersion := fmt.Sprintf("%s/%s", group, entry.Version)
				targetNamespace := namespaceForTarget(kind, targetAPIVersion, obj.GetNamespace(), "", resolver)

				relationships = append(relationships, Relationship{
					EdgeKind: graph.EdgeReferences,
					Target:   resource.NewReference(kind, targetAPIVersion, targetNamespace, name, ""),
					Reason:   "issuerRef",
					Path:     "spec.issuerRef",
					Role:     RelationshipRoleReferences,
				})
			}
		}
	}

	return relationships
}

func genericSpecRelationships(entry APIResource, obj *unstructured.Unstructured, resolver RelationshipScopeResolver) []Relationship {
	spec, found, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !found {
		return nil
	}

	relationships := make([]Relationship, 0, 6)
	walkStructured(spec, "spec", func(path string, key string, value any) {
		kind := kindFromSecretConfigKey(key)
		if kind != "" {
			name, ok := value.(string)
			if ok && name != "" {
				// Certificate.spec.secretName represents producer semantics and is
				// handled by cert-manager-specific extraction.
				if entry.APIVersion() == "cert-manager.io/v1" && entry.Kind == resource.Kind("Certificate") && path == "spec.secretName" {
					return
				}
				relationships = append(relationships, Relationship{
					EdgeKind: edgeForTarget(kind),
					Target: resource.NewReference(
						kind,
						apiVersionForKind(kind),
						namespaceForTarget(kind, apiVersionForKind(kind), obj.GetNamespace(), "", resolver),
						name,
						"",
					),
					Reason: key,
					Path:   path,
					Role:   RelationshipRoleUses,
				})
			}
		}

		if mapValue, ok := value.(map[string]any); ok {
			if rel, ok := localRefRelationship(obj.GetNamespace(), key, path, mapValue, resolver); ok {
				relationships = append(relationships, rel)
			}
			if rel, ok := objectRefRelationship(entry, obj.GetNamespace(), path, mapValue, resolver); ok {
				relationships = append(relationships, rel)
			}
		}
	})

	return relationships
}

func walkStructured(value any, path string, visit func(path string, key string, value any)) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPath := path + "." + key
			child := typed[key]
			visit(nextPath, key, child)
			walkStructured(child, nextPath, visit)
		}
	case []any:
		for i := range typed {
			nextPath := fmt.Sprintf("%s[%d]", path, i)
			visit(nextPath, "", typed[i])
			walkStructured(typed[i], nextPath, visit)
		}
	}
}

func kindFromSecretConfigKey(key string) resource.Kind {
	switch {
	case strings.EqualFold(key, "secretName"):
		return resource.ReferenceKindSecret
	case strings.EqualFold(key, "configMapName"):
		return resource.ReferenceKindConfigMap
	default:
		return ""
	}
}

func localRefRelationship(sourceNamespace, key, path string, value map[string]any, resolver RelationshipScopeResolver) (Relationship, bool) {
	name, _ := value["name"].(string)
	if name == "" {
		return Relationship{}, false
	}

	var kind resource.Kind
	switch {
	case strings.EqualFold(key, "secretRef"),
		strings.EqualFold(key, "secretKeyRef"),
		strings.HasSuffix(key, "SecretRef"),
		strings.HasSuffix(key, "secretRef"):
		kind = resource.ReferenceKindSecret
	case strings.EqualFold(key, "configMapRef"),
		strings.EqualFold(key, "configMapKeyRef"),
		strings.HasSuffix(key, "ConfigMapRef"),
		strings.HasSuffix(key, "configMapRef"):
		kind = resource.ReferenceKindConfigMap
	default:
		return Relationship{}, false
	}

	apiVersion := apiVersionForKind(kind)
	targetNamespace := namespaceForTarget(kind, apiVersion, sourceNamespace, "", resolver)

	return Relationship{
		EdgeKind: edgeForTarget(kind),
		Target:   resource.NewReference(kind, apiVersion, targetNamespace, name, ""),
		Reason:   key,
		Path:     path,
		Role:     RelationshipRoleUses,
	}, true
}

func objectRefRelationship(entry APIResource, sourceNamespace, path string, value map[string]any, resolver RelationshipScopeResolver) (Relationship, bool) {
	name, _ := value["name"].(string)
	kindString, _ := value["kind"].(string)
	if name == "" || kindString == "" {
		return Relationship{}, false
	}

	apiVersion, ok := inferReferenceAPIVersion(entry, resource.Kind(kindString), value)
	if !ok {
		return Relationship{}, false
	}

	explicitNamespace, _ := value["namespace"].(string)
	targetNamespace := namespaceForTarget(resource.Kind(kindString), apiVersion, sourceNamespace, explicitNamespace, resolver)
	return Relationship{
		EdgeKind: graph.EdgeReferences,
		Target:   resource.NewReference(resource.Kind(kindString), apiVersion, targetNamespace, name, ""),
		Reason:   lastPathToken(path),
		Path:     path,
		Role:     RelationshipRoleReferences,
	}, true
}

func inferReferenceAPIVersion(entry APIResource, kind resource.Kind, value map[string]any) (string, bool) {
	if apiVersion, ok := value["apiVersion"].(string); ok && apiVersion != "" {
		return apiVersion, true
	}

	if group, ok := value["group"].(string); ok && group != "" {
		version := entry.Version
		gv, err := schema.ParseGroupVersion(entry.APIVersion())
		if err == nil && gv.Group == group && gv.Version != "" {
			version = gv.Version
		}
		if version == "" {
			version = "v1"
		}
		return fmt.Sprintf("%s/%s", group, version), true
	}

	if builtin, found := LookupKind(kind); found {
		return builtin.APIVersion(), true
	}

	gv, err := schema.ParseGroupVersion(entry.APIVersion())
	if err == nil && gv.Group != "" {
		return fmt.Sprintf("%s/%s", gv.Group, entry.Version), true
	}
	return "", false
}

func lastPathToken(path string) string {
	if path == "" {
		return ""
	}
	clean := path
	if idx := strings.LastIndex(clean, "."); idx >= 0 {
		clean = clean[idx+1:]
	}
	if idx := strings.Index(clean, "["); idx >= 0 {
		clean = clean[:idx]
	}
	return clean
}

func apiVersionForKind(kind resource.Kind) string {
	if kind == resource.ReferenceKindSecret || kind == resource.ReferenceKindConfigMap {
		return apiVersionCore
	}
	if entry, found := LookupKind(kind); found {
		return entry.APIVersion()
	}
	return apiVersionCore
}

func edgeForTarget(kind resource.Kind) graph.EdgeKind {
	switch kind {
	case resource.ReferenceKindSecret:
		return graph.EdgeUsesSecret
	case resource.ReferenceKindConfigMap:
		return graph.EdgeUsesConfigMap
	default:
		return graph.EdgeReferences
	}
}

func dedupeRelationships(relationships []Relationship) []Relationship {
	seen := make(map[string]struct{}, len(relationships))
	out := make([]Relationship, 0, len(relationships))
	for _, relationship := range relationships {
		key := fmt.Sprintf("%s|%s|%s|%s", relationship.Target.LogicalKey(), relationship.EdgeKind, relationship.Path, relationship.Reason)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, relationship)
	}
	return out
}
