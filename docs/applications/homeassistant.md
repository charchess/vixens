# Home Assistant

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://homeassistant.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://homeassistant.dev.truxonline.com
# Attendu: HTTP 200 ou 302 (Redirection Login)
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la page de login s'affiche ("Home Assistant").

## Notes Techniques
- **Namespace :** `homeassistant`
- **Dépendances :**
    - `Infisical` (Secret `homeassistant-config`)
    - `Reloader` (Redémarrage auto sur modif config)
    - `PostgreSQL` (Cluster partagé, via `homeassistant-postgresql-credentials`)
    - `Traefik` (Ingress)
- **Particularités :** Utilise `hostNetwork: true` pour la découverte mDNS. Configuration montée via `subPath`.
