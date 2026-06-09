# Scenario Library

This directory contains intentionally broken Kubernetes manifests for manual `klue` testing.

## Prerequisites

- A working Kubernetes context (`kubectl config current-context`)
- `klue` installed or built locally (`make build` then use `./bin/klue`)

## Scenario Index

| Scenario | Target resource | Expected rule(s) | Run command |
| --- | --- | --- | --- |
| `pod-crashloop` | Pod `crashloop-demo` | `pod/crashloop` | `./bin/klue why pod crashloop-demo -n klue-demo-crashloop` |
| `pod-config-missing` | Pod `config-missing-demo` | `pod/config-missing` | `./bin/klue why pod config-missing-demo -n klue-demo-config-missing` |
| `service-selector-mismatch` | Service `mismatch-demo` | `service/selector-mismatch`, often `service/no-endpoints` | `./bin/klue why service mismatch-demo -n klue-demo-selector-mismatch` |
| `ingress-backend-missing` | Ingress `backend-missing-demo` | `ingress/backend-missing` | `./bin/klue why ingress backend-missing-demo -n klue-demo-ingress-backend` |
| `pvc-missing-storageclass` | PVC `missing-sc-demo` | `pvc/missing-storageclass` | `./bin/klue why pvc missing-sc-demo -n klue-demo-pvc-sc` |
| `complex-crd-multi-failure` | Ingress `complex-gateway` (plus Service/PVC/RoleBinding/CRD checks) | `ingress/backend-missing`, `ingress/tls-secret-missing`, `service/target-port-mismatch`, `pvc/missing-storageclass`, `rbac/missing-role`, `builtin/orphaned-owner` | `./bin/klue why ingress complex-gateway -n klue-demo-complex-crd` |
| `certmanager-certificate-failed` | Certificate `backend-demo-cert` (and Ingress `backend-demo`) | `builtin/failed-condition`, `ingress/tls-secret-missing` | `./bin/klue why certificate backend-demo-cert --api-version cert-manager.io/v1 -n klue-demo-certmanager-failed` |

## Run One Scenario

```bash
kubectl apply -f examples/scenarios/pod-config-missing/namespace.yaml -f examples/scenarios/pod-config-missing/workload.yaml
./bin/klue why pod config-missing-demo -n klue-demo-config-missing
kubectl delete namespace klue-demo-config-missing
```

## Apply All Scenarios

```bash
set -euo pipefail
for dir in examples/scenarios/*/; do
  kubectl apply -f "${dir}namespace.yaml" -f "${dir}workload.yaml"
done
```

## Teardown All Scenario Namespaces

```bash
set -euo pipefail
for ns in \
  klue-demo-crashloop \
  klue-demo-config-missing \
  klue-demo-selector-mismatch \
  klue-demo-ingress-backend \
  klue-demo-pvc-sc \
  klue-demo-complex-crd \
  klue-demo-certmanager-failed; do
  kubectl delete namespace "$ns" --ignore-not-found
done

kubectl delete crd widgets.demo.klue.io --ignore-not-found
```

## Notes

- Scenarios are plain YAML (no Helm/Kustomize).
- `pod-crashloop` usually needs about 20-40 seconds before `CrashLoopBackOff` appears.
- Findings may vary slightly by cluster version and event timing, but the listed rules should be reproducible.
