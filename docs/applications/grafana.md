# Grafana

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v10.x   |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://grafana.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://grafana.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS et le contenu (Page Login)
curl -L -k https://grafana.dev.truxonline.com/login | grep "Grafana"
# Attendu: Présence de "Grafana"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Se connecter (SSO ou admin/admin si configuré).
3. Vérifier que les dashboards sont accessibles et ne retournent pas d'erreur "Datasource missing".

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Prometheus` (DataSource)
    - `Loki` (DataSource)
    - `Infisical` (Secret admin)
- **Particularités :** Déployé via Helm Chart `grafana` (version chart 10.3.0). Sidecar activé pour l'import automatique des dashboards et datasources depuis ConfigMaps.