# ArgoCD

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v2.13.1 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://argocd.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
curl -I -k https://argocd.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface de login ArgoCD s'affiche.

## Notes Techniques
- **Namespace :** `argocd`
- **Dépendances :**
    - `Terraform` (Déploiement initial via Helm)
    - `Traefik` (Ingress)
    - `Cilium` (LoadBalancer)
- **Particularités :** Déployé initialement via Terraform (`helm_release`), puis gère ses propres applications via le pattern App-of-Apps (`root-app`). La version indiquée correspond au Chart Helm 7.7.7.
