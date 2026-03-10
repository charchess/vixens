# OpenClaw

## Deployment Information
| Environment | Deployed | Configured | Tested | Version |
|-------------|----------|-----------|-------|---------|
| Dev         | [x]      | [x]       | [x]   | latest  |
| Prod        | [x]      | [x]       | [x]   | latest  |

## Validation
**URL:** https://openclaw.truxonline.com

### Automatic Validation (CLI)
```bash
# Check pod status
kubectl --kubeconfig=.secrets/prod/kubeconfig-prod get pods -n services -l app=openclaw

# Check ingress
kubectl --kubeconfig=.secrets/prod/kubeconfig-prod get ingress -n services openclaw-ingress

# Test connectivity
curl https://openclaw.truxonline.com/
```

### Manual Validation
1. Open URL in browser.
2. Verify the OpenClaw UI loads correctly.

## Technical Notes
- **Namespace:** services
- **Category:** 60-services
- **Dependencies:**
    - MinIO (S3 backup)
    - Infisical (secrets management)
- **Specifics:**
    - Gateway mode: local
    - Binding: lan
    - Control UI: allowInsecureAuth enabled (open access)
    - Data persisted to PVC and synced to MinIO
- **Security Note:**
    - Currently open access (no authentication)
    - TODO: Add Authentik middleware via Traefik


## Known Issues

### ArgoCD OutOfSync (PVC volumeName)

**Status**: ⚠️ Permanent OutOfSync (Accepté)

**Description**:
ArgoCD affiche `OutOfSync` pour l'application openclaw en raison d'une différence détectée sur `spec.volumeName` du PVC `openclaw-data`.

**Détails techniques**:
- Manifest Git: `volumeName` absent (correct - géré automatiquement par Kubernetes)
- Cluster: `volumeName = pvc-075015f2-b8e6-426a-93ed-f1ee40a74c5b` (set par K8s lors du bind)
- ArgoCD détecte la différence et tente de patch, mais Kubernetes rejette (PVC spec immutable après bind)

**Résolution**:
- ✅ `ignoreDifferences` configuré dans `argocd/overlays/prod/apps/openclaw.yaml` (PR #1974)
- ✅ Application fonctionne parfaitement (Status: Healthy, Pods: Running)
- ⚠️ OutOfSync persiste (limitation ArgoCD connue: https://github.com/argoproj/argo-cd/issues/2913)

**Impact**: ❌ Aucun (cosmétique uniquement)

**Action**: Ignorer le warning OutOfSync pour openclaw. Monitorer `Health` status uniquement.

**Validation**:
```bash
# Application opérationnelle
kubectl -n services get pods -l app=openclaw
# Expected: Running (2/2)

# ignoreDifferences configuré
kubectl -n argocd get application openclaw -o jsonpath='{.spec.ignoreDifferences}'
# Expected: [{"jsonPointers":["/spec/volumeName"],"kind":"PersistentVolumeClaim","name":"openclaw-data"}]
```

**Référence**: Task vixens-wqrw