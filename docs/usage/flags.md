---
description: Global and why-specific CLI flags with defaults, duration syntax, and diagnostic rule IDs.
tags:
  - usage
  - reference
---

# Flags

## Global flags

These flags are persistent on the root command and apply to every subcommand.

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--namespace` | `-n` | `default` | Namespace to diagnose the resource in |
| `--kubeconfig` | — | *(empty)* | Path to kubeconfig; empty uses standard discovery |
| `--context` | — | *(empty)* | Kubeconfig context; empty uses current context |
| `--fetch-concurrency` | — | `6` | Max parallel Kubernetes list operations while fetching |
| `--client-qps` | — | `30` | API client rate limit in queries per second |
| `--client-burst` | — | `60` | API client burst size |
| `--timeout` | — | `0` | Max time to spend fetching cluster state (`0` = no extra limit) |

Connection-related flags are described in more detail in
[Kubernetes access](../getting-started/kubernetes-access.md).

## `why` flags

These flags apply only to `klue why`.

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--api-version` | — | *(empty)* | Disambiguate resource group/version (common for CRDs) |
| `--max-depth` | — | `0` | Max graph hops from target (`0` = unlimited) |
| `--event-window` | — | `1h` | Max age of warning events to consider relevant |
| `--terminating-grace` | — | `5m` | Time before terminating resources are reported as stuck |
| `--lease-stale-multiplier` | — | `4` | Lease durations before a holder is considered stale |
| `--no-namespace-scan` | — | `false` | Skip scanning unvisited namespace resources when graph traversal finds no issues |
| `--no-fetch-logs` | — | `false` | Skip fetching container logs for unhealthy related pods |
| `--log-tail-lines` | — | `100` | Trailing log lines to fetch per container |
| `--debug` | — | `false` | Include debug metadata (candidate reasons, fetch stats, correlation/dedupe details) |
| `--disable-rule` | — | — | Disable a diagnostic rule by ID (repeatable) |
| `--only-rule` | — | — | Run only the listed rule IDs (repeatable) |
| `--output` | `-o` | `text` | Output format: `text`, `json`, or `markdown` |

### Duration values

Flags such as `--event-window`, `--terminating-grace`, and `--timeout` accept Go
duration syntax: `30s`, `5m`, `1h`.

### Rule selection

!!! warning "Mutually exclusive"
    `--only-rule` and `--disable-rule` cannot be used together.

Rule IDs follow the `category/name` pattern. Common examples:

| Rule ID | Detects |
|---------|---------|
| `pod/crashloop` | Containers in a crash loop |
| `pod/image-pull` | Image pull failures, enriched by structured warning-event signals |
| `pod/config-missing` | Missing ConfigMaps or Secrets |
| `deployment/rollout-stuck` | Stuck Deployment rollouts |
| `service/selector-mismatch` | Service selector not matching any Pod |
| `ingress/backend-missing` | Ingress backend Service not found |
| `pvc/missing-storageclass` | PVC referencing a missing StorageClass |
| `builtin/warning-events` | Recent warning events on the resource |
| `builtin/log-signal` | Failure patterns detected in container logs |
| `builtin/terminating-stuck` | Resources stuck in terminating state |

```bash
# Run a single rule
klue why pod web-abc -n default --only-rule pod/crashloop

# Disable noisy rules
klue why deployment api -n prod --disable-rule builtin/warning-events
```

Unknown rule IDs produce an error listing the invalid values.

### Evidence correlation behavior

`klue why` correlates warning events and container logs during diagnosis:

- Warning events are indexed per involved object and consumed by rules as typed
  evidence, with a shared parser for image-pull, scheduling, probe, mount, and
  provisioning warning messages.
- Log fetching stays bounded (`--log-tail-lines`, candidate cap) and is focused
  on unhealthy containers related to the target, with selection reasons tracked
  in debug metadata.
- Some pod findings (notably `pod/image-pull` and `pod/probe-failure`) combine
  event and log/status evidence to improve explanation quality and confidence.
- Generic fallback findings (`builtin/warning-events`) are suppressed when the
  same event evidence is already captured by a stronger typed finding.

??? note "All built-in rule IDs"

    | Rule ID | Resource kind |
    |---------|---------------|
    | `pod/crashloop` | Pod |
    | `pod/image-pull` | Pod |
    | `pod/config-missing` | Pod |
    | `pod/pending` | Pod |
    | `pod/probe-failure` | Pod |
    | `pod/mount-failure` | Pod |
    | `deployment/rollout-stuck` | Deployment |
    | `deployment/unavailable` | Deployment |
    | `statefulset/unavailable` | StatefulSet |
    | `statefulset/rollout-stuck` | StatefulSet |
    | `replicaset/unavailable` | ReplicaSet |
    | `replicaset/replica-failure` | ReplicaSet |
    | `daemonset/unavailable` | DaemonSet |
    | `daemonset/misscheduled` | DaemonSet |
    | `job/failed` | Job |
    | `cronjob/suspended` | CronJob |
    | `cronjob/job-failures` | CronJob |
    | `node/not-ready` | Node |
    | `node/pressure` | Node |
    | `node/network-unavailable` | Node |
    | `node/unschedulable` | Node |
    | `service/no-endpoints` | Service |
    | `service/selector-mismatch` | Service |
    | `service/target-port-mismatch` | Service |
    | `pvc/unbound` | PVC |
    | `pvc/missing-storageclass` | PVC |
    | `pvc/provisioner-stuck` | PVC |
    | `pv/failed` | PV |
    | `pv/released-retained` | PV |
    | `storageclass/no-provisioner` | StorageClass |
    | `storageclass/wait-for-first-consumer` | StorageClass |
    | `ingress/backend-missing` | Ingress |
    | `ingress/tls-secret-missing` | Ingress |
    | `hpa/scaling-disabled` | HPA |
    | `hpa/missing-scale-target` | HPA |
    | `pdb/disruptions-blocked` | PDB |
    | `pdb/no-matching-pods` | PDB |
    | `networkpolicy/no-matching-pods` | NetworkPolicy |
    | `rbac/missing-role` | RBAC binding |
    | `rbac/no-subjects` | RBAC binding |
    | `csr/denied` | CertificateSigningRequest |
    | `csr/pending` | CertificateSigningRequest |
    | `lease/stale` | Lease |
    | `builtin/warning-events` | Any |
    | `builtin/log-signal` | Pod |
    | `builtin/failed-condition` | Any |
    | `builtin/terminating-stuck` | Any |
    | `builtin/missing-reference` | Any |
    | `builtin/orphaned-owner` | Any |

See [why](why.md) for usage examples.
