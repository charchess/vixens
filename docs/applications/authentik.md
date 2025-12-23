# Authentik

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 2025.2  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://authentik.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://authentik.dev.truxonline.com/flows/-/default/authentication/
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la page de login s'affiche.
3. Se connecter en tant qu'admin (akadmin).

## Notes Techniques
- **Namespace :** `auth`
- **Dépendances :**
    - `Redis` (Cluster partagé `redis-shared`)
    - `PostgreSQL` (Cluster partagé `postgresql-shared`)
    - `Infisical` (Secrets)
- **Particularités :** Identity Provider (IdP) pour le SSO. Gère les utilisateurs et les flows d'authentification.
