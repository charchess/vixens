# Goldilocks

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v10.2.0 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://goldilocks.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://goldilocks.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la liste des namespaces s'affiche.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `VPA` (Vertical Pod Autoscaler)
    - `Metrics Server`
- **Particularités :** Déployé via Helm Chart. Recommande des requêtes/limites CPU/RAM basées sur l'usage réel observé par VPA.
