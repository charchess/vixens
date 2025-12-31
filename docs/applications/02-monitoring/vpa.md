# Vertical Pod Autoscaler (VPA)

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v1.5.0  |
| Prod          | [x]     | [x]       | [ ]   | v1.5.0  |

## Architecture
**Type :** Infrastructure (Static Manifests)
**Ref :** [ADR-010](../../adr/010-static-manifests-for-infrastructure-apps.md)

VPA est déployé via des manifestes statiques générés (hydratés) à partir du Helm Chart officiel.
- **Source :** `apps/00-infra/vpa/base/manifests.yaml`
- **Chart Upstream :** `cowboysysop/vertical-pod-autoscaler` (v11.1.1)
- **Namespace :** `vpa`

## Configuration
Les manifestes sont générés avec les ressources suivantes :
- **admissionController :**
  - Requests: 50m CPU / 100Mi RAM
  - Limits: 200m CPU / 500Mi RAM
- **recommender :**
  - Requests: 50m CPU / 250Mi RAM
  - Limits: 200m CPU / 1Gi RAM
  - *Extra Args:* min-cpu: 10m, min-memory: 128Mi (pour Goldilocks)
- **updater :**
  - Requests: 50m CPU / 100Mi RAM
  - Limits: 200m CPU / 500Mi RAM

**Tolerations :** Tous les composants ont des tolérations pour le Control Plane.

## Procédure de Mise à Jour (Upgrade)
Pour mettre à jour VPA, il faut régénérer le fichier `manifests.yaml`.

1. **Préparer les valeurs temporaires (`/tmp/vpa-values.yaml`) :**
   (Se référer aux valeurs documentées ci-dessus)

2. **Générer les manifestes :**
   ```bash
   helm repo add cowboysysop https://cowboysysop.github.io/charts
   helm repo update
   helm template vpa cowboysysop/vertical-pod-autoscaler --version <NEW_VERSION> --namespace vpa --values /tmp/vpa-values.yaml > apps/00-infra/vpa/base/manifests.yaml
   ```

## Validation

### Méthode Automatique (Command Line)
```bash
# Vérifier que les composants VPA sont en ligne
kubectl get pods -n vpa -l app.kubernetes.io/name=vpa
# Attendu: Pods recommender, updater, admission-controller en statut Running
```

### Méthode Manuelle
1. Créer un objet `VerticalPodAutoscaler` de test.
2. Vérifier la génération de recommandations : `kubectl get vpa -n <namespace>`.

## Notes Techniques
- **Namespace :** `vpa` (Migré de `monitoring` vers son propre namespace pour isolation infrastructure).
- **Dépendances :** Metrics Server.
