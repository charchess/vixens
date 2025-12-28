# Renovate Bot

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 42.66.10 |
| Test          | [ ]     | [ ]       | [ ]   | -        |
| Staging       | [ ]     | [ ]       | [ ]   | -        |
| Prod          | [ ]     | [ ]       | [ ]   | -        |

## Validation

### Méthode Automatique (Kubectl)
```bash
# 1. Vérifier le CronJob existe
kubectl get cronjob -n tools renovate

# 2. Vérifier les jobs récents
kubectl get jobs -n tools -l app=renovate

# 3. Vérifier les logs du dernier job
kubectl logs -n tools -l app=renovate --tail=100

# 4. Vérifier le secret Infisical
kubectl get infisicalsecret -n tools renovate-secret-sync -o yaml
kubectl get secret -n tools renovate-secret

# 5. Vérifier la ConfigMap
kubectl get configmap -n tools renovate-config -o yaml
```

### Méthode Manuelle
1. Vérifier dans GitHub que Renovate crée des Pull Requests pour les mises à jour de dépendances.
2. Vérifier les labels "dependencies" et "renovate" sur les PRs.
3. Consulter les logs du CronJob pour s'assurer qu'il s'exécute sans erreur.

## Configuration

### Secrets Infisical
**Chemin :** `/apps/70-tools/renovate`
**Variables requises :**
- `RENOVATE_TOKEN` - GitHub Personal Access Token avec permissions:
  - `repo` (Full control of private repositories)
  - `workflow` (Update GitHub Action workflows)

### Planification
- **Schedule :** Toutes les 6 heures (`0 */6 * * *`)
- **Politique de concurrence :** `Forbid` (pas de jobs simultanés)
- **Historique :** 3 jobs réussis + 3 jobs échoués conservés

### Gestionnaires Activés
- **Terraform** - Mise à jour des modules et providers
- **Helm Values** - Mise à jour des versions de charts dans values.yaml
- **Kubernetes** - Mise à jour des images et versions dans kustomization.yaml
- **Regex** - Patterns personnalisés

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
  - `Infisical` (GitHub token)
  - GitHub repository access
- **Particularités :**
  - CronJob (pas de Deployment)
  - Utilise emptyDir pour le workspace temporaire
  - Configuration via ConfigMap (config.json)
  - PRs automatiques mais pas de merge automatique (sécurité)
  - Limite de 3 PRs concurrentes et 2 PRs par heure
  - Planification en semaine uniquement

## Références
- [Renovate Documentation](https://docs.renovatebot.com/)
- [Self-Hosting Examples](https://docs.renovatebot.com/examples/self-hosting/)
- [Kubernetes Manager](https://docs.renovatebot.com/modules/manager/kubernetes/)
- [Terraform Manager](https://docs.renovatebot.com/modules/manager/terraform/)
