# Whisparr

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | -       |

## Validation
**URL :** https://whisparr.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://whisparr.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://whisparr.dev.truxonline.com | grep "Whisparr"
# Attendu: Présence de "Whisparr"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface se charge.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** NFS Storage
- **Particularités :** Gestionnaire de contenu adulte.
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
