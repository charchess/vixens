# Jellyfin

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://jellyfin.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://jellyfin.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://jellyfin.dev.truxonline.com/web/index.html | grep "Jellyfin"
# Attendu: Présence de "Jellyfin"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Se connecter.
3. Lire une vidéo pour vérifier le transcodage et l'accès au stockage.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - NFS Storage (`/volume3/Content`)
    - GPU (Intel QuickSync) via Device Plugin (si configuré)
- **Particularités :** Serveur de streaming média.
---
> ⚠️ **HIBERNATION DEV**
> L'environnement `dev` peut être hiberné pour économiser les ressources ; vérifiez l'overlay dev courant avant de conclure qu'il est actif ou inactif.
> Pour réactiver durablement l'application, suivez `docs/procedures/dev-hibernation.md` via une branche/PR ; ne décommentez pas directement l'Application ArgoCD comme mécanisme de réveil.
