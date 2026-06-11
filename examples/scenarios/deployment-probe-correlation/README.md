# deployment-probe-correlation

Creates a small but multi-resource setup to exercise event and log correlation:

- **Deployment** `probe-correlation-demo` with 2 replicas
- **Service** `probe-correlation-demo` fronting the app pods
- **Service** `redis` with a selector that matches no pods (upstream dependency is unreachable)
- **Pods** run a multi-container template where:
  - `app` never serves `/healthz`, so readiness probes fail and Kubernetes emits `Unhealthy` warning events
  - `app` logs repeated `connection refused` messages to `redis:6379`
  - `metrics` stays idle so the pod remains `Running` with a not-ready `app` container

This is intentionally more complex than single-pod examples: diagnosing from the Deployment traverses owned Pods and related Services while combining warning events and container logs on the same finding.

## Expected rules

| Target | Primary signal | Correlation notes |
| --- | --- | --- |
| Pod (owned replica) | `pod/probe-failure` | `Unhealthy` event + log excerpt on the same finding |
| Deployment | `deployment/unavailable` | Summary only; traversal stops before pod-level correlation |
| Deployment (with unavailable disabled) | `pod/probe-failure` | Traverses to pods and surfaces correlated evidence |

Related graph signals you may also see depending on timing:

- `service/no-endpoints` on `redis` (no matching backends)
- `builtin/warning-events` suppressed when typed probe finding already captured the same event

## Apply

```bash
kubectl apply -f namespace.yaml -f workload.yaml
```

## Wait

Give the deployment a short time to create pods and accumulate probe failures/events:

```bash
kubectl rollout status deployment/probe-correlation-demo -n klue-demo-probe-correlation --timeout=120s
kubectl get pods -n klue-demo-probe-correlation -w
```

Pods should settle into `Running` with `READY 1/2` (metrics ready, app not ready).

## Run klue

Pod-level diagnosis (best for seeing event+log correlation directly):

```bash
POD="$(kubectl get pod -n klue-demo-probe-correlation -l app=probe-correlation -o jsonpath='{.items[0].metadata.name}')"
./bin/klue why pod "$POD" -n klue-demo-probe-correlation
./bin/klue why pod "$POD" -n klue-demo-probe-correlation --debug -o json
```

Deployment-level diagnosis (graph context + unavailable summary):

```bash
./bin/klue why deployment probe-correlation-demo -n klue-demo-probe-correlation
```

Deployment-level diagnosis with traversal to pod correlation findings:

```bash
./bin/klue why deployment probe-correlation-demo -n klue-demo-probe-correlation \
  --disable-rule deployment/unavailable \
  --debug
```

## Expected signal

Look for a finding similar to **Pod is failing health probes** (`pod/probe-failure`) with:

- warning **event** evidence (`Unhealthy`, readiness probe failure)
- **log** evidence containing `connection refused` from the `app` container
- explanation text enriched from log signals

With `--debug`, inspect:

- log candidate reason `running-not-ready-with-probe-warning`
- correlation counters (`correlatedFindings`, `suppressedFindings`)
- selected rules list

## Teardown

```bash
kubectl delete namespace klue-demo-probe-correlation
```
