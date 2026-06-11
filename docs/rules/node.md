---
description: Diagnostic rules for Kubernetes nodes — readiness, resource pressure, network availability, and schedulability.
tags:
  - rules
  - reference
  - node
---

# Node rules

The node rules inspect the health of individual Kubernetes **Nodes**. Node
problems affect every workload scheduled to them, so these rules rank among the
most impactful findings in a diagnosis.

---

## `node/not-ready`

**Severity:** error (Unknown) / critical (False) | **Confidence:** 0.90 | **Applies to:** Node

Detects nodes whose `Ready` condition is not `True`. A not-ready node cannot
reliably run workloads, and its pods may be evicted or fail to start.

#### When it fires

The node's `Ready` condition status is `False` or `Unknown`.

| Condition status | Finding severity |
|-----------------|-----------------|
| `Unknown` | error |
| `False` | critical |

#### Example finding

> **Node is not ready**
> The kubelet reports the node as not ready, so workloads may fail to schedule
> or run reliably.

#### Remediation

```bash
kubectl describe node <name>
kubectl get events --field-selector involvedObject.name=<name>
```

---

## `node/pressure`

**Severity:** error | **Confidence:** 0.85 | **Applies to:** Node

Detects nodes reporting active resource pressure conditions. A node under
pressure may evict pods or reject new scheduling requests.

#### When it fires

One or more of the following conditions are `True`:

| Condition | Meaning |
|-----------|---------|
| `MemoryPressure` | Available memory is low |
| `DiskPressure` | Available disk space or inodes are low |
| `PIDPressure` | The number of processes is approaching the limit |

#### Example finding

> **Node reports pressure conditions: MemoryPressure, DiskPressure**
> The node is under resource pressure and may evict pods or reject new
> scheduling requests.

#### Remediation

```bash
kubectl describe node <name>
kubectl top node <name>
```

---

## `node/network-unavailable`

**Severity:** error | **Confidence:** 0.80 | **Applies to:** Node

Detects nodes with the `NetworkUnavailable` condition set to `True`, indicating
that pod networking on the node is not functional.

#### When it fires

The node's `NetworkUnavailable` condition status is `True`. This condition is
typically set by the CNI plugin when it cannot configure networking for the node.

#### Example finding

> **Node network is unavailable**
> Pod networking on this node is unavailable, which can break service
> connectivity.

#### Remediation

Investigate the CNI daemon pods on the affected node:

```bash
kubectl describe node <name>
kubectl get pods -n kube-system -o wide | grep <node-name>
```

---

## `node/unschedulable`

**Severity:** warning | **Confidence:** 0.95 | **Applies to:** Node

Detects nodes that have been explicitly cordoned (`spec.unschedulable: true`).
New pods will not be scheduled to a cordoned node. This is often intentional
during maintenance but warrants attention when unexpected.

#### When it fires

`spec.unschedulable` is `true`.

#### Example finding

> **Node is marked unschedulable**
> New pods will not be scheduled to this node while it is cordoned.

#### Remediation

If the node should accept new workloads:

```bash
kubectl uncordon <name>
```
