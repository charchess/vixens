# Prometheus

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v2.55.1 |
| Prod          | [x]     | [x]       | [x]   | v2.55.1 |

## Status Elite ✅
- **PriorityClass:** `vixens-critical` (tous composants)
- **QoS:** **Guaranteed** (Requests = Limits)
- **PSA:** `privileged` (requis pour Node Exporter)
- **PVC:** 50Gi (Retain en Prod)

## Validation
**URL :** https://prometheus.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://prometheus.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS et le contenu (Graph)
curl -L -k https://prometheus.truxonline.com/graph | grep "Prometheus"
# Attendu: Présence de "Prometheus" dans le titre
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Prometheus s'affiche.
3. Exécuter une requête simple (ex: `up`) et vérifier qu'elle retourne des résultats (pas d'erreur "Network Error" ou vide).

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :** Aucune
- **Particularités :** Déployé via Helm Chart `prometheus` (version chart 25.30.1). Scraping automatique via annotations `prometheus.io/scrape`. Alertmanager intégré.
- **Résolution incident Prod (20/01/26) :** Correction d'une erreur de config (`multiple scrape configs with job name "kubernetes-pods"`) qui empêchait le redémarrage du serveur. Redondance supprimée dans `values.yaml`.
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
