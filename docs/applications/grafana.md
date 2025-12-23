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
curl -I -k https://grafana.dev.truxonline.com/login
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Se connecter (SSO ou admin/admin si configuré).
3. Vérifier que les dashboards sont accessibles.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Prometheus` (DataSource)
    - `Loki` (DataSource)
    - `Infisical` (Secret admin)
- **Particularités :** Déployé via Helm Chart `grafana` (version chart 10.3.0). Sidecar activé pour l'import automatique des dashboards et datasources depuis ConfigMaps.
