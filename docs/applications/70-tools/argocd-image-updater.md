# ArgoCD Image Updater

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v1.0.1  |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://image-updater.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://image-updater.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier le health check
curl -k https://image-updater.dev.truxonline.com/healthz
# Attendu: HTTP 200 (OK)
```

### Méthode Manuelle
1. Vérifier les logs pour voir si les nouvelles images sont détectées (`kubectl logs -n tools -l app.kubernetes.io/name=argocd-image-updater`).

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - `ArgoCD`
    - `Infisical` (Credentials Registries)
- **Particularités :** Met à jour automatiquement les images des applications ArgoCD en fonction de règles (semver, latest, etc.) et commite les changements dans Git (Write-Back).