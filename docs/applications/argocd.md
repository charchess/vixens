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
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://argocd.dev.truxonline.com
# Attendu: HTTP 301/302/307/308 (Location: https://...)

# 2. Vérifier l'accès HTTPS et le contenu
curl -L -k https://argocd.dev.truxonline.com | grep "Argo CD"
# Attendu: Présence de "Argo CD" dans le body ou title
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface de login ArgoCD s'affiche correctement sans erreur de chargement d'assets.

## Notes Techniques
- **Namespace :** `argocd`
- **Dépendances :**
    - `Terraform` (Déploiement initial via Helm)
    - `Traefik` (Ingress)
    - `Cilium` (LoadBalancer)
- **Particularités :** Déployé initialement via Terraform (`helm_release`), puis gère ses propres applications via le pattern App-of-Apps (`root-app`). La version indiquée correspond au Chart Helm 7.7.7.