# Linkwarden

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v2.4.9  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://linkwarden.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://linkwarden.dev.truxonline.com
# Attendu: HTTP 200 (ou redirect login)
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Se connecter ou vérifier la page de login.

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - `PostgreSQL` (Cluster partagé)
    - `Infisical` (Secrets DATABASE_URL, NEXTAUTH_SECRET)
- **Particularités :** Gestionnaire de favoris collaboratif.
