---
description: Generic diagnostic rules in klue that apply to every Kubernetes resource — warning events, failed conditions, stuck termination, missing references, and orphaned owners.
tags:
  - rules
  - reference
  - builtin
---

# Builtin rules

The rules in the `builtin` category are **generic**: they apply to every
Kubernetes resource kind, including custom resources. They reason about
cross-cutting patterns — warning events, status conditions, deletion timestamps,
owner references — that would otherwise have to be duplicated in every
resource-specific rule.

Generic rules intentionally carry **lower confidence** than typed rules so that
specific findings rank above them when both describe the same root cause.

---

## `builtin/warning-events`

**Severity:** warning | **Confidence:** 0.40 | **Applies to:** Any

Surfaces the most recent Kubernetes warning event recorded against a resource.

### When it fires

A `Warning`-type event has been recorded for the resource within the configured
event window (default: last 1 hour). This is a catch-all that fires for any
warning event not already explained by a more specific rule.

### Example finding

> **Warning event: BackOff**
> Kubernetes recorded a recent warning event for this resource that may explain
> its behaviour.

### Remediation

```bash
kubectl describe <kind> <name> -n <namespace>
```

!!! note "Low confidence"
    `builtin/warning-events` carries confidence 0.40. When a stronger typed
    rule fires for the same resource (for example `pod/crashloop`), the generic
    warning finding is suppressed to keep output focused on root causes.

---

## `builtin/failed-condition`

**Severity:** error | **Confidence:** 0.60 | **Applies to:** Any (unstructured objects only)

Detects failing status conditions on resources that klue does not have a
dedicated typed rule for. It inspects `status.conditions` for:

- Positive conditions (`Ready`, `Available`, `Healthy`, `Initialized`) whose
  status is `False`.
- Negative conditions (`Failed`, `Degraded`) whose status is `True`.

### When it fires

An unstructured (dynamically fetched) resource carries a condition in a failing
state. Typed objects — such as `Pod`, `Deployment`, etc. — are left to their
dedicated rules.

### Example finding

> **Condition Available=False**
> The resource reports condition "Available" in a failing state (False).

### Remediation

```bash
kubectl describe <kind> <name> -n <namespace>
```

---

## `builtin/missing-reference`

**Severity:** error | **Confidence:** 0.70 | **Applies to:** Any

Reports unresolved placeholder nodes introduced during resource graph
construction. When the graph builder encounters a reference to an object (for
example a `secretName` in an Ingress TLS entry) and that object does not exist
in the cluster, the missing object is represented as a placeholder node. This
rule fires on those placeholder nodes.

### When it fires

- A resource in the graph references another object by name (Secret, ConfigMap,
  Service, etc.).
- That referenced object does not exist in the cluster.
- No other resource is known to produce it (for example a cert-manager
  `Certificate` that would generate the Secret).

### Example finding

> **Missing referenced Secret "web-tls"**
> At least one resource references Secret "web-tls", but that object does not
> exist in the cluster.

### Remediation

```bash
# For a namespaced resource
kubectl get <kind> <name> -n <namespace>

# For a cluster-scoped resource
kubectl get <kind> <name>
```

---

## `builtin/orphaned-owner`

**Severity:** warning | **Confidence:** 0.60 | **Applies to:** Any

Detects resources whose `metadata.ownerReferences` point to an object that no
longer exists in the graph. An orphaned owner can leave a resource unmanaged —
for example a ReplicaSet whose Deployment was deleted without garbage collection,
or a custom resource whose owning controller object is gone.

### When it fires

A resource has one or more owner references and none of the referenced owners
are found in the resource graph.

### Example finding

> **Owner Deployment/web is missing**
> The resource is owned by Deployment "web" which no longer exists, so it may
> be unmanaged or left behind by a deletion.

### Remediation

```bash
kubectl get <kind> <name> -n <namespace> -o yaml
```

Check whether the resource should still exist and delete it if it is stale.

---

## `builtin/terminating-stuck`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** Any

Detects objects that have carried a `metadata.deletionTimestamp` for longer than
the grace period (default: 5 minutes) without being removed. This almost always
means a **finalizer** is not completing.

### When it fires

A resource has a `deletionTimestamp` and the elapsed time since that timestamp
exceeds the terminating grace period. Resources deleted within the grace period
do not trigger this rule.

### Example finding

> **Resource is stuck terminating**
> The resource has a deletion timestamp but has not been removed, usually
> because a finalizer is not completing.

### Remediation

```bash
# Inspect the pending finalizers
kubectl get <kind> <name> -n <namespace> -o jsonpath='{.metadata.finalizers}'
```

To force-remove the resource, patch the finalizers list to empty. Do this only
when you are certain the protected resource can be safely deleted:

```bash
kubectl patch <kind> <name> -n <namespace> \
  -p '{"metadata":{"finalizers":[]}}' --type=merge
```

---

## `builtin/log-signal`

**Severity:** warning | **Confidence:** 0.70–0.95 | **Applies to:** Pod

Surfaces failure patterns detected in container logs when pod status alone does
not already explain the failure. Log lines are matched against known signal
patterns — connection refused, DNS failures, missing config files, permission
denied, panics, and OOM kills — and ranked by confidence.

### When it fires

- The pod has container logs that contain a recognised failure pattern.
- Pod status does **not** already carry a strong signal (for example
  `CrashLoopBackOff`, `ImagePullBackOff`, `OOMKilled`, or `PodFailed`).

Common log signals and their confidence boosts:

| Signal | Example log text | Confidence boost |
|--------|------------------|-----------------|
| `connection-refused` | `dial tcp: connection refused` | +0.15 |
| `no-such-host` | `no such host` | +0.10 |
| `config-missing` | `no such file or directory` | +0.10 |
| `permission-denied` | `permission denied` | +0.10 |
| `panic` | `panic: runtime error` | +0.20 |
| `oom-killed` | `signal: killed` | +0.15 |

### Example finding

> **Container "app" logs show connection refused**
> Container "app" logs contain "dial tcp: connection refused", suggesting a
> downstream service is unreachable.

### Remediation

Suggestions depend on the detected signal:

=== "Connection refused / DNS"

    ```bash
    kubectl get svc,endpointslices -n <namespace>
    ```

=== "Missing config file"

    ```bash
    kubectl describe pod <name> -n <namespace>
    ```

=== "Permission denied"

    ```bash
    kubectl auth can-i --list
    ```
