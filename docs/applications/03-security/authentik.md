# Authentik

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 2025.2  |
| Prod          | [x]     | [x]       | [x]   | 2025.2  |

## Validation
**URL :** https://authentik.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://authentik.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS (Flow initial)
curl -L -k https://authentik.dev.truxonline.com/flows/-/default/authentication/ | grep "authentik"
# Attendu: Présence de "authentik" dans le contenu
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la page de login s'affiche.
3. Se connecter en tant qu'admin (akadmin) et vérifier l'accès au dashboard Admin.

## Notes Techniques
- **Namespace :** `auth`
- **Dépendances :**
    - `Redis` (Cluster partagé `redis-shared`)
    - `PostgreSQL` (Cluster partagé `postgresql-shared`)
    - `Infisical` (Secrets)
- **Particularités :** Identity Provider (IdP) pour le SSO. Gère les utilisateurs et les flows d'authentification. Configuration automatisée via **Blueprints** (`apps/03-security/authentik/base/configmap.yaml`) montés dans `/blueprints/vixens/`. Setup OIDC initial incluant Netbird. Standard **🏆 Elite** (Priorité `vixens-critical`, Profil Medium, stratégie `Recreate` pour RWO).
- **Service binding (worker) :** Le pod `authentik-worker` est un worker de queue asynchrone. Il ne reçoit aucun trafic réseau entrant et n'a pas de Service associé. Annoté avec `vixens.io/service-binding: "false"`.
