---
description: Diagnostic rules for Kubernetes reliability and autoscaling resources — HorizontalPodAutoscaler, PodDisruptionBudget, and Lease.
tags:
  - rules
  - reference
  - reliability
  - hpa
  - pdb
  - lease
---

# Reliability rules

The reliability rules diagnose problems with **HorizontalPodAutoscaler (HPA)**,
**PodDisruptionBudget (PDB)**, and **Lease** resources. Issues with these
resources can silently degrade availability or block operational tasks such as
node drains and rolling updates.

---

## HorizontalPodAutoscaler rules

### `hpa/scaling-disabled`

**Severity:** error | **Confidence:** 0.80 | **Applies to:** HorizontalPodAutoscaler

Detects HPAs that cannot compute or apply scaling decisions. This usually means
the metrics server is unavailable, metrics are stale, or the scale target is
missing.

#### When it fires

The HPA's `status.conditions` contain one of the following conditions set to
`False`:

| Condition | Meaning |
|-----------|---------|
| `AbleToScale` | The autoscaler cannot apply a scale operation |
| `ScalingActive` | The autoscaler cannot retrieve metrics to make a decision |

#### Example finding

> **HPA cannot scale (AbleToScale=False)**
> The autoscaler cannot retrieve metrics or apply a scale decision, so the
> workload will not scale with load.

#### Remediation

```bash
kubectl describe hpa <name> -n <namespace>
kubectl get apiservices | grep metrics
```

---

### `hpa/missing-scale-target`

**Severity:** error | **Confidence:** 0.80 | **Applies to:** HorizontalPodAutoscaler

Detects HPAs whose `spec.scaleTargetRef` points to a workload that does not
exist in the cluster. The autoscaler has nothing to scale.

#### When it fires

The resource referenced in `spec.scaleTargetRef` (typically a Deployment,
StatefulSet, or ReplicaSet) is not present in the graph.

#### Example finding

> **HPA scale target Deployment/api does not exist**
> The autoscaler points at a workload that does not exist, so it has nothing to
> scale.

#### Remediation

```bash
kubectl get deployment <name> -n <namespace>
kubectl describe hpa <name> -n <namespace>
```

---

## PodDisruptionBudget rules

### `pdb/disruptions-blocked`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** PodDisruptionBudget

Detects PodDisruptionBudgets that currently allow zero voluntary disruptions.
When no disruptions are allowed, node drains and rolling updates of the covered
pods will block until more replicas become healthy.

#### When it fires

The PDB has been observed by the controller (`status.observedGeneration > 0` or
`status.expectedPods > 0`) and `status.disruptionsAllowed` is `0`.

#### Example finding

> **PodDisruptionBudget allows no disruptions**
> No voluntary disruptions are currently allowed, so node drains and rolling
> updates of the covered pods will block until more replicas become healthy.

#### Remediation

```bash
kubectl describe pdb <name> -n <namespace>
kubectl get pods -n <namespace> --show-labels
```

Investigate why covered pods are unhealthy; resolving the underlying pod issue
will increase `currentHealthy` and restore available disruptions.

---

### `pdb/no-matching-pods`

**Severity:** warning | **Confidence:** 0.65 | **Applies to:** PodDisruptionBudget

Detects PodDisruptionBudgets whose selector matches no pods. A budget that
protects no pods provides no availability guarantee.

#### When it fires

The PDB has a non-nil `spec.selector` and no pod in the namespace carries all
of the required labels (no `Protects` edge in the resource graph and
`status.expectedPods` is `0`).

#### Example finding

> **PodDisruptionBudget selects no pods**
> The budget's selector matches no pods, so it provides no protection. The
> selector may be misconfigured.

#### Remediation

```bash
kubectl get pods -n <namespace> --show-labels
kubectl describe pdb <name> -n <namespace>
```

---

## Lease rules

### `lease/stale`

**Severity:** warning | **Confidence:** 0.60 | **Applies to:** Lease

Detects Leases whose holder has not renewed within several lease durations.
Leader-election-based components use Leases to signal liveness; a stale Lease
typically means the component holding the Lease is down or network-partitioned.

#### When it fires

The time since `spec.renewTime` exceeds `spec.leaseDurationSeconds × 4`
(default: `15s × 4 = 60s`). The multiplier can be configured with
`--lease-stale-multiplier`.

!!! note "Reference clock required"
    This rule requires a reference clock (`--now`) to avoid non-deterministic
    output. In standard `klue why` invocations the current time is always
    provided.

#### Example finding

> **Lease has not been renewed recently**
> The lease holder has not renewed within several lease durations. The
> leader-electing component may be down or partitioned.

#### Remediation

```bash
kubectl describe lease <name> -n <namespace>
```

Identify the component that holds the lease (from `spec.holderIdentity`) and
investigate its pods:

```bash
kubectl get pods -n <namespace> | grep <holder>
```
