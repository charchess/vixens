# Prowlarr

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://prowlarr.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://prowlarr.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://prowlarr.dev.truxonline.com | grep "Prowlarr"
# Attendu: Présence de "Prowlarr"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface se charge.
3. Vérifier le statut des indexeurs (pas d'erreur rouge, utilisation du proxy Gluetun confirmée).

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - `Gluetun` (Proxy pour contourner les blocages)
- **Particularités :** Gestionnaire d'indexeurs (Torrent/Usenet). Synchronise les indexeurs vers Sonarr/Radarr/etc.
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
