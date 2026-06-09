# pvc-missing-storageclass

Creates a PVC that references a StorageClass that does not exist.

- Expected primary rule: `pvc/missing-storageclass`

## Apply

```bash
kubectl apply -f namespace.yaml -f workload.yaml
```

## Wait

Usually immediate.

## Run klue

```bash
./bin/klue why pvc missing-sc-demo -n klue-demo-pvc-sc
```

## Expected signal

Look for a finding indicating missing StorageClass `does-not-exist-sc`.

## Teardown

```bash
kubectl delete namespace klue-demo-pvc-sc
```
