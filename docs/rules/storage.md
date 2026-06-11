---
description: Diagnostic rules for Kubernetes storage resources — PersistentVolumeClaim, PersistentVolume, and StorageClass.
tags:
  - rules
  - reference
  - storage
  - pvc
  - pv
  - storageclass
---

# Storage rules

The storage rules diagnose problems with **PersistentVolumeClaim (PVC)**,
**PersistentVolume (PV)**, and **StorageClass** resources. Storage issues
commonly surface as pods stuck in `Pending` because a required volume cannot be
mounted.

---

## PersistentVolumeClaim rules

### `pvc/unbound`

**Severity:** warning | **Confidence:** 0.80 | **Applies to:** PersistentVolumeClaim

Detects PVCs that are not in the `Bound` phase. An unbound PVC prevents any pod
that mounts it from being scheduled.

#### When it fires

`status.phase` is not `Bound` (typically `Pending` or `Lost`).

#### Example finding

> **PersistentVolumeClaim is not bound**
> The claim has not been bound to a volume. Check the storage class and
> provisioner.

#### Remediation

```bash
kubectl describe pvc <name> -n <namespace>
kubectl get pv
```

---

### `pvc/missing-storageclass`

**Severity:** error | **Confidence:** 0.85 | **Applies to:** PersistentVolumeClaim

Detects PVCs that reference a StorageClass which does not exist in the cluster.
The PVC can never be dynamically provisioned.

#### When it fires

`spec.storageClassName` refers to a StorageClass that is not present in the
cluster.

#### Example finding

> **StorageClass "fast-ssd" does not exist**
> The PVC references a storage class that is not defined, so it can never be
> provisioned.

#### Remediation

```bash
# List available storage classes
kubectl get storageclass

# Inspect the PVC
kubectl describe pvc <name> -n <namespace>
```

---

### `pvc/provisioner-stuck`

**Severity:** error | **Confidence:** 0.80 | **Applies to:** PersistentVolumeClaim

Detects PVCs whose external volume provisioner has reported a failure event
(`ProvisioningFailed`). The rule parses the event message to identify the root
cause.

#### When it fires

The PVC is not `Bound` and has a `ProvisioningFailed` warning event. Common
root causes detected from event messages:

| Cause | Explanation |
|-------|-------------|
| `quota-exceeded` | Quota exceeded in the storage backend or namespace |
| `permission-denied` | IAM/RBAC policy denied by the storage backend |
| `topology-constraint` | Zone or topology constraints preventing provisioning |
| _(other)_ | External provisioner failure |

#### Example finding

> **Volume provisioning is failing**
> The external provisioner reported a failure while creating the volume. Likely
> cause: quota exceeded in the storage backend or namespace.

#### Remediation

```bash
kubectl describe pvc <name> -n <namespace>
```

Also check the logs of the provisioner pod (CSI driver controller):

```bash
kubectl get pods -n kube-system | grep provisioner
kubectl logs -n kube-system <provisioner-pod>
```

---

## PersistentVolume rules

### `pv/failed`

**Severity:** error | **Confidence:** 0.85 | **Applies to:** PersistentVolume

Detects PersistentVolumes in the `Failed` phase. This happens when the volume's
automatic reclamation (recycle or delete) fails and the backing storage may need
manual cleanup.

#### When it fires

`status.phase` is `Failed`.

#### Example finding

> **PersistentVolume is in the Failed phase**
> The volume's automatic reclamation (recycle or delete) failed, so the backing
> storage may need manual cleanup.

#### Remediation

```bash
kubectl describe pv <name>
```

---

### `pv/released-retained`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** PersistentVolume

Detects PersistentVolumes that are in the `Released` phase with a `Retain`
reclaim policy. Such volumes will never be automatically re-bound and must be
manually reclaimed or cleaned up.

#### When it fires

`status.phase` is `Released` **and** `spec.persistentVolumeReclaimPolicy` is
`Retain`.

#### Example finding

> **PersistentVolume is released but retained**
> The bound claim was deleted but the Retain policy keeps the volume. It will
> not be re-bound automatically and must be cleaned up or re-claimed manually.

#### Remediation

To make the volume available for re-binding, clear the `claimRef`:

```bash
kubectl patch pv <name> -p '{"spec":{"claimRef":null}}'
```

Or delete the PV if the backing storage should be released:

```bash
kubectl delete pv <name>
```

---

## StorageClass rules

### `storageclass/no-provisioner`

**Severity:** info | **Confidence:** 0.90 | **Applies to:** StorageClass

Flags StorageClasses that use the `kubernetes.io/no-provisioner` sentinel value,
which means no dynamic provisioning is available. PVCs that request this class
will remain `Pending` until a matching PersistentVolume is created manually.

#### When it fires

`spec.provisioner` is `kubernetes.io/no-provisioner`.

#### Example finding

> **StorageClass cannot dynamically provision volumes**
> This StorageClass uses the no-provisioner sentinel, so PVCs using it stay
> Pending until a matching PersistentVolume is created manually.

#### Remediation

Pre-provision a matching PersistentVolume:

```bash
kubectl get pv --selector=<class-selector>
```

---

### `storageclass/wait-for-first-consumer`

**Severity:** info | **Confidence:** 0.80 | **Applies to:** StorageClass

Flags StorageClasses with `volumeBindingMode: WaitForFirstConsumer`. PVCs using
this class intentionally stay `Pending` until a consuming pod is scheduled; the
binding and provisioning are deferred to that moment.

This finding is informational — a `Pending` PVC is expected behavior for this
binding mode when no consuming pod has been scheduled yet.

#### When it fires

`spec.volumeBindingMode` is `WaitForFirstConsumer`.

#### Example finding

> **StorageClass defers binding until a consumer is scheduled**
> PVCs using this class stay Pending by design until a pod that mounts them is
> scheduled. A Pending PVC may simply have no consuming pod yet.

#### Remediation

Confirm a pod consuming the PVC has been scheduled:

```bash
kubectl get pods -n <namespace> -o wide
kubectl describe pvc <name> -n <namespace>
```
