# Hydrus Server

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://hydrus.[env].truxonline.com (API/Web)

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://hydrus.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://hydrus.dev.truxonline.com
# Attendu: HTTP 200 (Page d'accueil ou API status)
```

### Méthode Manuelle
1. Vérifier la connexion depuis le client Hydrus (test de l'API Key).

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :** PVC Storage
- **Particularités :** Serveur pour Hydrus Network (Image Booru personnel).