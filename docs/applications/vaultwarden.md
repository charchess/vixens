# Vaultwarden

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://vaultwarden.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://vaultwarden.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://vaultwarden.dev.truxonline.com | grep "Vaultwarden"
# Attendu: Présence de "Vaultwarden" (ou "Bitwarden")
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface de login s'affiche.
3. Tenter une création de compte (si activé) ou un login.

## Notes Techniques
- **Namespace :** `services` (A vérifier)
- **Dépendances :**
    - `Infisical` (Admin Token)
- **Particularités :** Serveur Bitwarden léger (Rust). Utilise SQLite (sur PVC) par défaut (ou Postgres si configuré).