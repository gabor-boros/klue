# pod-crashloop

Creates a Pod that exits with status code `1` repeatedly, so Kubernetes puts it into `CrashLoopBackOff`.

- Expected primary rule: `pod/crashloop`

## Apply

```bash
kubectl apply -f namespace.yaml -f workload.yaml
```

## Wait

Wait around 20-40 seconds for restart attempts to accumulate:

```bash
kubectl get pod crashloop-demo -n klue-demo-crashloop -w
```

## Run klue

```bash
./bin/klue why pod crashloop-demo -n klue-demo-crashloop
```

## Expected signal

Look for a summary/root-cause mentioning `CrashLoopBackOff`. When log access is
available, klue also surfaces relevant log excerpts (for example the intentional
exit message from the crashing container). With `--debug`, JSON output also
shows log-candidate selection reasons and correlation metadata.

## Teardown

```bash
kubectl delete namespace klue-demo-crashloop
```
