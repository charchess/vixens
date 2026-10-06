# Linkwarden

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
| Dev           | [x]     | [x]       | [x]   | v2.4.9  |
| Prod          | [x]     | [x]       | [x]   | v2.4.9  |



## Validation
**URL :** https://linkwarden.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://linkwarden.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://linkwarden.dev.truxonline.com | grep "Linkwarden"
# Attendu: Présence de "Linkwarden"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Se connecter ou vérifier la page de login.

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - `PostgreSQL` (Cluster partagé)
    - OpenBao + External Secrets Operator (`ExternalSecret/linkwarden-secrets-sync` → `Secret/linkwarden-secrets`)
- **Particularités :** Gestionnaire de favoris collaboratif.
- **Note Stockage (2025-12-31) :** Augmentation du stockage PostgreSQL partagé de 20Gi à 50Gi pour résoudre la saturation disque causant des CrashLoopBackOff.
---
> ⚠️ **HIBERNATION DEV**
> L'environnement `dev` peut être hiberné pour économiser les ressources ; vérifiez l'overlay dev courant avant de conclure qu'il est actif ou inactif.
> Pour réactiver durablement l'application, suivez `docs/procedures/dev-hibernation.md` via une branche/PR ; ne décommentez pas directement l'Application ArgoCD comme mécanisme de réveil.
