# n8n

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|----------|-----------|-------|---------|
| Dev           | [ ]      | [ ]       | [ ]   | latest  |
| Prod          | [ ]      | [ ]       | [ ]   | latest  |

## Validation
**URL :** https://n8n.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://n8n.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://n8n.dev.truxonline.com | grep "n8n"
# Attendu: Présence de "n8n"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la page d'accueil n8n s'affiche.
3. Tester la création d'un compte ou login.

## Notes Techniques
- **Namespace :** `services`
- **Dépendances :**
    - `Infisical` (secrets: N8N_ENCRYPTION_KEY)
    - `postgresql-shared` (database: n8n)
- **Particularités :** Workflow automation tool (like Zapier). Standard **Gold** :
    - **Priorité :** `vixens-medium`.
    - **Profil :** B-medium.
    - **Stockage :** PVC 5Gi (synelia-iscsi-retain).
    - **Database :** PostgreSQL shared cluster.
    - **Encryption :** Clé de chiffrement gérée par Infisical.

## Secrets Requis (Infisical)

### Chemin : `/apps/60-services/n8n`

| Secret | Description | Required |
|--------|-------------|----------|
| `N8N_ENCRYPTION_KEY` | Clé de chiffrement pour les credentials | Oui |

### Credentials DB (chemin: `/apps/04-databases/postgresql-shared`)

| Secret | Description | Required |
|--------|-------------|----------|
| `username` | Username PostgreSQL (owner: n8n) | Oui |
| `password` | Password PostgreSQL | Oui |

---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
