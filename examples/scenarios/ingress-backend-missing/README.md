# ingress-backend-missing

Creates an Ingress that points to a Service which does not exist.

- Expected primary rule: `ingress/backend-missing`

## Apply

```bash
kubectl apply -f namespace.yaml -f workload.yaml
```

## Wait

Immediate (no controller is required for this API-level check).

## Run klue

```bash
./bin/klue why ingress backend-missing-demo -n klue-demo-ingress-backend
```

## Expected signal

Look for a finding that backend service `ghost-service` is missing.

## Teardown

```bash
kubectl delete namespace klue-demo-ingress-backend
```
