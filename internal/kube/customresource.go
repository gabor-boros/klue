package kube

import (
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"

	"github.com/gabor-boros/klue/pkg/resource"
)

// listVerb is the API verb a resource must support to be listable, and
// therefore fetchable for the diagnosis graph.
const listVerb = "list"

// resourceDiscoverer is the narrow slice of the discovery API klue depends on.
// It is satisfied by discovery.DiscoveryInterface and is easy to fake in tests.
type resourceDiscoverer interface {
	ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error)
}

// discoverCustomResources returns descriptors for every listable custom
// resource served by the cluster. Built-in Kubernetes groups are skipped; only
// custom (CRD-backed) groups are returned so they can be fetched dynamically.
//
// Discovery is best-effort: a partial discovery failure (one or more API groups
// failing to load, common when an aggregated API server is unhealthy) is
// tolerated and the resources that did load are still returned.
func discoverCustomResources(d resourceDiscoverer) ([]APIResource, error) {
	groups, lists, err := d.ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, fmt.Errorf("discover server resources: %w", err)
	}

	preferred := preferredGroupVersions(groups)

	var out []APIResource
	seen := make(map[string]struct{})

	for _, list := range lists {
		if list == nil {
			continue
		}

		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}

		// Skip the core group and any Kubernetes built-in group; those are
		// already covered by the typed and built-in dynamic fetchers.
		if gv.Group == "" || IsBuiltinGroup(gv.Group) {
			continue
		}

		// A group may serve multiple versions of the same resource. Restrict to
		// the server's preferred version so instances are not fetched twice.
		if pv, ok := preferred[gv.Group]; ok && pv != "" && pv != gv.Version {
			continue
		}

		for _, apiResource := range list.APIResources {
			if IsSubresource(apiResource.Name) || !verbsContain(apiResource.Verbs, listVerb) {
				continue
			}

			key := gv.Group + "/" + apiResource.Name
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}

			out = append(out, APIResource{
				Group:      gv.Group,
				Version:    gv.Version,
				Resource:   apiResource.Name,
				Kind:       resource.Kind(apiResource.Kind),
				Namespaced: apiResource.Namespaced,
				Typed:      false,
				Custom:     true,
			})
		}
	}

	sortAPIResources(out)
	return out, nil
}

// preferredGroupVersions maps each API group to its server-preferred version.
func preferredGroupVersions(groups []*metav1.APIGroup) map[string]string {
	preferred := make(map[string]string, len(groups))
	for _, group := range groups {
		if group == nil || group.PreferredVersion.Version == "" {
			continue
		}
		preferred[group.Name] = group.PreferredVersion.Version
	}
	return preferred
}

// verbsContain reports whether the verb set contains the given verb.
func verbsContain(verbs metav1.Verbs, verb string) bool {
	for _, v := range verbs {
		if v == verb {
			return true
		}
	}
	return false
}

// sortAPIResources orders descriptors deterministically by group, version and
// resource so discovery output is stable across runs.
func sortAPIResources(resources []APIResource) {
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Group != resources[j].Group {
			return resources[i].Group < resources[j].Group
		}
		if resources[i].Version != resources[j].Version {
			return resources[i].Version < resources[j].Version
		}
		return resources[i].Resource < resources[j].Resource
	})
}

// DiscoverResources returns the full set of resources klue can diagnose: the
// built-in catalog plus any custom resources discovered on the cluster. When no
// typed clientset is configured only the built-in catalog is returned.
func (c *Client) DiscoverResources() ([]APIResource, error) {
	catalog := BuiltinCatalog()
	if c.clientset == nil {
		return catalog, nil
	}

	customs, err := discoverCustomResources(c.clientset.Discovery())
	if err != nil {
		return catalog, err
	}

	return append(catalog, customs...), nil
}

// ResolveResource resolves a CLI token (a kind, plural resource name or alias)
// to a single resource descriptor. When apiVersion is non-empty it is used to
// disambiguate tokens served by multiple groups or versions. An ambiguous token
// without an apiVersion qualifier yields an error listing the candidates.
func ResolveResource(resources []APIResource, token, apiVersion string) (APIResource, error) {
	token = strings.ToLower(strings.TrimSpace(token))
	apiVersion = strings.TrimSpace(apiVersion)

	var matches []APIResource
	for _, entry := range resources {
		if !resourceMatchesToken(entry, token) {
			continue
		}
		if apiVersion != "" && entry.APIVersion() != apiVersion {
			continue
		}
		matches = append(matches, entry)
	}

	switch len(matches) {
	case 0:
		if apiVersion != "" {
			return APIResource{}, fmt.Errorf("no resource %q served by apiVersion %q", token, apiVersion)
		}
		return APIResource{}, fmt.Errorf("unknown resource %q", token)
	case 1:
		return matches[0], nil
	default:
		return APIResource{}, ambiguousResourceError(token, matches)
	}
}

// resourceMatchesToken reports whether the token names the given resource by its
// kind, plural resource name or one of its CLI aliases.
func resourceMatchesToken(entry APIResource, token string) bool {
	if token == "" {
		return false
	}
	if entry.CommandToken() == token || strings.ToLower(entry.Resource) == token {
		return true
	}
	for _, alias := range entry.Aliases {
		if strings.ToLower(alias) == token {
			return true
		}
	}
	return false
}

// ambiguousResourceError builds an error that lists the apiVersions a token
// resolves to so the user can re-run with --api-version.
func ambiguousResourceError(token string, matches []APIResource) error {
	versions := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		gv := match.APIVersion()
		if _, ok := seen[gv]; ok {
			continue
		}
		seen[gv] = struct{}{}
		versions = append(versions, gv)
	}
	sort.Strings(versions)

	return fmt.Errorf(
		"resource %q is ambiguous across apiVersions %s; specify --api-version",
		token, strings.Join(versions, ", "),
	)
}
