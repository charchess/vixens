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
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://goldilocks.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://goldilocks.dev.truxonline.com | grep "Goldilocks"
# Attendu: Présence de "Goldilocks"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la liste des namespaces s'affiche et que les recommandations VPA sont visibles.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `VPA` (Vertical Pod Autoscaler)
    - `Metrics Server`
- **Particularités :** Déployé via Helm Chart. Recommande des requêtes/limites CPU/RAM basées sur l'usage réel observé par VPA.