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
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://booklore.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://booklore.dev.truxonline.com | grep "Booklore"
# Attendu: Présence de "Booklore"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier l'accès à la bibliothèque.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** PVC Storage
- **Particularités :** Gestionnaire de livres.