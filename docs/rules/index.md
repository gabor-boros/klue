---
description: Overview of all diagnostic rules built into klue — IDs, severities, confidence levels, and the resource kinds each rule targets.
tags:
  - rules
  - reference
---

# Diagnostic rules

klue ships with a set of built-in diagnostic rules that run against the resource
graph whenever you execute `klue why`. Each rule produces zero or more
**findings** — structured observations with a severity, a confidence score, an
explanation, and one or more kubectl remediation suggestions.

## Rule anatomy

Every rule has:

| Field | Description |
|-------|-------------|
| **ID** | Unique dotted identifier (for example `pod/crashloop`) used with `--rule` / `--disable-rule` flags |
| **Severity** | `critical`, `error`, `warning`, or `info` — how serious the condition is |
| **Confidence** | `0.0`–`1.0` — how certain klue is that the finding represents a real problem |
| **Applies to** | The Kubernetes resource kind(s) the rule evaluates |

## Severity levels

| Level | Meaning |
|-------|---------|
| `critical` | The resource is broken and likely affecting availability right now |
| `error` | A definite problem that prevents the resource from working correctly |
| `warning` | A degraded or risky state that warrants attention |
| `info` | A noteworthy configuration state that is not necessarily a problem |

## All rules

The table below lists every built-in rule. Click the **ID** link to jump to the
detailed description.

| ID | Severity | Confidence | Applies to |
|----|----------|------------|------------|
| [builtin/failed-condition](builtin.md#builtinfailed-condition) | error | 0.60 | Any |
| [builtin/log-signal](builtin.md#builtinlog-signal) | warning | 0.70–0.95 | Pod |
| [builtin/missing-reference](builtin.md#builtinmissing-reference) | error | 0.70 | Any |
| [builtin/orphaned-owner](builtin.md#builtinorphaned-owner) | warning | 0.60 | Any |
| [builtin/terminating-stuck](builtin.md#builtinterminating-stuck) | warning | 0.70 | Any |
| [builtin/warning-events](builtin.md#builtinwarning-events) | warning | 0.40 | Any |
| [pod/crashloop](workloads.md#podcrashloop) | critical | 0.95 | Pod |
| [pod/image-pull](workloads.md#podimage-pull) | error | 0.60–0.90 | Pod |
| [pod/config-missing](workloads.md#podconfig-missing) | error | 0.85 | Pod |
| [pod/mount-failure](workloads.md#podmount-failure) | error | 0.85 | Pod |
| [pod/pending](workloads.md#podpending) | error | 0.80 | Pod |
| [pod/probe-failure](workloads.md#podprobe-failure) | warning | 0.70 | Pod |
| [deployment/rollout-stuck](workloads.md#deploymentrollout-stuck) | error | 0.85 | Deployment |
| [deployment/unavailable](workloads.md#deploymentunavailable) | warning | 0.70 | Deployment |
| [statefulset/unavailable](workloads.md#statefulsetunavailable) | warning | 0.70 | StatefulSet |
| [statefulset/rollout-stuck](workloads.md#statefulsetrollout-stuck) | warning | 0.70 | StatefulSet |
| [replicaset/unavailable](workloads.md#replicasetunavailable) | warning | 0.70 | ReplicaSet |
| [replicaset/replica-failure](workloads.md#replicasetreplica-failure) | error | 0.80 | ReplicaSet |
| [daemonset/unavailable](workloads.md#daemonsetunavailable) | warning | 0.70 | DaemonSet |
| [daemonset/misscheduled](workloads.md#daemonsetmisscheduled) | warning | 0.60 | DaemonSet |
| [job/failed](batch.md#jobfailed) | error | 0.85 | Job |
| [cronjob/suspended](batch.md#cronjobsuspended) | info | 0.90 | CronJob |
| [cronjob/job-failures](batch.md#cronjobjob-failures) | error | 0.75 | CronJob |
| [node/not-ready](node.md#nodenot-ready) | error / critical | 0.90 | Node |
| [node/pressure](node.md#nodepressure) | error | 0.85 | Node |
| [node/network-unavailable](node.md#nodenetwork-unavailable) | error | 0.80 | Node |
| [node/unschedulable](node.md#nodeunschedulable) | warning | 0.95 | Node |
| [service/no-endpoints](networking.md#serviceno-endpoints) | error | 0.80 | Service |
| [service/selector-mismatch](networking.md#serviceselector-mismatch) | error | 0.75 | Service |
| [service/target-port-mismatch](networking.md#servicetarget-port-mismatch) | warning | 0.70 | Service |
| [ingress/backend-missing](networking.md#ingressbackend-missing) | error | 0.85 | Ingress |
| [ingress/tls-secret-missing](networking.md#ingresstls-secret-missing) | warning | 0.75 | Ingress |
| [networkpolicy/no-matching-pods](networking.md#networkpolicyno-matching-pods) | warning | 0.60 | NetworkPolicy |
| [pvc/unbound](storage.md#pvcunbound) | warning | 0.80 | PersistentVolumeClaim |
| [pvc/missing-storageclass](storage.md#pvcmissing-storageclass) | error | 0.85 | PersistentVolumeClaim |
| [pvc/provisioner-stuck](storage.md#pvcprovisioner-stuck) | error | 0.80 | PersistentVolumeClaim |
| [pv/failed](storage.md#pvfailed) | error | 0.85 | PersistentVolume |
| [pv/released-retained](storage.md#pvreleased-retained) | warning | 0.70 | PersistentVolume |
| [storageclass/no-provisioner](storage.md#storageclassno-provisioner) | info | 0.90 | StorageClass |
| [storageclass/wait-for-first-consumer](storage.md#storageclasswait-for-first-consumer) | info | 0.80 | StorageClass |
| [hpa/scaling-disabled](reliability.md#hpascaling-disabled) | error | 0.80 | HorizontalPodAutoscaler |
| [hpa/missing-scale-target](reliability.md#hpamissing-scale-target) | error | 0.80 | HorizontalPodAutoscaler |
| [pdb/disruptions-blocked](reliability.md#pdbdisruptions-blocked) | warning | 0.70 | PodDisruptionBudget |
| [pdb/no-matching-pods](reliability.md#pdbno-matching-pods) | warning | 0.65 | PodDisruptionBudget |
| [lease/stale](reliability.md#leasestale) | warning | 0.60 | Lease |
| [rbac/missing-role](security.md#rbacmissing-role) | error | 0.80 | RoleBinding, ClusterRoleBinding |
| [rbac/no-subjects](security.md#rbacno-subjects) | warning | 0.70 | RoleBinding, ClusterRoleBinding |
| [csr/denied](security.md#csrdenied) | error | 0.85 | CertificateSigningRequest |
| [csr/pending](security.md#csrpending) | warning | 0.60 | CertificateSigningRequest |

## Selecting rules

Use the `--rule` and `--disable-rule` flags with `klue why` to control which
rules run:

```bash
# Run only specific rules
klue why pod my-pod -n default --rule pod/crashloop --rule pod/image-pull

# Disable low-signal rules
klue why deployment api -n prod --disable-rule builtin/warning-events
```

See [Flags](../usage/flags.md) for the complete flag reference.

## Rule categories

| Category | Page | Rules |
|----------|------|-------|
| Generic (any resource) | [Builtin](builtin.md) | `builtin/*` |
| Workloads | [Workloads](workloads.md) | `pod/*`, `deployment/*`, `statefulset/*`, `replicaset/*`, `daemonset/*` |
| Batch | [Batch](batch.md) | `job/*`, `cronjob/*` |
| Nodes | [Node](node.md) | `node/*` |
| Networking | [Networking](networking.md) | `service/*`, `ingress/*`, `networkpolicy/*` |
| Storage | [Storage](storage.md) | `pvc/*`, `pv/*`, `storageclass/*` |
| Reliability & autoscaling | [Reliability](reliability.md) | `hpa/*`, `pdb/*`, `lease/*` |
| Security & certificates | [Security](security.md) | `rbac/*`, `csr/*` |
