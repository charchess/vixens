# ChangeDetection.io

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://changedetection.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://changedetection.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Ajouter une URL à surveiller (ex: google.com).

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :** `Browserless` (Sidecar Chrome pour le rendu JS)
- **Particularités :** Outil de surveillance de changements de sites web. Utilise un volume PVC pour stocker l'historique.
