# Prometheus

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v2.53.x |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://prometheus.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://prometheus.dev.truxonline.com/graph
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Prometheus s'affiche.
3. Exécuter une requête simple (ex: `up`).

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :** Aucune
- **Particularités :** Déployé via Helm Chart `prometheus` (version chart 25.30.1). Scraping automatique via annotations `prometheus.io/scrape`. Alertmanager intégré.
