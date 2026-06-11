---
description: Diagnostic rules for Kubernetes networking resources — Service, Ingress, and NetworkPolicy.
tags:
  - rules
  - reference
  - networking
  - service
  - ingress
  - networkpolicy
---

# Networking rules

The networking rules diagnose problems with **Service**, **Ingress**, and
**NetworkPolicy** resources. Networking misconfigurations are a common source of
hard-to-debug application failures because the error only manifests at request
time.

---

## Service rules

### `service/no-endpoints`

**Severity:** error | **Confidence:** 0.80 | **Applies to:** Service

Detects Services that have no ready endpoints. Traffic to the service will fail
because no healthy pods back it.

#### When it fires

The Service has no ready addresses in its associated EndpointSlice(s). This can
happen when:

- No pods exist with labels matching the service selector.
- All matching pods are unhealthy (not `Ready`).
- The service uses a manual `Endpoints` resource that is empty.

#### Example finding

> **Service has no ready endpoints**
> Traffic to the service will fail because no ready pods back it. The selected
> pods may be unhealthy or missing.

#### Remediation

```bash
kubectl get endpointslices -n <namespace> -l kubernetes.io/service-name=<name>
kubectl get pods -n <namespace> --show-labels
```

---

### `service/selector-mismatch`

**Severity:** error | **Confidence:** 0.75 | **Applies to:** Service

Detects Services whose label selector matches no pods in the namespace. Unlike
`service/no-endpoints` — which fires when pods exist but are unhealthy — this
rule fires when the selector itself is misconfigured.

#### When it fires

The Service has a non-empty `spec.selector` and no pod in the namespace carries
all of the required labels.

#### Example finding

> **Service selector matches no pods**
> No pod in the namespace carries all of the labels in the service selector.

#### Remediation

```bash
# Compare the service selector with pod labels
kubectl get pods -n <namespace> --show-labels
kubectl describe service <name> -n <namespace>
```

---

### `service/target-port-mismatch`

**Severity:** warning | **Confidence:** 0.70 | **Applies to:** Service

Detects Services whose `targetPort` does not match any container port exposed by
the selected pods. Connections to the service will be refused at the pod level.

#### When it fires

For each port in the service spec, the rule checks that the `targetPort` (by
number or by named port) matches at least one container port on the pods
selected by the service.

#### Example finding

> **Service port 80 targets "http" which no pod exposes**
> The service target port does not match any container port on the selected
> pods, so connections will be refused.

#### Remediation

```bash
# Align the service targetPort with the container port
kubectl describe service <name> -n <namespace>
kubectl get pods -n <namespace> --selector=<selector> -o jsonpath='{.items[*].spec.containers[*].ports}'
```

---

## Ingress rules

### `ingress/backend-missing`

**Severity:** error | **Confidence:** 0.85 | **Applies to:** Ingress

Detects Ingress resources whose backend service does not exist in the cluster.
Traffic routed to the missing service will return errors.

#### When it fires

A `spec.rules[].http.paths[].backend.service.name` or
`spec.defaultBackend.service.name` in the Ingress refers to a Service that is
not present in the namespace.

#### Example finding

> **Ingress backend service "api" does not exist**
> The ingress routes traffic to service "api" which is not present in namespace
> "default".

#### Remediation

```bash
kubectl get service <name> -n <namespace>
kubectl describe ingress <name> -n <namespace>
```

---

### `ingress/tls-secret-missing`

**Severity:** warning | **Confidence:** 0.75 | **Applies to:** Ingress

Detects Ingress resources whose TLS configuration references a Secret that does
not exist. TLS termination will fail and the ingress controller may fall back to
plain HTTP or return a certificate error.

#### When it fires

A `spec.tls[].secretName` in the Ingress refers to a Secret that is not present
in the namespace.

!!! note "cert-manager integration"
    If a cert-manager `Certificate` resource is configured to produce the TLS
    secret, klue detects the producer edge in the graph and suppresses this
    finding — the secret will be created once the certificate is issued.

#### Example finding

> **Ingress TLS secret "web-tls" does not exist**
> The ingress references TLS secret "web-tls" which is not present in namespace
> "default", so TLS termination will fail.

#### Remediation

```bash
kubectl get secret <name> -n <namespace>
kubectl describe ingress <name> -n <namespace>
```

---

## NetworkPolicy rules

### `networkpolicy/no-matching-pods`

**Severity:** warning | **Confidence:** 0.60 | **Applies to:** NetworkPolicy

Detects NetworkPolicies whose `spec.podSelector` matches no pods in the
namespace. A policy that matches no pods neither allows nor restricts any
traffic — it is effectively a no-op.

#### When it fires

The NetworkPolicy has a non-empty `spec.podSelector` (at least one `matchLabel`
or `matchExpression`) and no pod in the namespace carries all of the required
labels.

!!! note
    A **completely empty** `podSelector` is intentional — it matches all pods
    in the namespace — and does not trigger this rule.

#### Example finding

> **NetworkPolicy selects no pods**
> The policy's pod selector matches no pods in the namespace, so it neither
> allows nor restricts any traffic. The selector may be misconfigured.

#### Remediation

```bash
kubectl get pods -n <namespace> --show-labels
kubectl describe networkpolicy <name> -n <namespace>
```
