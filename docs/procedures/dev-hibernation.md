# Dev Environment Hibernation

## Overview

Most applications in the dev environment should be put into hibernation mode when not actively being tested or developed. This conserves cluster resources and improves overall stability.

## Why Hibernate?

### Resource Conservation
- Dev cluster has limited resources
- Multiple applications consume CPU/memory even when idle
- Hibernated apps free resources for active testing

### Operational Benefits
- Easier to identify actively developed applications
- Cleaner resource monitoring
- Faster troubleshooting (fewer running pods)
- Lower power consumption

### When to Keep Apps Active
- Infrastructure components (ArgoCD, Traefik, Cilium, etc.)
- Monitoring stack (Prometheus, Grafana, Loki)
- Storage components (Synology CSI)
- Services under active development/testing

---

## Standard Hibernation Method: Git-Based Replicas Patch

**The official and ONLY method for hibernating applications in dev environment.**

### How It Works

Applications are hibernated by patching their deployment replicas to 0 in the environment overlay:

```yaml
# In apps/<category>/<app>/overlays/dev/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

patches:
  - patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: <app-name>
      spec:
        replicas: 0  # Hibernated
```

**Applications ALWAYS remain active in ArgoCD** (`argocd/overlays/dev/kustomization.yaml`). The hibernation state is managed entirely in the application's own overlay.

### Advantages

- **GitOps-compliant**: All state in Git, no manual kubectl commands
- **Visible in ArgoCD**: App appears as "Synced" with 0 replicas (not disabled)
- **Environment-specific**: Dev can be hibernated while prod stays active
- **Clear intent**: Explicit replicas patch shows hibernation state
- **Survives cluster resets**: Hibernation state persists through infrastructure changes
- **No sync conflicts**: ArgoCD recognizes this as the desired state

### Example: HomeAssistant (Already Hibernated)

```yaml
# apps/10-home/homeassistant/overlays/dev/kustomization.yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: homeassistant
resources:
  - ../../base
  - ingress.yaml

patches:
  - patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: homeassistant
      spec:
        replicas: 0  # ← Hibernated in dev
```

### Wake Up Process

To wake up a hibernated application, simply change the replicas patch:

```yaml
patches:
  - patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: <app-name>
      spec:
        replicas: 1  # ← Active
```

Or remove the patch entirely to use the base value.

### Step-by-Step: Hibernating a New Application

1. **Navigate to the app's dev overlay:**
   ```bash
   cd apps/<category>/<app-name>/overlays/dev
   ```

2. **Edit or create `kustomization.yaml`:**
   ```yaml
   apiVersion: kustomize.config.k8s.io/v1beta1
   kind: Kustomization
   namespace: <namespace>
   
   resources:
     - ../../base
     # ... other resources
   
   patches:
     - patch: |-
         apiVersion: apps/v1
         kind: Deployment
         metadata:
           name: <deployment-name>
         spec:
           replicas: 0
   ```

3. **Commit and push:**
   ```bash
   git add .
   git commit -m "feat(dev): hibernate <app-name> - set replicas to 0"
   git push
   ```

4. **Verify in ArgoCD:**
   - App should show as "Synced" with 0/0 pods
   - No "OutOfSync" warning

---

## Workflow

### Before Testing Session

1. **Identify Apps to Wake**
   ```bash
   # List hibernated apps
   kubectl get deployments -A | grep "0/0"
   ```

2. **Wake Required Apps**
   ```bash
   kubectl scale deployment <app-name> -n <namespace> --replicas=1
   ```

3. **Wait for Ready**
   ```bash
   kubectl wait --for=condition=ready pod -l app=<app-name> -n <namespace> --timeout=300s
   ```

4. **Begin Testing**

### After Testing Session

1. **Scale Down Non-Essential Apps**
   ```bash
   # Scale down apps not needed for next session
   kubectl scale deployment <app-name> -n <namespace> --replicas=0
   ```

2. **Verify Hibernation**
   ```bash
   kubectl get pods -n <namespace>
   ```

3. **Document Active Apps**
   - Update STATUS.md if long-term changes
   - Note in Beads task if temporary

---

## Recommended Hibernation States

### Always Active (Infrastructure)
```
argocd/*
traefik/*
cilium/*
cert-manager/*
synology-csi/*
prometheus/*
grafana/*
loki/*
```

### Hibernate When Not Testing (Applications)
```
apps/10-home/*      (except core home automation)
apps/20-media/*     (except actively used media services)
apps/60-services/*  (test applications)
apps/70-tools/*     (development tools)
apps/99-test/*      (all test applications)
```

### Case-by-Case (Databases)
```
apps/04-databases/*
  - Keep active: shared databases (postgresql-shared, redis-shared)
  - Hibernate: app-specific databases not in use
```

---

## Quick Reference Commands

### Automated Commands (Planned - See vixens-6c9j)

The following commands will simplify hibernation management once implemented:

```bash
# Hibernate an application (set replicas=0 in overlay)
just hibernate <app-name>

# Wake up an application (set replicas=1 or remove patch)
just unhibernate <app-name>

# List all hibernated applications
just hibernated
```

**Status**: Not yet implemented. See Beads task `vixens-6c9j` for automation work.

### Manual Commands (Current Method)

```bash
# List all deployments with 0 replicas (hibernated)
kubectl get deployments -A | grep "0/0"

# List all deployments with >0 replicas (active)
kubectl get deployments -A | grep -v "0/0" | grep -v "READY"

# Check resource usage (before/after hibernation)
kubectl top nodes
kubectl top pods -A
```

---

## Automation (Planned)

Future automation via `just` commands will:
1. Detect the application directory in `apps/`
2. Add/modify the replicas patch in `overlays/dev/kustomization.yaml`
3. Commit changes with descriptive message
4. Push to main branch for ArgoCD auto-sync

See Beads task `vixens-6c9j` for implementation tracking.

---

## Important Notes

### Data Persistence
- Hibernating (scale to 0) does NOT delete PVCs
- Data is preserved when app is woken up
- PVC cleanup is separate operation (requires manual delete)

### ArgoCD Behavior
- Applications with `replicas: 0` appear as "Synced" in ArgoCD (desired state)
- ArgoCD correctly recognizes the replicas patch as the intended configuration
- No auto-heal conflicts since the git state matches the cluster state
- Applications remain visible in ArgoCD UI (not disabled or removed)

### Resource Considerations
- Hibernated apps still consume PVC storage
- PVC cleanup requires separate action
- Consider using emptyDir for truly temporary apps

---

## Related Documentation

- [Application Testing](./application-testing.md)
- [Resource Management](../guides/resource-management.md)
- [ArgoCD Management](../guides/argocd-management.md)

---

**Last Updated:** 2026-01-12
