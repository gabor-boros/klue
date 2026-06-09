# pod-config-missing

Creates a Pod that references a ConfigMap which does not exist.

- Expected primary rule: `pod/config-missing`

## Apply

```bash
kubectl apply -f namespace.yaml -f workload.yaml
```

## Wait

Usually immediate; this pod commonly enters `CreateContainerConfigError`.

## Run klue

```bash
./bin/klue why pod config-missing-demo -n klue-demo-config-missing
```

## Expected signal

Look for a finding that the referenced ConfigMap is missing.

## Teardown

```bash
kubectl delete namespace klue-demo-config-missing
```
