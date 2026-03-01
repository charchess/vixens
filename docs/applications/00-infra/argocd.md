# ArgoCD

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v3.3.0  |
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
    - `Terraform` (Déploiement initial via Helm - Version Helm 7.7.7)
    - `GitOps Self-Managed` (Depuis Sprint 4+, version v3.3.0)
    - `Traefik` (Ingress)
    - `Cilium` (LoadBalancer)
- **Particularités :** Argo CD se gère désormais lui-même via le pattern App-of-Apps. L'utilisation du Server-Side Apply (SSA) permet la coexistence avec la release Helm initiale de Terraform. La transition a nécessité une application forcée des CRDs v3.3.0.