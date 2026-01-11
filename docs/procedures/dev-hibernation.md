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

## Hibernation Methods

### Method 1: Scale to Zero (Preferred)

Best for applications that can be quickly restarted without data loss.

```bash
# Hibernate
kubectl scale deployment <app-name> -n <namespace> --replicas=0

# Wake up
kubectl scale deployment <app-name> -n <namespace> --replicas=1
```

**Advantages:**
- Quick to hibernate/wake
- No ArgoCD sync issues
- PVC data preserved

**Use For:**
- Stateless applications
- Applications with persistent storage
- Quick testing iterations

### Method 2: ArgoCD Suspend

Best for complex applications or when you want ArgoCD to stop managing temporarily.

```bash
# Suspend ArgoCD sync
kubectl patch application <app-name> -n argocd \
  --type merge \
  -p '{"spec":{"syncPolicy":null}}'

# Then scale down
kubectl scale deployment <app-name> -n <namespace> --replicas=0

# Wake up: re-enable sync
kubectl patch application <app-name> -n argocd \
  --type merge \
  -p '{"spec":{"syncPolicy":{"automated":{"prune":true,"selfHeal":true}}}}'
```

**Advantages:**
- Prevents ArgoCD from auto-healing
- Useful for complex multi-resource apps

**Use For:**
- Complex applications (multiple deployments, statefulsets)
- When testing manual changes
- Long-term hibernation

### Method 3: Git-Based Hibernation (Most GitOps-Compliant)

Best for long-term hibernation or when coordinating across team.

```yaml
# In apps/<category>/<app>/overlays/dev/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

# Add replica override
patches:
  - patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: <app-name>
      spec:
        replicas: 0
```

**Advantages:**
- Git-tracked (visible in version control)
- Survives cluster resets
- Clear intent in repository

**Use For:**
- Long-term hibernation
- Planned resource management
- Clear documentation of hibernated apps

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

## Quick Reference Commands (Recommended)

Utilisez les commandes `just` pour une gestion GitOps simplifiée :

```bash
# Mettre une application en hibernation (replicas=0)
just hibernate <app-name>

# Réactiver une application (replicas=1)
just unhibernate <app-name>

# Lister toutes les applications hibernées via Git
just hibernated
```

### Manual Commands (Fallback)

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

## Automation Logic

Les commandes `just` effectuent les actions suivantes :
1. Détection du répertoire de l'application dans `apps/`.
2. Application d'un patch Kustomize `replicas: 0` (ou 1) dans `overlays/dev/kustomization.yaml`.
3. Activation automatique dans ArgoCD (décommentage dans `argocd/overlays/dev/kustomization.yaml`).
4. Commit GitOps automatique.
5. Push sur la branche principale.


See Beads task: `vixens-xxxx` for automation implementation.

---

## Important Notes

### Data Persistence
- Hibernating (scale to 0) does NOT delete PVCs
- Data is preserved when app is woken up
- PVC cleanup is separate operation (requires manual delete)

### ArgoCD Behavior
- ArgoCD may auto-heal scaled-down apps if sync policy is automated
- Use Method 2 (suspend sync) if ArgoCD fights hibernation
- Method 3 (git-based) is GitOps-compliant and prevents auto-heal

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

**Last Updated:** 2026-01-11
