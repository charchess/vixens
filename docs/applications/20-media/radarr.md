# Radarr

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://radarr.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://radarr.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://radarr.dev.truxonline.com | grep "Radarr"
# Attendu: Présence de "Radarr"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface se charge.
3. Vérifier les connexions (Prowlarr, Download Client).

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - NFS Storage
    - `Prowlarr`
    - Download Clients
- **Particularités :** Gestionnaire de films.
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
