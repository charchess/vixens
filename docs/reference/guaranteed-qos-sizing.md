# Guaranteed QoS Sizing

✅ **IMPLEMENTED** - Kyverno sizing profiles for Orichalcum tier applications requiring Guaranteed QoS class.
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
| `G-small` | 25m | 256Mi | Small fixed-footprint workloads |
| `G-medium` | 50m | 512Mi | Medium fixed-footprint workloads |
| `G-large` | 100m | 1Gi | Larger fixed-footprint workloads |
| `G-xlarge` | 200m | 2Gi | High-memory fixed-footprint workloads |
| `G-2xlarge` | 500m | 4Gi | Very large fixed-footprint workloads |

## Implementation Status

✅ **FULLY IMPLEMENTED** (as of 2026-03-03)

Current G-sizing profiles are implemented by the per-container v2 sizing policy.

**Policy Location**: `apps/00-infra/kyverno/base/policies/sizing-v2-mutate.yaml`

The executable policy is authoritative for exact values; do not copy older v1
global-label examples.

**Non-persistent admission test:**
```bash
# Ask the API server/Kyverno to admit a G-large pod without creating it.
kubectl create --dry-run=server -o yaml -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: test-g-large
  labels:
    vixens.io/sizing.test: G-large
spec:
  containers:
  - name: test
    image: nginx:alpine
EOF
```

Inspect the returned `resources:` block and confirm requests equal limits. The dry-run must not be replaced by a persistent test object merely to validate this policy.

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
        vixens.io/sizing.app: "G-small"  # key suffix must match the container name
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

- **Policy Name:** `sizing-v2-mutate`
- **Location:** `apps/00-infra/kyverno/base/policies/sizing-v2-mutate.yaml`
- **Label shape:** `vixens.io/sizing.<container-name>: G-<size>`
- **Guaranteed profiles:** G-nano through G-2xlarge as admitted by the current policy
- Other B/SB/V modes use the same per-container label namespace with different burst semantics.

## Migration Guide

To migrate an app to Guaranteed QoS:

1. **Verify app needs Orichalcum tier** - Guaranteed is only required for Orichalcum
2. **Select appropriate size** - Use the smallest size that meets requirements
3. **Add sizing label:**
   ```yaml
   labels:
     vixens.io/sizing.<container-name>: "G-small"
   ```
4. **Merge through the normal GitOps workflow and verify the resulting pod:**
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
1. Label key matches the real container name, e.g. `vixens.io/sizing.app: G-small`.
2. Kyverno is running: `kubectl get pods -n kyverno`.
3. Current policy is active: `kubectl get clusterpolicy sizing-v2-mutate`.
4. Compare the admitted resources with the executable policy rather than a stale copied table.

## See Also

- [ADR-022: 7-Tier Goldification System](../adr/022-7-tier-goldification-system.md)
- [Sizing Migration Guide](../guides/sizing-migration.md) — historical compatibility stub
- [Sizing Standards](./RESOURCE_STANDARDS.md) - Standard Burstable profiles
- [Kyverno Policy Source](../../apps/00-infra/kyverno/base/policies/sizing-v2-mutate.yaml) - Implementation

## Changelog

- **2026-10-06**: align G profiles, per-container label shape and policy source with sizing-v2-mutate
- **2026-09-26**: replaced persistent test instructions with server-side dry-run/GitOps verification
- **2026-03-03**: G-sizing fully implemented in Kyverno policy
- **2024-02-24**: Initial documentation created
