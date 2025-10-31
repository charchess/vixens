# ADR-002: ArgoCD pour GitOps

## Statut
✅ Accepté

## Contexte

Besoin d'un outil GitOps pour gérer déclarativement l'infrastructure Kubernetes :
- Déploiement automatique depuis Git
- Drift detection et auto-heal
- Multi-cluster support (dev/test/staging/prod)
- Helm + Kustomize support

### Alternatives Évaluées

1. **Flux CD**
   - ✅ CNCF graduated project
   - ✅ GitOps Toolkit modulaire
   - ✅ Multi-tenant natif
   - ❌ Pas d'UI (CLI uniquement)
   - ❌ Courbe d'apprentissage (architecture complexe)

2. **ArgoCD**
   - ✅ UI riche et intuitive
   - ✅ RBAC granulaire
   - ✅ App-of-Apps pattern
   - ✅ Multi-cluster natif
   - ✅ Helm + Kustomize support
   - ❌ Monolithe (vs Flux modulaire)

3. **Rancher Fleet**
   - ✅ Simple, focus multi-cluster
   - ❌ Écosystème Rancher requis
   - ❌ Moins mature qu'Argo/Flux

4. **Kubectl + CI/CD**
   - ✅ Simple, pas d'overhead
   - ❌ Pas de drift detection
   - ❌ Pas d'UI
   - ❌ Gestion manuelle rollback

## Décision

**Adopter ArgoCD avec pattern App-of-Apps**

### Justifications

1. **UI pour l'Apprentissage** : Visualisation du cluster, statut sync, logs en temps réel
2. **App-of-Apps** : Architecture modulaire (1 root app → N apps enfants)
3. **Multi-Cluster** : 1 ArgoCD par cluster (autonomie) avec possibilité central future
4. **Kustomize Overlays** : Base + overlays par environnement (dev/test/prod)
5. **RBAC** : Authentification Authelia future + RBAC ArgoCD
6. **Helm Support** : Déploiement charts officiels (Traefik, MetalLB, cert-manager)

## Conséquences

### Positives
- ✅ **Visibilité** : UI montre état complet du cluster
- ✅ **GitOps pur** : Git = source of truth, `git push` = déploiement
- ✅ **Auto-heal** : Drift corrigé automatiquement (si syncPolicy.selfHeal: true)
- ✅ **Rollback simple** : Git revert = rollback infra
- ✅ **Validation pré-déploiement** : Dry-run, diff avant sync

### Négatives
- ⚠️ **Single Point of Failure** : ArgoCD down = pas de sync (mais apps continuent de tourner)
  - **Mitigation** : Déployer ArgoCD en HA (3 replicas)
- ⚠️ **Complexité initiale** : Structure App-of-Apps, kustomize overlays
  - **Mitigation** : Documentation détaillée, exemples
- ⚠️ **Secret Management** : Secrets en clair dans Git (si pas SOPS)
  - **Mitigation** : Intégrer SOPS Phase 2 (bas priorité)

## Architecture Choisie

### Pattern App-of-Apps

```
Root App (clusters/dev/root-app.yaml)
  └─> apps/metallb/overlays/dev/
  └─> apps/traefik/overlays/dev/
  └─> apps/cert-manager/overlays/dev/
  └─> apps/synology-csi/overlays/dev/
  └─> ...
```

### Structure Git

```
argocd/
  base/
    - argocd-install.yaml (Helm Application)
    - root-app.yaml (App-of-Apps)
  overlays/
    dev/
      - kustomization.yaml (patches env-specific)
    test/
    prod/

apps/
  metallb/
    base/
      - helm-release.yaml
      - ipaddresspool.yaml
    overlays/
      dev/
        - ipaddresspool-dev.yaml (patch pour VLAN 208)
```

### Workflow Promotion

```
Branch dev → Cluster Dev (auto-sync)
  │
  ├─> PR dev → test
  │   └─> Review → Merge → Cluster Test sync
  │
  ├─> PR test → staging
  │   └─> Review → Merge → Cluster Staging sync
  │
  └─> PR staging → main
      └─> Review → Merge → Cluster Prod sync
```

## Références

- [ArgoCD Documentation](https://argo-cd.readthedocs.io/)
- [App-of-Apps Pattern](https://argo-cd.readthedocs.io/en/stable/operator-manual/cluster-bootstrapping/)
- [Kustomize Integration](https://argo-cd.readthedocs.io/en/stable/user-guide/kustomize/)

## Notes d'Implémentation

**Déploiement** : Via Terraform Helm provider (bootstrap initial), puis auto-géré via GitOps

**Version** : v2.11+ (support Kustomize components)

**Configuration** :
```yaml
spec:
  project: vixens
  source:
    repoURL: https://github.com/charchess/vixens
    targetRevision: dev  # ou test, staging, main
    path: apps/metallb/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: metallb-system
  syncPolicy:
    automated:
      prune: true       # Supprimer ressources supprimées du Git
      selfHeal: true    # Corriger drift automatiquement
    syncOptions:
      - CreateNamespace=true
```

---

**Date** : 2025-10-30
**Auteur** : Infrastructure Team
**Révisé** : N/A
