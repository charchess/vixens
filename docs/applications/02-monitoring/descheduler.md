# Descheduler (Compliant Version)

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v0.32.2 |
| Prod          | [x]     | [x]       | [x]   | v0.32.2 |

## Architecture
**Type :** Infrastructure (Static Manifests)
**Ref :** [ADR-010](../../adr/010-static-manifests-for-infrastructure-apps.md)

Le Descheduler est déployé pour rééquilibrer automatiquement les pods entre les nœuds du cluster de manière conservatrice.

## Policy (Conservative)
- **Schedule**: `0 */4 * * *` (CronJob)
- **Mode**: DryRun Enabled (`--dry-run=true`)
- **Strategy**: `LowNodeUtilization`
  - **Thresholds**: 30% (Underutilized)
  - **Target Thresholds**: 60% (Overutilized)
- **Constraints**: 
  - `numberOfNodes: 3` (Minimum pour agir)
  - `priorityThreshold: 10000` (Ne touche pas aux pods > 10000)
- **Exclusions**: Namespaces `kube-system` et `homeassistant`.

## Protections (PriorityClasses)
Les classes suivantes sont définies pour protéger les services critiques :
- `homelab-critical` (100000) : Jamais évincé (Home Assistant, Traefik, ArgoCD).
- `homelab-important` (50000) : Évincé en dernier recours (Frigate, Jellyfin).

## Validation

### Méthode Automatique (Command Line)
```bash
# Vérifier les derniers runs du CronJob
kubectl get jobs -n kube-system -l app.kubernetes.io/name=descheduler

# Vérifier les logs du dernier run (Mode DryRun)
kubectl logs -n kube-system -l job-name=<JOB_NAME>
# Attendu : "totalEvicted=0" ou la liste des pods qui SERAIENT évincés.
```

### Résultats de la Semaine 1 (Dev)
- **Status** : DryRun fonctionnel.
- **Observations** : 0 évictions détectées lors des premiers cycles (équilibre actuel satisfaisant).

## Notes Techniques
- **Namespace :** `kube-system`
- **Particularités :** Utilise des PriorityClasses pour la segmentation des évictions. La stratégie `PodLifeTime` est activée pour ignorer les pods de moins de 24h.
