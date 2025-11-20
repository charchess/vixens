# Infisical Multi-Environment Setup Procedure

## Phase 1: Infisical UI Configuration (MANUEL)

Cette phase doit être effectuée manuellement dans l'interface Infisical avant de déployer les configurations GitOps.

### Prérequis

- Accès à Infisical UI: http://192.168.111.69:8085
- Projet existant: `vixens`
- Environnements existants: dev, test, staging, prod

### 1.1 Créer les dossiers d'isolation par environnement

Pour **chaque environnement** (test, staging, prod), créer les dossiers suivants:

**Environment: test**
1. Naviguer vers Project `vixens` → Environment `test`
2. Créer dossier `/cert-manager`
3. Créer dossier `/synology-csi`

**Environment: staging**
1. Naviguer vers Project `vixens` → Environment `staging`
2. Créer dossier `/cert-manager`
3. Créer dossier `/synology-csi`

**Environment: prod**
1. Naviguer vers Project `vixens` → Environment `prod`
2. Créer dossier `/cert-manager`
3. Créer dossier `/synology-csi`

### 1.2 Peupler les secrets dans chaque environnement

**Test Environment:**
1. Dans `/cert-manager/`:
   - Créer secret `api-token` avec la valeur du token Gandi API (même que dev ou spécifique)

2. Dans `/synology-csi/`:
   - Créer secret `client-info.yaml` avec le contenu:
     ```yaml
     ---
     synology-csi-client-info: |
       {
         "host": "192.168.111.69",
         "port": 5000,
         "username": "admin",
         "password": "VOTRE_MOT_DE_PASSE",
         "sslVerify": false,
         "sessionName": "test-cluster"
       }
     ```

**Staging Environment:**
1. Dans `/cert-manager/`:
   - Créer secret `api-token` avec la valeur du token Gandi API

2. Dans `/synology-csi/`:
   - Créer secret `client-info.yaml` (même structure, `sessionName`: `staging-cluster`)

**Prod Environment:**
1. Dans `/cert-manager/`:
   - Créer secret `api-token` avec la valeur du token Gandi API (PROD uniquement)

2. Dans `/synology-csi/`:
   - Créer secret `client-info.yaml` (même structure, `sessionName`: `prod-cluster`)

### 1.3 Créer les Machine Identities

Pour chaque environnement, créer une Machine Identity dédiée avec Universal Auth:

**Test Environment Machine Identity:**
1. Naviguer vers Project Settings → Machine Identities
2. Cliquer "Create Identity"
3. Nom: `vixens-test-k8s-operator`
4. Type: `Universal Auth`
5. Access:
   - Project: `vixens`
   - Environment: `test` (SEULEMENT)
   - Paths: `/cert-manager`, `/synology-csi` (ou `/*` pour tout l'environnement)
6. Générer Client ID et Client Secret
7. **DOCUMENTER** les credentials:
   ```
   vixens-test-k8s-operator:
     clientId: [COPIER ICI]
     clientSecret: [COPIER ICI]
   ```

**Staging Environment Machine Identity:**
1. Répéter les étapes ci-dessus
2. Nom: `vixens-staging-k8s-operator`
3. Environment: `staging` (SEULEMENT)
4. **DOCUMENTER** les credentials:
   ```
   vixens-staging-k8s-operator:
     clientId: [COPIER ICI]
     clientSecret: [COPIER ICI]
   ```

**Prod Environment Machine Identity:**
1. Répéter les étapes ci-dessus
2. Nom: `vixens-prod-k8s-operator`
3. Environment: `prod` (SEULEMENT)
4. **DOCUMENTER** les credentials:
   ```
   vixens-prod-k8s-operator:
     clientId: [COPIER ICI]
     clientSecret: [COPIER ICI]
   ```

### Validation Phase 1

Avant de passer à la Phase 2 (GitOps), vérifier:

- [ ] 3 environnements configurés (test, staging, prod)
- [ ] 6 dossiers créés (2 par environnement: cert-manager, synology-csi)
- [ ] 6 secrets dans chaque environnement (api-token × 3, client-info.yaml × 3)
- [ ] 3 Machine Identities créées avec clientId/clientSecret documentés
- [ ] Isolation vérifiée: test identity n'a PAS accès à staging/prod

### Notes de Sécurité

- **Ne JAMAIS** committer les clientId/clientSecret dans Git
- Les credentials seront stockés dans les overlays Kustomize (qui peuvent être chiffrés avec SOPS ou sealed-secrets si nécessaire)
- Pour ce homelab, les credentials restent en clair dans les overlays (risque acceptable)
- En production, utiliser External Secrets Operator ou Sealed Secrets

### Prochaines Étapes

Une fois la Phase 1 terminée, procéder à la Phase 2:
- Créer les overlays GitOps avec les credentials Machine Identity
- Déployer via ArgoCD

Voir: [OpenSpec propagate-infisical-multi-env](../../openspec/changes/propagate-infisical-multi-env/tasks.md)
