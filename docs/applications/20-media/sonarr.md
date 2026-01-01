# Sonarr

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://sonarr.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://sonarr.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://sonarr.dev.truxonline.com | grep "Sonarr"
# Attendu: Présence de "Sonarr"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface se charge.
3. Vérifier la connexion à Prowlarr et au client de téléchargement dans Settings > Connect.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - NFS Storage (`/volume3/Content`, `/volume3/Downloads`)
    - `Prowlarr` (Indexers)
    - `Sabnzbd` / `Transmission` (Download Clients)
- **Particularités :** Gestionnaire de séries TV (PVR).
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
