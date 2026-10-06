# IT-Tools

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 2024.10.22 |
| Prod          | [x]     | [x]       | [x]   | 2024.10.22 |

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
> L'environnement `dev` peut être hiberné pour économiser les ressources ; vérifiez l'overlay dev courant avant de conclure qu'il est actif ou inactif.
> Pour réactiver durablement l'application, suivez `docs/procedures/dev-hibernation.md` via une branche/PR ; ne décommentez pas directement l'Application ArgoCD comme mécanisme de réveil.
