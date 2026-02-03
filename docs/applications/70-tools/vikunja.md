# Vikunja

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [ ]   | v0.24.2 |
| Prod          | [x]     | [x]       | [x]   | v0.24.2 |

## Validation
**URL :** https://vikunja.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://vikunja.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://vikunja.dev.truxonline.com | grep "Vikunja"
# Attendu: Présence de "Vikunja"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Créer un compte ou se connecter.
3. Créer un projet et une tâche Kanban.

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - PostgreSQL Shared Cluster
    - Redis Shared Instance
    - Infisical Secrets
- **Particularités :** Outil de Kanban/gestion de tâches. Configuré avec Postgres et Redis pour la performance.
Last validated: mar. 03 févr. 2026 17:42:27 CET
