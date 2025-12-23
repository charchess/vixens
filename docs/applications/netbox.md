# Netbox

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v3.7.3  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://netbox.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://netbox.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Netbox s'affiche.

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - `PostgreSQL` (Cluster partagé)
    - `Redis` (Cluster partagé)
    - `Infisical` (Secrets DB et Secret Key)
- **Particularités :** IPAM et DCIM. Migrations de base de données exécutées au démarrage du conteneur (délai possible).
