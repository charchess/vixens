# ADR-026: KEDA HTTP Add-on for Scale-to-Zero

**Date:** 2026-03-26
**Status:** Accepted
**Tags:** scaling, performance, keda

## Context

The cluster runs ~90 applications, many of which are rarely used (once per week or less). These apps consume memory and CPU permanently even when idle. With cluster memory often at 90%+, freeing resources from idle apps is valuable.

## Decision

Deploy **KEDA** (Kubernetes Event-Driven Autoscaling) with the **HTTP Add-on** for request-driven scale-to-zero on low-traffic applications.

### How It Works

1. App at rest → replicas: 0 (zero resources consumed)
2. HTTP request arrives → interceptor proxy buffers it → KEDA scales to 1
3. App starts (10-30s) → interceptor forwards the buffered request
4. After configurable idle period → KEDA scales back to 0

### Architecture

```
User → Traefik → Interceptor Proxy → App Pod (0→1→0)
                       ↕
                  KEDA Scaler (metrics)
                       ↕
                  KEDA Operator (scaling decisions)
```

### Components (5 pods, ~130Mi idle)

| Component | Role |
|---|---|
| keda-operator | Core autoscaling engine |
| keda-metrics-apiserver | Custom metrics for HPA |
| keda-admission-webhooks | Validate ScaledObject CRDs |
| http-add-on-interceptor | Proxy that buffers requests for scaled-to-zero apps |
| http-add-on-scaler | Reports HTTP traffic metrics to KEDA |

### Per-App Configuration

Each app needs an `HTTPScaledObject` CRD:
```yaml
apiVersion: http.keda.sh/v1alpha1
kind: HTTPScaledObject
metadata:
  name: my-app
spec:
  hosts: ["my-app.truxonline.com"]
  scaleTargetRef:
    name: my-app
    kind: Deployment
  replicas:
    min: 0
    max: 1
  scaledownPeriod: 300  # 5min idle → scale down
```

And the Ingress backend must point to the interceptor service instead of the app directly.

## Alternatives Evaluated

| Option | Verdict | Reason |
|---|---|---|
| kube-green | Schedule-based only | No wake-on-request (users get 503 during downtime) |
| Knative Serving | Overkill | Requires re-architecting all Deployments to Knative CRDs |
| Osiris | Dead | Archived project |
| kube-downscaler | Schedule-based | Same limitation as kube-green |

## Trade-offs

### Accepted

- **Cold start latency** (10-30s): First request after idle triggers pod startup. Acceptable for low-traffic tools.
- **5 additional pods**: ~130Mi permanent overhead. Offset by savings from scaled-to-zero apps.
- **Ingress backend change**: Per-app, the Ingress must point to the interceptor. Managed via Kustomize overlay.

### Mitigations

- **Loading page**: Interceptor can return a "starting up" response while pod boots.
- **Selective adoption**: Only apply to apps that are genuinely low-traffic. Keep critical apps always-on.
- **Configurable idle**: `scaledownPeriod` per app (5-30min depending on usage pattern).

## Candidate Apps

| App | Usage | RAM saved | Cold start |
|---|---|---|---|
| stirling-pdf | Occasional | ~200Mi | ~15s |
| it-tools | Occasional | ~50Mi | ~5s |
| headlamp | Admin only | ~100Mi | ~10s |
| g4f | Occasional | ~200Mi | ~10s |
| radar | Admin only | ~200Mi | ~10s |

## Implementation

- KEDA operator: Helm chart `kedacore/keda` via ArgoCD
- HTTP Add-on: Helm chart `kedacore/keda-add-ons-http` via ArgoCD
- Per-app: `HTTPScaledObject` in app overlay + Ingress backend patch
- Namespace: `keda` with control-plane tolerations

## References

- [KEDA HTTP Add-on](https://github.com/kedacore/http-add-on)
- [KEDA Docs](https://keda.sh/docs/)
- Issue: #2227
