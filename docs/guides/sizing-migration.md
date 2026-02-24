# Migration Guide: Standardized Resource Sizing

This guide explains how to migrate to the new standardized resource sizing system using Kyverno policies.

## Overview

The new system uses **Kyverno mutation policies** to automatically apply resource sizing based on labels:

- **Phase 1: Audit** (Current) - Identify apps missing sizing labels
- **Phase 2: Labeling** - Add labels to critical apps manually
- **Phase 3: Migration** - Enable automatic mutation

## Phase 1: Audit (Active Now)

The `sizing-audit` policy is currently in **Audit** mode. It will not block deployments but will generate reports.

### Check Audit Results

```bash
# Generate audit report
./scripts/sizing-audit-report.sh

# Or manually check policy reports
kubectl get policyreports -n kyverno -o yaml | grep -A10 sizing-audit
```

This shows which pods are missing the `vixens.io/sizing` label.

## Phase 2: Label Critical Apps

Before enabling automatic mutation, label your critical apps:

### In overlay kustomization.yaml:

```yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

metadata:
  labels:
    vixens.io/sizing: medium  # Choose: micro/small/medium/large/xlarge

resources:
  - ../../base
  - ingress.yaml

components:
  - ../../../../_shared/components/default
  - ../../../../_shared/components/priority/medium
```

### Sizing Reference (from RESOURCE_STANDARDS.md):

| Sizing | CPU (Req/Lim) | Memory (Req/Lim) | Typical Use |
|--------|---------------|------------------|-------------|
| **micro** | 10m/100m | 64Mi/128Mi | Sidecars, exporters |
| **small** | 50m/500m | 256Mi/512Mi | Optimized apps (Go/Rust) |
| **medium** | 200m/1000m | 512Mi/1Gi | Standard web apps |
| **large** | 1000m/2000m | 2Gi/4Gi | Databases, heavy apps |
| **xlarge** | 2000m/4000m | 4Gi/8Gi | AI processing, indexing |

## Phase 3: Enable Mutation

When you're confident all critical apps are labeled:

```bash
# Enable automatic sizing mutation
kubectl patch clusterpolicy sizing-mutate \
  -p '{"spec":{"validationFailureAction":"Enforce"}}'

# Verify it's enabled
kubectl get clusterpolicy sizing-mutate -o jsonpath='{.spec.validationFailureAction}'
```

## Rollback

If something goes wrong:

```bash
# Disable mutation (back to audit)
kubectl patch clusterpolicy sizing-mutate \
  -p '{"spec":{"validationFailureAction":"Audit"}}'

# Apps will keep their current resources, no automatic changes
```

## What Happens When Mutation is Enabled?

1. **Pod creation**: Kyverno intercepts the pod
2. **Label check**: 
   - Label present → Apply corresponding sizing
   - No label → Apply **micro** sizing (default)
3. **Resources injected**: CPU/memory requests and limits added
4. **Pod admitted**: Deployment proceeds with mutated pod

## FAQ

**Q: What if my app already has resources defined?**
A: Kyverno will overwrite them. Label your app appropriately first.

**Q: Can I override sizing in an emergency?**
A: Yes, add the label to the overlay and redeploy. Or temporarily disable the policy.

**Q: Will this affect running pods?**
A: No, Kyverno only mutates on creation/update. Restart pods to apply changes.

**Q: What about the old sizing/* Kustomize components?**
A: They will be removed once migration is complete.

## Timeline Recommendation

- **Week 1**: Run audit, identify critical apps
- **Week 2**: Label critical apps (large/xlarge)
- **Week 3**: Label standard apps (small/medium)
- **Week 4**: Enable mutation, monitor closely

## Monitoring

```bash
# Check policy status
kubectl get clusterpolicy sizing-audit sizing-mutate

# View recent mutations
kubectl get events --field-selector reason=PolicyApplied

# Check policy reports
kubectl get policyreports -n kyverno
```
