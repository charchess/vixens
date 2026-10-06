# Sabnzbd

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://sabnzbd.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://sabnzbd.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://sabnzbd.dev.truxonline.com | grep "SABnzbd"
# Attendu: Présence de "SABnzbd"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la file d'attente de téléchargement est visible et qu'il n'y a pas d'erreurs de dossiers (Permissions NFS).

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** NFS Storage (`/volume3/Downloads`)
- **Particularités :** Client de téléchargement Usenet.
---
> ⚠️ **HIBERNATION DEV**
> L'environnement `dev` peut être hiberné pour économiser les ressources ; vérifiez l'overlay dev courant avant de conclure qu'il est actif ou inactif.
> Pour réactiver durablement l'application, suivez `docs/procedures/dev-hibernation.md` via une branche/PR ; ne décommentez pas directement l'Application ArgoCD comme mécanisme de réveil.
