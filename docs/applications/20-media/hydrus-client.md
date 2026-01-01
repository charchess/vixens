# Hydrus Client (Web)

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://hydrus-web.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://hydrus-web.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://hydrus-web.dev.truxonline.com | grep "Hydrus"
# Attendu: Présence de "Hydrus"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier l'affichage de la galerie.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** `Hydrus Server`
- **Particularités :** Interface Web pour Hydrus.
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
