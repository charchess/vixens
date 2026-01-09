# Alertmanager

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v0.30.0 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | v0.27.0 |

## Validation
**URL :** https://alertmanager.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://alertmanager.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://alertmanager.dev.truxonline.com | grep "Alertmanager"
# Attendu: Présence de "Alertmanager"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Alertmanager s'affiche et liste les alertes (même vides).

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Infisical` (Webhook URL)
    - `Prometheus` (Source des alertes)
- **Particularités :** Déployé via Helm Chart `prometheus-alertmanager`.
- **Secrets :**
    - Path Infisical : `/apps/02-monitoring/alertmanager`
    - Variables requises : `DISCORD_WEBHOOK_URL`