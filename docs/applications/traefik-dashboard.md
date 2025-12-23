# Traefik Dashboard

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | -       |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://traefik.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://traefik.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS au dashboard
curl -L -k https://traefik.dev.truxonline.com/dashboard/ | grep "Traefik"
# Attendu: Présence de "Traefik" dans le titre ou le body
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface dashboard de Traefik s'affiche avec les routeurs et services (pas de page blanche).

## Notes Techniques
- **Namespace :** `traefik`
- **Dépendances :**
    - `Traefik`
- **Particularités :** Expose le dashboard interne de Traefik via une Ingress sécurisée.