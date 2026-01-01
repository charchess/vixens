# IT-Tools

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [ ]       | [ ]   | latest  |

## Validation
**URL :** https://it-tools.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://it-tools.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://it-tools.dev.truxonline.com | grep "IT Tools"
# Attendu: Présence de "IT Tools"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que la suite d'outils (Crypto, Converter, etc.) est visible.
3. Tester un outil simple (ex: "UUIDs generator").

## Notes Techniques
- **Namespace :** `tools`
- **Chart Helm :** `jeffresc/it-tools` (Version 0.1.4)
- **Image :** `ghcr.io/corentinth/it-tools:latest`
- **Particularités :**
  - Déploiement via ArgoCD Helm Sources pour support Renovate.
  - Ingress géré par Kustomize via l'application `it-tools-ingress`.
- **Ressources :**
  - Requests: 10m CPU / 32Mi RAM
  - Limits: 100m CPU / 128Mi RAM

---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
