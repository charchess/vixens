# ArgoCD Image Updater

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v1.0.1  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://image-updater.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://image-updater.dev.truxonline.com/healthz
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Vérifier les logs pour voir si les nouvelles images sont détectées.

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - `ArgoCD`
    - `Infisical` (Credentials Registries)
- **Particularités :** Met à jour automatiquement les images des applications ArgoCD en fonction de règles (semver, latest, etc.) et commite les changements dans Git (Write-Back).
