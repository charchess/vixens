# Mylar

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://mylar.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://mylar.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://mylar.dev.truxonline.com | grep "Mylar"
# Attendu: Présence de "Mylar"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface se charge.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** NFS Storage
- **Particularités :** Gestionnaire de Comics.