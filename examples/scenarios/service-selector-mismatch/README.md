# service-selector-mismatch

Creates a Service whose selector does not match any running Pod labels.

- Expected primary rule: `service/selector-mismatch`
- Often also appears: `service/no-endpoints`

## Apply

```bash
kubectl apply -f namespace.yaml -f workload.yaml
```

## Wait

Usually immediate.

## Run klue

```bash
./bin/klue why service mismatch-demo -n klue-demo-selector-mismatch
```

## Expected signal

Look for findings indicating selector mismatch and/or missing ready endpoints.

## Teardown

```bash
kubectl delete namespace klue-demo-selector-mismatch
```
