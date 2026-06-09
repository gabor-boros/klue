package kube

import (
	"context"
	"fmt"
	"sort"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	certificatesv1 "k8s.io/api/certificates/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Fetch retrieves all resources relevant to a diagnosis for the given
// namespace. Cluster-scoped resources (Nodes, PersistentVolumes, StorageClasses)
// are fetched in full. The first error encountered aborts the fetch.
func (c *Client) Fetch(ctx context.Context, namespace string) (*ResourceSnapshot, error) {
	return c.fetch(ctx, namespace, nil, false)
}

// FetchWithResources fetches the diagnosis snapshot while reusing a pre-fetched
// API resource catalog (for example the output of DiscoverResources). This
// avoids a second custom-resource discovery call during dynamic fetch.
func (c *Client) FetchWithResources(ctx context.Context, namespace string, resources []APIResource) (*ResourceSnapshot, error) {
	return c.fetch(ctx, namespace, resources, true)
}

func (c *Client) fetch(ctx context.Context, namespace string, resources []APIResource, providedResources bool) (*ResourceSnapshot, error) {
	snapshot := &ResourceSnapshot{Namespace: namespace}

	if err := c.fetchCoreTyped(ctx, namespace, snapshot); err != nil {
		return nil, err
	}

	if err := c.fetchExtendedTyped(ctx, namespace, snapshot); err != nil {
		return nil, err
	}

	if err := c.fetchDynamic(ctx, namespace, resources, providedResources, snapshot); err != nil {
		return nil, err
	}

	return snapshot, nil
}

type listJob struct {
	run func(context.Context) error
}

// requiredListJob wraps a list operation whose failure aborts the whole fetch.
// The label is used to produce a consistent "list <label>" error context.
func requiredListJob(label string, run func(context.Context) error) listJob {
	return listJob{
		run: func(ctx context.Context) error {
			if err := run(ctx); err != nil {
				return fmt.Errorf("list %s: %w", label, err)
			}
			return nil
		},
	}
}

// optionalListJob wraps a list operation whose ignorable failures (for example
// forbidden errors or resources the API server does not serve) are tolerated so
// an unrelated gap does not abort the diagnosis.
func optionalListJob(label string, run func(context.Context) error) listJob {
	return listJob{
		run: func(ctx context.Context) error {
			if err := run(ctx); err != nil && !isIgnorableListError(err) {
				return fmt.Errorf("list %s: %w", label, err)
			}
			return nil
		},
	}
}

func (c *Client) runListJobs(ctx context.Context, jobs []listJob) error {
	return c.runListJobsWithConcurrency(ctx, jobs, c.fetchConcurrency)
}

func (c *Client) runListJobsWithConcurrency(ctx context.Context, jobs []listJob, concurrency int) error {
	if len(jobs) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if concurrency <= 0 {
		concurrency = defaultFetchConcurrency
	}

	sem := make(chan struct{}, concurrency)
	var (
		wg       sync.WaitGroup
		once     sync.Once
		firstErr error
	)

	for i := range jobs {
		job := jobs[i]
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			if err := job.run(ctx); err != nil {
				once.Do(func() {
					firstErr = err
					cancel()
				})
			}
		}()
	}

	wg.Wait()
	return firstErr
}

func dynamicFetchConcurrency(fetchConcurrency int) int {
	if fetchConcurrency <= 0 {
		return 2
	}
	if fetchConcurrency > 4 {
		return 4
	}
	if fetchConcurrency > 1 {
		return fetchConcurrency / 2
	}

	return 1
}

func (c *Client) fetchCoreTyped(ctx context.Context, namespace string, snapshot *ResourceSnapshot) error {
	core := c.clientset.CoreV1()
	apps := c.clientset.AppsV1()
	batch := c.clientset.BatchV1()
	discovery := c.clientset.DiscoveryV1()
	networking := c.clientset.NetworkingV1()
	storage := c.clientset.StorageV1()

	var (
		pods           []corev1.Pod
		services       []corev1.Service
		endpointSlices []discoveryv1.EndpointSlice
		configMaps     []corev1.ConfigMap
		secrets        []corev1.Secret
		pvcs           []corev1.PersistentVolumeClaim
		events         []corev1.Event
		nodes          []corev1.Node
		pvs            []corev1.PersistentVolume
		deployments    []appsv1.Deployment
		replicaSets    []appsv1.ReplicaSet
		statefulSets   []appsv1.StatefulSet
		jobs           []batchv1.Job
		cronJobs       []batchv1.CronJob
		ingresses      []networkingv1.Ingress
		storageClasses []storagev1.StorageClass
	)

	err := c.runListJobs(ctx, []listJob{
		requiredListJob("pods", func(ctx context.Context) error {
			list, err := core.Pods(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				pods = list.Items
			}
			return err
		}),
		requiredListJob("services", func(ctx context.Context) error {
			list, err := core.Services(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				services = list.Items
			}
			return err
		}),
		requiredListJob("endpointslices", func(ctx context.Context) error {
			list, err := discovery.EndpointSlices(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				endpointSlices = list.Items
			}
			return err
		}),
		requiredListJob("configmaps", func(ctx context.Context) error {
			list, err := core.ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				configMaps = list.Items
			}
			return err
		}),
		requiredListJob("secrets", func(ctx context.Context) error {
			list, err := core.Secrets(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				secrets = list.Items
			}
			return err
		}),
		requiredListJob("persistentvolumeclaims", func(ctx context.Context) error {
			list, err := core.PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				pvcs = list.Items
			}
			return err
		}),
		requiredListJob("events", func(ctx context.Context) error {
			list, err := core.Events(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				events = list.Items
			}
			return err
		}),
		requiredListJob("nodes", func(ctx context.Context) error {
			list, err := core.Nodes().List(ctx, metav1.ListOptions{})
			if err == nil {
				nodes = list.Items
			}
			return err
		}),
		requiredListJob("persistentvolumes", func(ctx context.Context) error {
			list, err := core.PersistentVolumes().List(ctx, metav1.ListOptions{})
			if err == nil {
				pvs = list.Items
			}
			return err
		}),
		requiredListJob("deployments", func(ctx context.Context) error {
			list, err := apps.Deployments(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				deployments = list.Items
			}
			return err
		}),
		requiredListJob("replicasets", func(ctx context.Context) error {
			list, err := apps.ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				replicaSets = list.Items
			}
			return err
		}),
		requiredListJob("statefulsets", func(ctx context.Context) error {
			list, err := apps.StatefulSets(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				statefulSets = list.Items
			}
			return err
		}),
		requiredListJob("jobs", func(ctx context.Context) error {
			list, err := batch.Jobs(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				jobs = list.Items
			}
			return err
		}),
		requiredListJob("cronjobs", func(ctx context.Context) error {
			list, err := batch.CronJobs(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				cronJobs = list.Items
			}
			return err
		}),
		requiredListJob("ingresses", func(ctx context.Context) error {
			list, err := networking.Ingresses(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				ingresses = list.Items
			}
			return err
		}),
		requiredListJob("storageclasses", func(ctx context.Context) error {
			list, err := storage.StorageClasses().List(ctx, metav1.ListOptions{})
			if err == nil {
				storageClasses = list.Items
			}
			return err
		}),
	})
	if err != nil {
		return err
	}

	snapshot.Pods = pods
	snapshot.Services = services
	snapshot.EndpointSlices = endpointSlices
	snapshot.ConfigMaps = configMaps
	snapshot.Secrets = secrets
	snapshot.PersistentVolumeClaims = pvcs
	snapshot.Events = events
	snapshot.Nodes = nodes
	snapshot.PersistentVolumes = pvs
	snapshot.Deployments = deployments
	snapshot.ReplicaSets = replicaSets
	snapshot.StatefulSets = statefulSets
	snapshot.Jobs = jobs
	snapshot.CronJobs = cronJobs
	snapshot.Ingresses = ingresses
	snapshot.StorageClasses = storageClasses

	return nil
}

// fetchExtendedTyped lists the additional built-in resources that have typed
// diagnostic rules. Unlike the core resources fetched above, list failures for
// these (for example RBAC forbidden errors) are tolerated so an unrelated
// permission gap does not abort the whole diagnosis.
func (c *Client) fetchExtendedTyped(ctx context.Context, namespace string, snapshot *ResourceSnapshot) error {
	apps := c.clientset.AppsV1()
	networking := c.clientset.NetworkingV1()
	autoscaling := c.clientset.AutoscalingV2()
	policy := c.clientset.PolicyV1()
	rbac := c.clientset.RbacV1()
	certificates := c.clientset.CertificatesV1()
	coordination := c.clientset.CoordinationV1()

	var (
		daemonSets       []appsv1.DaemonSet
		networkPolicies  []networkingv1.NetworkPolicy
		hpas             []autoscalingv2.HorizontalPodAutoscaler
		pdbs             []policyv1.PodDisruptionBudget
		roles            []rbacv1.Role
		roleBindings     []rbacv1.RoleBinding
		clusterRoles     []rbacv1.ClusterRole
		clusterRoleBinds []rbacv1.ClusterRoleBinding
		csrs             []certificatesv1.CertificateSigningRequest
		leases           []coordinationv1.Lease
	)

	if err := c.runListJobs(ctx, []listJob{
		optionalListJob("daemonsets", func(ctx context.Context) error {
			list, err := apps.DaemonSets(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				daemonSets = list.Items
			}
			return err
		}),
		optionalListJob("networkpolicies", func(ctx context.Context) error {
			list, err := networking.NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				networkPolicies = list.Items
			}
			return err
		}),
		optionalListJob("horizontalpodautoscalers", func(ctx context.Context) error {
			list, err := autoscaling.HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				hpas = list.Items
			}
			return err
		}),
		optionalListJob("poddisruptionbudgets", func(ctx context.Context) error {
			list, err := policy.PodDisruptionBudgets(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				pdbs = list.Items
			}
			return err
		}),
		optionalListJob("roles", func(ctx context.Context) error {
			list, err := rbac.Roles(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				roles = list.Items
			}
			return err
		}),
		optionalListJob("rolebindings", func(ctx context.Context) error {
			list, err := rbac.RoleBindings(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				roleBindings = list.Items
			}
			return err
		}),
		optionalListJob("clusterroles", func(ctx context.Context) error {
			list, err := rbac.ClusterRoles().List(ctx, metav1.ListOptions{})
			if err == nil {
				clusterRoles = list.Items
			}
			return err
		}),
		optionalListJob("clusterrolebindings", func(ctx context.Context) error {
			list, err := rbac.ClusterRoleBindings().List(ctx, metav1.ListOptions{})
			if err == nil {
				clusterRoleBinds = list.Items
			}
			return err
		}),
		optionalListJob("certificatesigningrequests", func(ctx context.Context) error {
			list, err := certificates.CertificateSigningRequests().List(ctx, metav1.ListOptions{})
			if err == nil {
				csrs = list.Items
			}
			return err
		}),
		optionalListJob("leases", func(ctx context.Context) error {
			list, err := coordination.Leases(namespace).List(ctx, metav1.ListOptions{})
			if err == nil {
				leases = list.Items
			}
			return err
		}),
	}); err != nil {
		return err
	}

	snapshot.DaemonSets = daemonSets
	snapshot.NetworkPolicies = networkPolicies
	snapshot.HorizontalPodAutoscalers = hpas
	snapshot.PodDisruptionBudgets = pdbs
	snapshot.Roles = roles
	snapshot.RoleBindings = roleBindings
	snapshot.ClusterRoles = clusterRoles
	snapshot.ClusterRoleBindings = clusterRoleBinds
	snapshot.CertificateSigningRequests = csrs
	snapshot.Leases = leases

	return nil
}

// fetchDynamic lists every resource that is not retrieved through the typed
// clientset, storing each object as unstructured data. This covers both
// built-in resources without a typed fetcher and custom resources discovered on
// the cluster. Per-resource list failures (no permission, resource not served
// by the API server) are tolerated so the diagnosis can proceed with whatever
// could be read.
func (c *Client) fetchDynamic(ctx context.Context, namespace string, resources []APIResource, providedResources bool, snapshot *ResourceSnapshot) error {
	if c.dynamic == nil {
		return nil
	}

	entries := make([]APIResource, 0)
	for _, entry := range BuiltinCatalog() {
		if entry.Typed {
			continue
		}
		entries = append(entries, entry)
	}

	customEntries := c.customResourceEntries(resources, providedResources)
	entries = append(entries, customEntries...)

	dynamicObjects, err := c.listDynamicEntries(ctx, namespace, entries)
	if err != nil {
		return err
	}
	snapshot.Dynamic = dynamicObjects

	return nil
}

func (c *Client) customResourceEntries(resources []APIResource, providedResources bool) []APIResource {
	var customs []APIResource
	if providedResources {
		customs = customEntries(resources)
		sortAPIResources(customs)
		return customs
	}

	discovered, err := discoverCustomResources(c.clientset.Discovery())
	if err != nil {
		// Discovery problems must not abort a diagnosis; built-in resources
		// remain useful on their own.
		return nil
	}

	return discovered
}

func customEntries(resources []APIResource) []APIResource {
	customs := make([]APIResource, 0)
	seen := make(map[string]struct{})
	for i := range resources {
		entry := resources[i]
		if !entry.Custom || entry.Typed {
			continue
		}
		key := entry.Group + "/" + entry.Version + "/" + entry.Resource
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		customs = append(customs, entry)
	}

	return customs
}

func (c *Client) listDynamicEntries(ctx context.Context, namespace string, entries []APIResource) ([]DynamicObject, error) {
	results := make([][]DynamicObject, len(entries))
	jobs := make([]listJob, 0, len(entries))

	for i := range entries {
		idx := i
		entry := entries[idx]
		jobs = append(jobs, listJob{
			run: func(ctx context.Context) error {
				list, err := c.listDynamicObjects(ctx, namespace, entry)
				if err != nil {
					return err
				}
				results[idx] = list
				return nil
			},
		})
	}

	if err := c.runListJobsWithConcurrency(ctx, jobs, dynamicFetchConcurrency(c.fetchConcurrency)); err != nil {
		return nil, err
	}

	objects := make([]DynamicObject, 0)
	for i := range results {
		objects = append(objects, results[i]...)
	}

	sort.SliceStable(objects, func(i, j int) bool {
		ri, rj := objects[i].Resource, objects[j].Resource
		if ri.Group != rj.Group {
			return ri.Group < rj.Group
		}
		if ri.Version != rj.Version {
			return ri.Version < rj.Version
		}
		if ri.Resource != rj.Resource {
			return ri.Resource < rj.Resource
		}

		oi, oj := objects[i].Object, objects[j].Object
		if oi.GetNamespace() != oj.GetNamespace() {
			return oi.GetNamespace() < oj.GetNamespace()
		}
		if oi.GetName() != oj.GetName() {
			return oi.GetName() < oj.GetName()
		}

		return string(oi.GetUID()) < string(oj.GetUID())
	})

	return objects, nil
}

// listDynamicObjects lists a single resource via the dynamic client. Ignorable
// list errors (forbidden, not found, no matching kind) are tolerated.
func (c *Client) listDynamicObjects(ctx context.Context, namespace string, entry APIResource) ([]DynamicObject, error) {
	gvr := entry.GroupVersionResource()

	var (
		list *unstructured.UnstructuredList
		err  error
	)
	if entry.Namespaced {
		list, err = c.dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = c.dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		if isIgnorableListError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list %s: %w", entry.Resource, err)
	}

	out := make([]DynamicObject, 0, len(list.Items))
	for i := range list.Items {
		item := list.Items[i]
		out = append(out, DynamicObject{Resource: entry, Object: &item})
	}

	return out, nil
}

// isIgnorableListError reports whether a list error should be tolerated rather
// than aborting the diagnosis. Permission errors and resources the API server
// does not serve are common in restricted clusters and must not be fatal.
func isIgnorableListError(err error) bool {
	return apierrors.IsForbidden(err) ||
		apierrors.IsNotFound(err) ||
		apierrors.IsMethodNotSupported(err) ||
		meta.IsNoMatchError(err)
}
