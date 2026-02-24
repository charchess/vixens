# Guaranteed QoS Sizing

Kyverno policy for Orichalcum tier applications requiring Guaranteed QoS class.

## Overview

This policy provides **Guaranteed Quality of Service (QoS)** resource sizing through Kyverno mutation. Unlike Burstable sizing (the default), Guaranteed QoS requires:

```yaml
requests.cpu == limits.cpu
requests.memory == limits.memory
```

## When to Use Guaranteed QoS

### ✅ Use Guaranteed for:
- **Orichalcum tier applications** (highest maturity level)
- Critical infrastructure components
- Apps requiring strict resource isolation
- Workloads sensitive to resource contention

### ❌ Do NOT use Guaranteed for:
- Most general applications
- Development/test workloads
- Burst-friendly applications
- Resource-constrained environments

**WARNING:** Guaranteed QoS consumes more cluster resources (no overcommit possible). Most apps should use Burstable sizing for better resource utilization.

## Available Sizes

| Label | CPU | Memory | Use Case |
|-------|-----|--------|----------|
| `G-small` | 50m | 128Mi | Minimal apps, test services |
| `G-medium` | 200m | 512Mi | Small production apps |
| `G-large` | 1000m | 2Gi | Medium production apps |
| `G-xl` | 2000m | 4Gi | Large critical apps |

## Usage

Add the sizing label to your pod or deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-critical-app
spec:
  template:
    metadata:
      labels:
        vixens.io/sizing: "G-small"  # or G-medium, G-large, G-xl
    spec:
      containers:
        - name: app
          image: myapp:v1.0.0
```

Kyverno will automatically mutate the pod to add matching requests and limits.

## QoS Classes Explained

### Guaranteed (requests == limits)
- **Scheduling:** Pod is scheduled only if full resource amount available
- **Eviction:** Last to be evicted when node is under pressure
- **Overcommit:** Not possible - resources are fully reserved
- **Use for:** Critical apps, Orichalcum tier

### Burstable (requests < limits)
- **Scheduling:** Scheduled based on requests, can burst to limits
- **Eviction:** May be evicted before Guaranteed pods
- **Overcommit:** Possible - unused resources available to others
- **Use for:** Most production apps (Gold through Diamond tier)

### BestEffort (no requests/limits)
- **Scheduling:** No guarantees
- **Eviction:** First to be evicted
- **Overcommit:** Maximum
- **Use for:** Development, batch jobs, non-critical workloads

## Policy Reference

- **Policy Name:** `sizing-guaranteed`
- **Location:** `apps/00-infra/kyverno/base/policies/sizing-guaranteed.yaml`
- **Action:** Enforce (mutates pods automatically)
- **Scope:** Cluster-wide

## Migration Guide

To migrate an app to Guaranteed QoS:

1. **Verify app needs Orichalcum tier** - Guaranteed is only required for Orichalcum
2. **Select appropriate size** - Use the smallest size that meets requirements
3. **Add sizing label:**
   ```yaml
   labels:
     vixens.io/sizing: "G-small"
   ```
4. **Deploy and verify:**
   ```bash
   kubectl get pod <pod> -o jsonpath='{.status.qosClass}'
   # Should output: Guaranteed
   ```

## Troubleshooting

### Pod fails to schedule
Guaranteed pods require the full resource amount to be available. If scheduling fails:
- Check node resources: `kubectl describe node <node>`
- Consider using a smaller G-* size
- Or use Burstable sizing instead

### QoS class not Guaranteed
Check that Kyverno mutated the pod:
```bash
kubectl get pod <pod> -o yaml | grep -A5 resources
```

If requests != limits, verify:
1. Label is correct: `vixens.io/sizing: G-small`
2. Kyverno is running: `kubectl get pods -n kyverno`
3. Policy is active: `kubectl get clusterpolicy sizing-guaranteed`

## See Also

- [ADR-022: 7-Tier Goldification System](../../adr/022-7-tier-goldification-system.md)
- [Sizing Migration Guide](../../guides/sizing-migration.md)
- [Kyverno sizing-mutate policy](./sizing-mutate.yaml) - Burstable sizing
