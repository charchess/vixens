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
curl -I -k https://traefik.dev.truxonline.com/dashboard/
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface dashboard de Traefik s'affiche.

## Notes Techniques
- **Namespace :** `traefik`
- **Dépendances :**
    - `Traefik`
- **Particularités :** Expose le dashboard interne de Traefik via une Ingress sécurisée.
