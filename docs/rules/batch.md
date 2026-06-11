---
description: Diagnostic rules for Kubernetes batch workloads — Job and CronJob.
tags:
  - rules
  - reference
  - batch
  - job
  - cronjob
---

# Batch rules

The batch rules cover **Job** and **CronJob** resources, diagnosing failures and
misconfigurations that prevent scheduled or one-off workloads from completing
successfully.

---

## Job rules

### `job/failed`

**Severity:** error | **Confidence:** 0.85 | **Applies to:** Job

Detects Jobs whose `Failed` condition is `True`. The rule tailors its
explanation to the specific failure reason — backoff limit exhaustion, active
deadline exceeded, or a general failure — and attaches container log evidence
from owned pods where available.

#### When it fires

The Job's `status.conditions` contain a `Failed` condition with status `True`.

Failure reasons and their explanations:

| Reason | Title | Explanation |
|--------|-------|-------------|
| `BackoffLimitExceeded` | Job exceeded its backoff limit | The Job retried its pods up to `backoffLimit` and gave up. The container is failing on every attempt. |
| `DeadlineExceeded` | Job exceeded its active deadline | The Job ran longer than `activeDeadlineSeconds` and was terminated before completing. |
| _(other)_ | Job failed | The Job did not complete successfully. Inspect the failed pods. |

#### Example finding

> **Job exceeded its backoff limit**
> The Job retried its pods up to backoffLimit and gave up. The container is
> failing on every attempt.

#### Remediation

```bash
# Inspect failed Job pods and their logs
kubectl logs job/<name> -n <namespace>
kubectl describe job <name> -n <namespace>
```

---

## CronJob rules

### `cronjob/suspended`

**Severity:** info | **Confidence:** 0.90 | **Applies to:** CronJob

Flags CronJobs that have `spec.suspend: true`, which prevents any new Jobs from
being created. This state may be intentional (for example during maintenance) but
is surfaced as an `info` finding so it is visible in diagnoses.

#### When it fires

`spec.suspend` is `true`.

#### Example finding

> **CronJob is suspended**
> The CronJob is suspended, so no new Jobs are created until it is resumed.
> This may be intentional.

#### Remediation

If the CronJob should be running:

```bash
kubectl patch cronjob <name> -n <namespace> -p '{"spec":{"suspend":false}}'
```

---

### `cronjob/job-failures`

**Severity:** error | **Confidence:** 0.75 | **Applies to:** CronJob

Detects CronJobs whose most recently owned Jobs are in a failed state. The
scheduled task is not completing successfully.

#### When it fires

One or more Jobs owned by the CronJob (visible in the resource graph) have a
`Failed` condition set to `True`.

#### Example finding

> **CronJob has 2 failing Job(s)**
> Jobs created by this CronJob are failing, so the scheduled task is not
> completing successfully.

#### Remediation

```bash
# List Jobs created by the CronJob, sorted by creation time
kubectl get jobs -n <namespace> --sort-by=.metadata.creationTimestamp

# Inspect the most recent failed Job
kubectl describe job <job-name> -n <namespace>
kubectl logs job/<job-name> -n <namespace>
```
