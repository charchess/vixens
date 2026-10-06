# ChangeDetection.io

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 0.51.4  |
| Prod          | [x]     | [x]       | [x]   | 0.51.4  |

## Validation
**URL :** https://changedetection.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://changedetection.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://changedetection.dev.truxonline.com | grep "ChangeDetection.io"
# Attendu: Présence du titre
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Ajouter une URL à surveiller (ex: google.com) et vérifier que le check passe au vert (Browserless fonctionnel).

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :** `Browserless` (Sidecar Chrome pour le rendu JS)
- **Particularités :** Outil de surveillance de changements de sites web. Utilise un volume PVC pour stocker l'historique.
---
> ⚠️ **HIBERNATION DEV**
> L'environnement `dev` peut être hiberné pour économiser les ressources ; vérifiez l'overlay dev courant avant de conclure qu'il est actif ou inactif.
> Pour réactiver durablement l'application, suivez `docs/procedures/dev-hibernation.md` via une branche/PR ; ne décommentez pas directement l'Application ArgoCD comme mécanisme de réveil.
