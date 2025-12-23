# Booklore

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://booklore.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://booklore.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** PVC Storage
- **Particularités :** Gestionnaire de livres.
