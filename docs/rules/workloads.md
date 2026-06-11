---
description: Diagnostic rules for Kubernetes workloads — Pod, Deployment, StatefulSet, ReplicaSet, and DaemonSet.
tags:
  - rules
  - reference
  - workloads
  - pod
  - deployment
  - statefulset
  - replicaset
  - daemonset
---

# Workload rules

The workload rules cover the most common Kubernetes workload kinds: **Pod**,
**Deployment**, **StatefulSet**, **ReplicaSet**, and **DaemonSet**. These rules
are the first line of diagnosis for application availability problems.

---

## Pod rules

### `pod/crashloop`

**Severity:** critical | **Confidence:** 0.95 | **Applies to:** Pod

Detects containers in `CrashLoopBackOff`. This is one of the most reliable
signals in Kubernetes: the container started, failed, and the backoff timer is
preventing another immediate restart.

#### When it fires

A container's waiting state reason is `CrashLoopBackOff`. klue also attaches
container log evidence from the previous run (when available) to help identify
the crash reason.

#### Example finding

> **Container "app" is in CrashLoopBackOff**
> The container is crashing repeatedly. The last exit code and restart count
> indicate a persistent failure.

#### Remediation

```bash
# Inspect logs from the previous (crashed) container run
kubectl logs <pod> -n <namespace> -c <container> --previous
```

---

### `pod/image-pull`

**Severity:** error | **Confidence:** 0.60–0.90 | **Applies to:** Pod

Detects containers that cannot pull their image (`ImagePullBackOff` or
`ErrImagePull`). Confidence is adjusted based on corroborating warning events:

| Reason | Confidence |
|--------|-----------|
| `ErrImagePull` without corroborating event | 0.60 |
| `ImagePullBackOff` with matching warning event | 0.90 |

#### When it fires

A container's waiting state reason is `ImagePullBackOff` or `ErrImagePull`.

Common root causes detected from events:

| Event signal | Explanation |
|--------------|-------------|
| Network error | Registry is unreachable |
| TLS / x509 | Certificate trust issue with a private registry |
| Unauthorized | Missing or invalid `imagePullSecret` |
| Not found | Image tag does not exist in the registry |

#### Example finding

> **Container "app" cannot pull image "myregistry/api:v2.1"**
> The container cannot pull the image. Check the image reference and registry
> access.

#### Remediation

=== "Verify registry access"

    ```bash
    kubectl get events -n <namespace> --field-selector involvedObject.name=<pod>
    ```

=== "Inspect pull secrets"

    ```bash
    kubectl get pod <pod> -n <namespace> -o jsonpath='{.spec.imagePullSecrets}'
    ```

=== "Check image reference"

    ```bash
    kubectl describe pod <pod> -n <namespace>
    ```

---

### `pod/config-missing`

**Severity:** error | **Confidence:** 0.85 | **Applies to:** Pod

Detects pods that reference a ConfigMap or Secret that does not exist in the
cluster. The pod will be stuck in a `CreateContainerConfigError` state.

#### When it fires

A container's waiting reason is `CreateContainerConfigError` and the referenced
ConfigMap or Secret is not present in the namespace.

#### Example finding

> **ConfigMap "app-config" referenced by the pod does not exist**
> The pod references ConfigMap "app-config" which is not present in namespace
> "default".

#### Remediation

```bash
# Verify the ConfigMap / Secret exists
kubectl get configmap <name> -n <namespace>
kubectl get secret <name> -n <namespace>
```

---

### `pod/mount-failure`

**Severity:** error | **Confidence:** 0.85 | **Applies to:** Pod

Detects volume mount and attachment failures surfaced through Kubernetes warning
events. The rule parses structured event signals to pinpoint the cause.

#### When it fires

Kubernetes warning events for the pod contain mount or attachment failure
patterns (`FailedMount`, `FailedAttachVolume`).

Common causes detected:

| Cause | Explanation |
|-------|-------------|
| `secret-missing` | A Secret volume source does not exist |
| `configmap-missing` | A ConfigMap volume source does not exist |
| `pvc-not-bound` | The referenced PVC is still Pending |
| `csi-driver-failure` | CSI driver error during attach/mount |
| `node-constraint` | Volume and pod topology mismatch |

#### Example finding

> **Pod volume mount failed: Secret "tls-cert" not found**
> Kubernetes warning events indicate volume mount/attachment failures for this
> pod. Cause: the Secret volume source does not exist.

#### Remediation

```bash
kubectl describe pod <name> -n <namespace>
kubectl get pvc -n <namespace>
```

---

### `pod/pending`

**Severity:** error | **Confidence:** 0.80 | **Applies to:** Pod

Detects pods that the scheduler cannot place on any node. The pod remains
indefinitely in the `Pending` phase.

#### When it fires

The pod's phase is `Pending` and the scheduler has emitted a `FailedScheduling`
warning event with an `Unschedulable` reason.

Common reasons:

- Insufficient CPU or memory on all available nodes
- `nodeSelector` or node affinity rules match no nodes
- Taints on all nodes that the pod does not tolerate
- `PodDisruptionBudget` blocking the placement

#### Example finding

> **Pod cannot be scheduled**
> The scheduler could not find a node that satisfies the pod's resource
> requirements and scheduling constraints.

#### Remediation

```bash
kubectl describe pod <name> -n <namespace>
kubectl get nodes -o wide
```

---

### `pod/probe-failure`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** Pod

Detects pods whose liveness or readiness probes are failing. Failing probes keep
pods out of service endpoints and may trigger unnecessary restarts.

#### When it fires

Kubernetes warning events for the pod contain probe failure patterns
(`Liveness probe failed`, `Readiness probe failed`).

#### Example finding

> **Pod is failing health probes**
> Liveness or readiness probe failures are being recorded. The pod may be
> removed from Service endpoints or restarted.

#### Remediation

```bash
kubectl describe pod <name> -n <namespace>
kubectl logs <name> -n <namespace>
```

---

## Deployment rules

### `deployment/rollout-stuck`

**Severity:** error | **Confidence:** 0.85 | **Applies to:** Deployment

Detects deployments whose rollout has exceeded the `progressDeadlineSeconds`
limit. Kubernetes sets the `Progressing` condition to `False` with reason
`ProgressDeadlineExceeded` when this happens.

#### When it fires

The deployment's `Progressing` status condition has reason
`ProgressDeadlineExceeded`.

#### Example finding

> **Deployment rollout is stuck**
> The deployment did not progress within its deadline. New pods may be failing
> to become ready.

#### Remediation

```bash
kubectl rollout status deployment/<name> -n <namespace>
kubectl describe deployment <name> -n <namespace>
```

---

### `deployment/unavailable`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** Deployment

Detects deployments with fewer available replicas than the desired count.

#### When it fires

`status.availableReplicas` is less than `spec.replicas` (or less than 1 when
`spec.replicas` is unset).

#### Example finding

> **Deployment has 1/3 replicas available**
> Some replicas are not available. Inspect the owned pods to find the underlying
> cause.

#### Remediation

```bash
kubectl get pods -n <namespace> -l app=<name>
kubectl describe deployment <name> -n <namespace>
```

---

## StatefulSet rules

### `statefulset/unavailable`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** StatefulSet

Detects StatefulSets with fewer ready replicas than the desired count. Because
StatefulSets roll out sequentially, a single stuck pod blocks every subsequent
pod.

#### When it fires

`status.readyReplicas` is less than `spec.replicas`.

#### Example finding

> **StatefulSet has 2/3 replicas ready**
> Some replicas are not ready. StatefulSet pods roll out sequentially, so a
> single stuck pod blocks the rest.

#### Remediation

```bash
kubectl get pods -n <namespace> -l app=<name> --sort-by=.metadata.name
kubectl describe statefulset <name> -n <namespace>
```

---

### `statefulset/rollout-stuck`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** StatefulSet

Detects StatefulSet rollouts where a newer revision has been started but not all
pods have been updated. The next ordinal pod is likely failing to become ready.

#### When it fires

`status.updatedReplicas` is less than `spec.replicas` and the current update
revision differs from the update revision.

#### Example finding

> **StatefulSet rollout has not completed**
> A newer revision is only partially rolled out. The next ordinal pod is likely
> failing to become ready.

#### Remediation

```bash
kubectl rollout status statefulset/<name> -n <namespace>
kubectl get pods -n <namespace> -l app=<name> --sort-by=.metadata.name
```

---

## ReplicaSet rules

### `replicaset/unavailable`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** ReplicaSet

Detects ReplicaSets that cannot reach their desired replica count.

#### When it fires

`status.readyReplicas` is less than `spec.replicas`.

#### Example finding

> **ReplicaSet has 0/3 replicas ready**
> The ReplicaSet cannot reach its desired replica count. Inspect the owned pods
> to find the underlying cause.

#### Remediation

```bash
kubectl describe replicaset <name> -n <namespace>
kubectl get pods -n <namespace> --selector=<selector>
```

---

### `replicaset/replica-failure`

**Severity:** error | **Confidence:** 0.80 | **Applies to:** ReplicaSet

Detects ReplicaSets that have failed to create pods — for example because a
resource quota was exceeded or an admission webhook rejected the pod template.

#### When it fires

The ReplicaSet has a `ReplicaFailure` condition set to `True`.

#### Example finding

> **ReplicaSet cannot create pods**
> The ReplicaSet failed to create pods, often due to resource quotas, limit
> ranges, or admission webhooks rejecting the pod template.

#### Remediation

```bash
kubectl describe replicaset <name> -n <namespace>
kubectl get resourcequota -n <namespace>
```

---

## DaemonSet rules

### `daemonset/unavailable`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** DaemonSet

Detects DaemonSets that are not running healthily on every eligible node.

#### When it fires

`status.numberUnavailable > 0` or `status.numberReady < status.desiredNumberScheduled`.

#### Example finding

> **DaemonSet has 4/6 pods ready**
> The DaemonSet is not running healthily on every eligible node. Affected nodes
> may lack resources or be failing to pull the image.

#### Remediation

```bash
kubectl get pods -n <namespace> -l app=<name> -o wide
kubectl describe daemonset <name> -n <namespace>
```

---

### `daemonset/misscheduled`

**Severity:** warning | **Confidence:** 0.60 | **Applies to:** DaemonSet

Detects DaemonSet pods running on nodes that should not run them. This can
happen after a node selector or taint change when the old pods are not yet
evicted.

#### When it fires

`status.numberMisscheduled > 0`.

#### Example finding

> **DaemonSet has 2 misscheduled pods**
> Some DaemonSet pods are running on nodes that no longer match the node
> selector or tolerate the node's taints.

#### Remediation

```bash
kubectl describe daemonset <name> -n <namespace>
kubectl get pods -n <namespace> -l app=<name> -o wide
```
