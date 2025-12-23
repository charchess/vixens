# Docspell

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v0.43.0 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://docspell.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://docspell.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Se connecter et uploader un document test.

## Notes Techniques
- **Namespace :** `services` (A vérifier, probable)
- **Dépendances :**
    - `PostgreSQL` (Shared Cluster)
    - `Solr` (Composant Joex interne)
    - `Infisical` (Secrets)
- **Particularités :** Gestionnaire de documents (DMS). Architecture micro-services (RestServer + Joex).
