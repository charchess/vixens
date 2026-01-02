# Netbox

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v3.7.3  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | v3.7.3  |

## Validation
**URL :** https://netbox.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://netbox.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://netbox.dev.truxonline.com | grep "NetBox"
# Attendu: Présence de "NetBox"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Netbox s'affiche.
3. Vérifier que la connexion à la base de données est fonctionnelle (pas d'erreur 500 ou de page de maintenance prolongée).

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - `PostgreSQL` (Cluster partagé)
    - `Redis` (Cluster partagé)
    - `Infisical` (Secrets DB et Secret Key)
- **Particularités :** IPAM et DCIM. Migrations de base de données exécutées au démarrage du conteneur (délai possible).
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
