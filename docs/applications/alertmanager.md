# Alertmanager

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v0.30.0 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://alertmanager.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -L -k https://alertmanager.dev.truxonline.com
# Attendu: HTTP 200 (Page HTML)
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Alertmanager s'affiche.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Infisical` (Webhook URL)
    - `Prometheus` (Source des alertes)
- **Particularités :** Déployé via Helm Chart `prometheus-alertmanager`. Webhook injecté via `sed` au démarrage.
