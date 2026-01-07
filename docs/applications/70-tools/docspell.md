# Docspell

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v0.43.0 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | v0.43.0 |

## Validation
**URL :** https://docspell.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://docspell.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://docspell.dev.truxonline.com | grep "Docspell"
# Attendu: Présence de "Docspell"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Se connecter et vérifier que l'upload de documents fonctionne.
3. Vérifier que la recherche Solr fonctionne (pas d'erreur "Solr not available").

## Notes Techniques
- **Namespace :** `services`
- **Dépendances :**
    - `PostgreSQL` (Shared Cluster)
    - `Solr` (Composant Joex interne)
    - `Infisical` (Secrets)
- **Particularités :** Gestionnaire de documents (DMS). Architecture micro-services (RestServer + Joex). Standard **✅ Valid** (Priorité `vixens-medium`, Profil Medium, DB Postgres mutualisée).
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.