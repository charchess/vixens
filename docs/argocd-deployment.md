# ArgoCD Deployment - Sprint 4

**Date**: 2025-11-01
**Status**: ✅ DEPLOYED
**Cluster**: vixens-dev
**Sprint**: 4 - GitOps Bootstrap

## Overview

ArgoCD deployed via Terraform Helm provider with comprehensive control-plane tolerations for full HA cluster (3 control planes, 0 workers).

## Deployment Details

| Component | Value |
|-----------|-------|
| **Chart** | argo-cd v7.7.7 |
| **App Version** | v2.13.1 |
| **Namespace** | argocd |
| **Deployment Method** | Terraform Helm provider |
| **Deployment Time** | 1m5s |
| **Mode** | HTTP (--insecure) |

## Architecture

### Pods Distribution

| Pod | Node | Status | Component |
|-----|------|--------|-----------|
| argocd-server | opale | Running | API Server & UI |
| argocd-repo-server | obsy | Running | Git Repository Server |
| argocd-application-controller | onyx | Running | Application Controller (StatefulSet) |
| argocd-redis | obsy | Running | Cache & State Storage |
| argocd-applicationset-controller | opale | Running | ApplicationSet CRD Controller |
| argocd-notifications-controller | opale | Running | Notifications Manager |

**Job:** `argocd-redis-secret-init` - Completed (14s) - auto-cleaned

### Services

| Service | Type | ClusterIP | Ports |
|---------|------|-----------|-------|
| argocd-server | ClusterIP | 10.103.179.178 | 80/TCP, 443/TCP |
| argocd-repo-server | ClusterIP | 10.106.51.66 | 8081/TCP |
| argocd-redis | ClusterIP | 10.111.204.32 | 6379/TCP |
| argocd-applicationset-controller | ClusterIP | 10.110.169.205 | 7000/TCP |

## Control-Plane Tolerations

All components configured with control-plane tolerations:

```yaml
tolerations:
  - key: node-role.kubernetes.io/control-plane
    operator: Exists
    effect: NoSchedule
```

**Components with tolerations:**
1. ✅ server
2. ✅ repoServer
3. ✅ controller
4. ✅ redis
5. ✅ applicationSet
6. ✅ notifications
7. ✅ redisSecretInit (Job) - **Critical for full CP clusters!**

## Access & Credentials

### Initial Admin Credentials

```bash
Username: admin
Password: gvB3Cq-fjYTC8UPR
Secret: argocd-initial-admin-secret (namespace: argocd)
```

**Security Note:** Change admin password in production environments.

### Access Methods

#### 1. Port-Forward (Development)

```bash
export KUBECONFIG=/root/vixens/terraform/environments/dev/kubeconfig-dev
kubectl port-forward -n argocd svc/argocd-server 8080:80
```

Access: `http://localhost:8080`

#### 2. ArgoCD CLI

```bash
argocd login localhost:8080 --username admin --password gvB3Cq-fjYTC8UPR --insecure
argocd cluster list
argocd app list
```

#### 3. External Access (Future - Sprint 6)

Via Traefik Ingress with TLS termination:
- URL: `https://argocd.vixens.lab` (VLAN 208)
- Certificate: cert-manager (Let's Encrypt or self-signed)

## Configuration

### Terraform Configuration

Location: `terraform/environments/dev/argocd.tf`

Key configuration highlights:

```hcl
resource "helm_release" "argocd" {
  name             = "argocd"
  repository       = "https://argoproj.github.io/argo-helm"
  chart            = "argo-cd"
  version          = "7.7.7"
  namespace        = "argocd"
  create_namespace = true

  wait          = true
  wait_for_jobs = true
  timeout       = 600

  values = [yamlencode({
    server = {
      extraArgs   = ["--insecure"]
      tolerations = [...]
    }
    redisSecretInit = {
      tolerations = [...]  # Critical!
    }
    configs = {
      params = {
        "server.insecure" = true
      }
    }
  })]

  depends_on = [helm_release.cilium]
}
```

### Helm Values Applied

- **Insecure mode**: HTTP only (TLS via Traefik later)
- **Dex disabled**: SSO configuration deferred to future sprint
- **Full control-plane tolerations**: All 7 components + Job

## Validation

### Health Checks

```bash
# Check all pods
kubectl get pods -n argocd

# Check services
kubectl get svc -n argocd

# Test health endpoint
kubectl port-forward -n argocd svc/argocd-server 8080:80 &
curl http://localhost:8080/healthz
# Expected: "ok"
```

### Helm Release Status

```bash
helm list -n argocd
# NAME    NAMESPACE  REVISION  STATUS    CHART          APP VERSION
# argocd  argocd     1         deployed  argo-cd-7.7.7  v2.13.1
```

### Events & Logs

```bash
# Recent events
kubectl get events -n argocd --sort-by='.lastTimestamp'

# Server logs
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-server --tail=20

# Redis logs
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-redis --tail=20
```

## Troubleshooting

### Issue 1: Redis Secret Init Job Pending

**Symptom:** Job `argocd-redis-secret-init` stuck in Pending state

**Error:**
```
FailedScheduling: 0/3 nodes available: 3 node(s) had untolerated taint {node-role.kubernetes.io/control-plane}
```

**Solution:** Add `redisSecretInit.tolerations` to Helm values

```yaml
redisSecretInit:
  tolerations:
    - key: node-role.kubernetes.io/control-plane
      operator: Exists
      effect: NoSchedule
```

**Resolution Time:** Job completes in ~14s after fix

### Issue 2: Redis Connection Warnings

**Symptom:** Logs show "Failed to resync revoked tokens: operation not permitted"

**Root Cause:** Transient Redis connection during startup

**Status:** Self-resolving, no action required

## Integration Points

### Current (Sprint 4)

- ✅ Cilium CNI: Network policy enforcement
- ✅ Kubernetes API: Cluster management
- ✅ Talos Linux: Immutable OS platform

### Future Sprints

- **Sprint 5**: MetalLB (LoadBalancer services)
- **Sprint 6**: Traefik (Ingress with TLS termination)
- **Sprint 7**: cert-manager (TLS certificates)
- **Sprint 8**: Synology CSI (Persistent storage)

## GitOps Workflow (Sprint 4 Remaining Tasks)

### Task 4.2: ArgoCD Structure

Create repository structure:
```
argocd/
├── base/
│   ├── argocd-install.yaml  # ArgoCD self-management
│   └── root-app.yaml         # App-of-Apps pattern
└── overlays/
    └── dev/
        └── kustomization.yaml
```

### Task 4.3: Git Push

Commit structure to `dev` branch, merge to `main` after validation.

### Task 4.4: Bootstrap Root App

```bash
kubectl apply -f argocd/overlays/dev/root-app.yaml
argocd app sync root-app
```

### Task 4.5: Validation

- ArgoCD manages itself via Git
- Auto-sync enabled
- Test: Git change → auto-deployment

## Lessons Learned

### Pattern: Full Control-Plane Tolerations

**Finding:** Jobs and init containers often missed in toleration configurations

**Solution:** Systematically check ALL workload types in Helm chart:
- Deployments
- StatefulSets
- DaemonSets
- Jobs ⚠️
- Init Containers ⚠️

**Example Components Requiring Tolerations:**
- `redisSecretInit` (Job)
- Future: `cert-manager` webhook jobs
- Future: `metallb` speaker DaemonSet

### Helm Chart Investigation

When deploying third-party Helm charts on full control-plane clusters:

1. Search chart values.yaml for toleration keys
2. Review chart templates for Job/CronJob resources
3. Test deployment, monitor events for scheduling failures
4. Add missing tolerations incrementally

**Tools:**
```bash
# Download chart values
helm show values argo/argo-cd > argocd-values.yaml

# Search for toleration keys
grep -n "toleration" argocd-values.yaml
```

## Next Steps

**Immediate (Sprint 4):**
- [ ] Task 4.2: Create ArgoCD structure
- [ ] Task 4.3: Push to Git (dev branch)
- [ ] Task 4.4: Apply root-app
- [ ] Task 4.5: Validate self-management

**Future (Sprint 5+):**
- [ ] Sprint 5: Deploy MetalLB (with tolerations!)
- [ ] Sprint 6: Deploy Traefik + ArgoCD Ingress
- [ ] Sprint 7: TLS certificates via cert-manager
- [ ] Sprint 8: Configure Synology CSI for persistence

## References

- [ArgoCD Helm Chart](https://github.com/argoproj/argo-helm/tree/main/charts/argo-cd)
- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [Talos + ArgoCD Guide](https://www.talos.dev/v1.11/kubernetes-guides/configuration/argocd/)
- Vixens ROADMAP: `docs/ROADMAP.md`
- Validation Report: `docs/validation-1cp-2w.md`
