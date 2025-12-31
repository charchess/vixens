# Headlamp

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://headlamp.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://headlamp.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://headlamp.dev.truxonline.com | grep "Headlamp"
# Attendu: Présence de "Headlamp"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Headlamp s'affiche et liste les ressources du cluster.

## Notes Techniques
- **Namespace :** `tools`
- **Dépendances :**
    - `ServiceAccount` (Permissions cluster-view)
- **Particularités :** Authentification OIDC désactivée temporairement. Accès direct.