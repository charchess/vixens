# Alertmanager

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v0.30.0 |
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
    - OpenBao + External Secrets Operator (webhook/receiver secrets)
    - `Prometheus` (Chart parent)
- **Particularités :** Déployé en tant que **Subchart** via le chart `prometheus`. Configuration définie dans `apps/02-monitoring/prometheus/base/values.yaml`.
- **Secrets :**
    - `ExternalSecret/alertmanager-secrets-sync` uses `ClusterSecretStore/openbao`.
    - Base path: `vixens/dev/apps/02-monitoring/alertmanager` (prod overlay patches the environment path).
    - Target: `Secret/alertmanager-secrets`.
    - Required value includes `DISCORD_WEBHOOK_URL`.
