# Reloader

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version | Statut |
|---------------|---------|-----------|-------|---------|--------|
| Dev           | [x]     | [x]       | [x]   | v1.4.12 | 🏆 Elite |
| Prod          | [x]     | [x]       | [ ]   | v1.4.12 | 🏆 Elite |

## Architecture
**Type :** Infrastructure (Static Manifests)
**Ref :** [ADR-010](../../adr/010-static-manifests-for-infrastructure-apps.md)

Reloader est déployé via des manifestes statiques générés (hydratés) à partir du Helm Chart officiel.
- **Source :** `apps/00-infra/reloader/base/manifests.yaml`
- **Chart Upstream :** `stakater/reloader`

## Configuration
Les manifestes sont générés avec les valeurs par défaut, modifiées pour :
- **SecurityContext :** `runAsNonRoot: true`, `runAsUser: 65534`, `readOnlyRootFileSystem: true`
- **Resources :** Requests/Limits définis explicitement
- **Tolerations :** Ajoutées pour le Control Plane

## Procédure de Mise à Jour (Upgrade)
Pour mettre à jour Reloader, il faut régénérer le fichier `manifests.yaml`.

1. **Préparer les valeurs temporaires (`/tmp/reloader-values.yaml`) :**
   ```yaml
   reloader:
     readOnlyRootFileSystem: true
     deployment:
       resources:
         requests:
           cpu: 10m
           memory: 128Mi
         limits:
           cpu: 100m
           memory: 256Mi
       tolerations:
         - key: "node-role.kubernetes.io/control-plane"
           operator: "Exists"
           effect: "NoSchedule"
       securityContext:
         runAsNonRoot: true
         runAsUser: 65534
         seccompProfile:
           type: RuntimeDefault
   ```

2. **Générer les manifestes :**
   ```bash
   helm repo update
   helm template reloader stakater/reloader --version <NEW_VERSION> --namespace tools --values /tmp/reloader-values.yaml > apps/00-infra/reloader/base/manifests.yaml
   ```

3. **Nettoyer le namespace généré (si nécessaire) :**
   Supprimer le bloc `Namespace` du fichier généré si Kustomize le gère déjà via `namespace.yaml` ou le champ `namespace:`.

## Validation
**URL :** N/A (Controller)

### Méthode Automatique (Command Line)
```bash
# Vérifier que le pod est en ligne
kubectl get pods -n tools -l app=reloader-reloader
# Attendu: Pod en statut Running (1/1)
```

### Méthode Manuelle
1. Modifier un ConfigMap annoté avec `reloader.stakater.com/auto: "true"`.
2. Vérifier que les pods utilisant ce ConfigMap redémarrent (Rolling Restart).
3. Vérifier les logs de Reloader : `kubectl logs -n tools -l app=reloader-reloader`.

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :** Aucune
- **Particularités :** Surveille les ConfigMaps et Secrets pour redémarrer automatiquement les Deployments associés.
