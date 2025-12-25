# ArgoCD Sync Waves Configuration

## Objectif

Améliorer la vitesse et la fiabilité du déploiement en ordonnançant correctement les applications avec des sync waves.

## Problème Actuel

Après redéploiement du cluster dev (25/12/2024):
- **1ère vague (t+0):** ~20 applications démarrées simultanément
- **2ème vague (t+1h):** ~30 applications supplémentaires
- **Problèmes identifiés:**
  - Applications dépendantes démarrées avant leurs dépendances (ex: apps avec PostgreSQL)
  - InfisicalSecrets avec erreurs de syntaxe bloquant les pods
  - CRDs non synchronisées avant les opérateurs

## Stratégie de Sync Waves

### Wave -5: CRDs (Custom Resource Definitions)
**Objectif:** Installer les CRDs avant tout le reste

Applications:
- `cloudnative-pg-crds` (actuellement wave 2 ✅)

### Wave -4: Operators
**Objectif:** Installer les opérateurs qui gèrent les CRDs

Applications:
- `infisical-operator`
- `cloudnative-pg` (actuellement wave 3 ✅)
- `cert-manager`
- `synology-csi`

### Wave -3: Secrets & Configuration
**Objectif:** Créer les secrets avant les services qui en dépendent

Applications:
- `cert-manager-secrets`
- `cert-manager-config`
- `synology-csi-secrets`
- Tous les InfisicalSecret standalone

### Wave -2: Infrastructure de Base
**Objectif:** Déployer l'infrastructure réseau et stockage

Applications:
- `cilium-lb` (LoadBalancer IPAM)
- `traefik` (Ingress controller)
- `nfs-storage`

### Wave -1: Services Partagés
**Objectif:** Déployer les services utilisés par plusieurs applications

Applications:
- `postgresql-shared` (base de données partagée)
- `redis-shared` (cache partagé)

### Wave 0: Applications (par défaut)
**Objectif:** Déployer les applications métier

Toutes les autres applications (homeassistant, authentik, netbox, etc.)

## Implémentation

### Méthode 1: Annotation dans Application
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: postgresql-shared
  annotations:
    argocd.argoproj.io/sync-wave: "-1"
```

### Méthode 2: Annotation dans les ressources Kubernetes
```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    argocd.argoproj.io/sync-wave: "0"
```

## Délais Entre Waves

ArgoCD attend que toutes les ressources d'une wave soient **Healthy** avant de passer à la suivante.

**Recommandation:**
- Pas de délai artificiel nécessaire
- ArgoCD gère automatiquement la progression
- Pour forcer un délai: utiliser un Job avec `sleep` si vraiment nécessaire

## Problèmes Spécifiques Identifiés

### CloudNativePG CRDs

**Symptôme:** Application `cloudnative-pg-crds` reste OutOfSync malgré `ServerSideApply=true`

**Cause:** Conflit de field managers entre:
1. Application ArgoCD qui applique les CRDs (wave 2)
2. Modifications ultérieures par l'opérateur ou Helm

**Impact:** Les CRDs sont fonctionnelles, juste le status ArgoCD incorrect

**Solution Testée:**
```yaml
syncOptions:
  - ServerSideApply=true  # ✅ Déjà présent
  - CreateNamespace=true  # ✅ Déjà présent
```

**Solution Alternative (si problème persiste):**
1. Supprimer l'app `cloudnative-pg-crds`
2. Laisser le chart Helm `cloudnative-pg` gérer les CRDs avec `crds.create: true`

**Décision:** Garder la configuration actuelle (CRDs séparées), c'est une best practice.

### InfisicalSecret API Migration

**Symptôme:** Certaines applications ne démarrent pas (homepage, prowlarr, sonarr)

**Cause:** InfisicalSecrets utilisent l'ancienne API sans `credentialsRef`

**Correction Appliquée (25/12/2024):**
```yaml
# AVANT (❌ invalide)
spec:
  authentication:
    universalAuth:
      secretsScope:
        secretName: infisical-universal-auth  # ❌ Mauvais emplacement
        secretNamespace: argocd

# APRÈS (✅ correct)
spec:
  authentication:
    universalAuth:
      credentialsRef:  # ✅ Nouvelle section requise
        secretName: infisical-universal-auth
        secretNamespace: argocd
      secretsScope:
        projectSlug: vixens
        envSlug: dev
        secretsPath: "/apps/..."
```

**Applications corrigées:**
- apps/70-tools/homepage/overlays/dev/ ✅
- apps/70-tools/homepage/overlays/prod/ ✅
- apps/20-media/prowlarr/base/ ✅
- apps/20-media/sonarr/base/ ✅

## Plan d'Action

### Phase 1: Vérification (Immédiate)
- [ ] Lister toutes les applications ArgoCD
- [ ] Identifier les dépendances entre applications
- [ ] Vérifier les sync-waves actuelles

### Phase 2: Configuration (Sprint actuel)
- [ ] Ajouter sync-wave annotations aux Applications manquantes
- [ ] Tester sur cluster dev
- [ ] Documenter les dépendances dans chaque app/

### Phase 3: Validation (Avant passage en test)
- [ ] Destroy/recreate cluster dev pour valider l'ordre
- [ ] Mesurer le temps total de déploiement
- [ ] Vérifier qu'aucune application ne démarre avant ses dépendances

### Phase 4: Propagation (Sprint suivant)
- [ ] Appliquer les mêmes waves sur test/staging/prod
- [ ] Créer des scripts de validation automatique

## Métriques de Succès

**Avant (Déploiement actuel):**
- Temps total: ~2h (2 vagues)
- Apps en erreur au démarrage: 3-5 (InfisicalSecret, dépendances)
- Interventions manuelles: Plusieurs sync forcées

**Cible (Avec sync waves):**
- Temps total: ~30-45 minutes (déploiement séquentiel optimisé)
- Apps en erreur: 0 (dépendances respectées)
- Interventions manuelles: 0 (auto-sync fonctionne)

## Références

- [ArgoCD Sync Waves](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-waves/)
- [ArgoCD Sync Options](https://argo-cd.readthedocs.io/en/stable/user-guide/sync-options/)
- [CloudNativePG Best Practices](https://cloudnative-pg.io/documentation/current/installation_upgrade/)
