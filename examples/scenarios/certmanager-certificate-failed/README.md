# certmanager-certificate-failed

Creates a realistic ACME-style cert-manager `Issuer` configuration that cannot complete because the DNS provider token secret is missing, plus a normal app path (`Pod` + `Service` + `Ingress`) that expects the certificate's TLS secret.

- Expected primary signal on the `Certificate`: `builtin/failed-condition` (for a failing `Ready=False` status condition)
- Expected ingress impact: `ingress/tls-secret-missing` on `Ingress/backend-demo`
- You may also see warning-event findings depending on controller timing.

## Prerequisite

This scenario assumes cert-manager CRDs/controllers are already installed in the cluster.

## Apply

```bash
kubectl apply -f namespace.yaml -f workload.yaml
```

## Wait

Wait ~10-30 seconds for cert-manager to reconcile:

```bash
kubectl get issuer,certificate,ingress,service,pod -n klue-demo-certmanager-failed -w
```

## Run klue

```bash
./bin/klue why certificate backend-demo-cert --api-version cert-manager.io/v1 -n klue-demo-certmanager-failed
./bin/klue why ingress backend-demo -n klue-demo-certmanager-failed
```

## Expected signal

Look for findings that the certificate is not ready because the referenced ACME issuer cannot proceed (missing Cloudflare API token secret), even though the issuer uses a realistic email/server configuration (`dev@example.com`, Let's Encrypt staging directory). The Ingress should also report missing TLS secret because `backend-demo-tls` is never issued.

## Teardown

```bash
kubectl delete namespace klue-demo-certmanager-failed --ignore-not-found
```
