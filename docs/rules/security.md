---
description: Diagnostic rules for Kubernetes security resources — RBAC bindings, Roles, and CertificateSigningRequests.
tags:
  - rules
  - reference
  - security
  - rbac
  - csr
---

# Security rules

The security rules diagnose problems with **RBAC** (RoleBinding,
ClusterRoleBinding) and **CertificateSigningRequest (CSR)** resources.
Misconfigurations here silently remove permissions or block TLS certificate
issuance.

---

## RBAC rules

### `rbac/missing-role`

**Severity:** error | **Confidence:** 0.80 | **Applies to:** RoleBinding, ClusterRoleBinding

Detects RoleBindings and ClusterRoleBindings whose `roleRef` points to a Role or
ClusterRole that does not exist in the cluster. A binding that references a
missing role confers no permissions.

#### When it fires

The `roleRef.name` in a RoleBinding or ClusterRoleBinding refers to a Role or
ClusterRole that is not present in the graph.

#### Example finding

> **Binding references missing ClusterRole "app-reader"**
> The binding "web-reader" grants a role that does not exist, so it confers no
> permissions.

#### Remediation

```bash
# Verify the referenced role exists
kubectl get role <name> -n <namespace>
kubectl get clusterrole <name>
```

Create the missing Role/ClusterRole or update the binding to reference an
existing one.

---

### `rbac/no-subjects`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** RoleBinding, ClusterRoleBinding

Detects RoleBindings and ClusterRoleBindings with an empty `subjects` list. A
binding with no subjects grants its role to nobody and has no effect.

#### When it fires

The binding's `subjects` list is empty or absent.

#### Example finding

> **Binding has no subjects**
> The binding "web-reader" has an empty subjects list, so it grants its role to
> no users, groups, or service accounts.

#### Remediation

```bash
kubectl describe rolebinding <name> -n <namespace>
```

Either add the intended subjects or remove the unused binding.

---

## CertificateSigningRequest rules

### `csr/denied`

**Severity:** error | **Confidence:** 0.85 | **Applies to:** CertificateSigningRequest

Detects CertificateSigningRequests that were denied or failed. A denied or
failed CSR will not produce a certificate, and the requester must submit a new
request after addressing the reason.

#### When it fires

The CSR's `status.conditions` contain a condition of type `Denied` or `Failed`.

#### Example finding

> **CertificateSigningRequest was Denied**
> The signing request will not produce a certificate. The requester must submit
> a new CSR after addressing the denial reason.

#### Remediation

```bash
kubectl describe csr <name>
```

Review the denial reason and correct the CSR attributes (key usage, subject,
signer name) before re-submitting.

---

### `csr/pending`

**Severity:** warning | **Confidence:** 0.60 | **Applies to:** CertificateSigningRequest

Detects CertificateSigningRequests that are still awaiting approval. A pending
CSR has been submitted but neither approved, denied, nor failed — it is waiting
for a human approver or an automated approval controller.

#### When it fires

The CSR has no `Approved`, `Denied`, or `Failed` condition. The finding
includes the `signerName` so the right approver can be identified.

#### Example finding

> **CertificateSigningRequest is pending approval**
> The CSR has neither been approved nor denied. It needs an approver (manual or
> controller) to proceed.

#### Remediation

If the request is legitimate, approve it:

```bash
kubectl certificate approve <name>
```

To review the request before approving:

```bash
kubectl describe csr <name>
```
