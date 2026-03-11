# AdGuard Home

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Prod          | [x]     | [x]       | [x]   | latest  |

## Validation
**URL :** https://adguard.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://adguard.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://adguard.dev.truxonline.com | grep "AdGuard Home"
# Attendu: Présence de "AdGuard Home"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface s'affiche (page de login ou dashboard).
3. Tester la résolution DNS via l'IP du LoadBalancer (UDP 53): `dig @192.168.111.69 google.com`

## Notes Techniques
- **Namespace :** `networking`
- **Dépendances :** Aucune
- **Particularités :** Serveur DNS bloqueur de publicités. Expose les ports DNS (53 UDP/TCP) via IngressRouteUDP/TCP sur le LoadBalancer.

## Known Issues

### DNS Dependency Cycle (Resolved)

**Issue:** AdGuard Home pod stuck in Init:CrashLoopBackOff during cluster bootstrap.

**Root Cause:**
- AdGuard Home provides internal DNS for the cluster
- Litestream init container needs DNS to resolve MinIO endpoint (`synelia.internal.truxonline.com`)
- Without DNS running, pod cannot start → infinite loop

**Resolution:**
1. Changed MinIO endpoint from DNS to IP: `http://192.168.111.69:9000`
2. This breaks the circular dependency (pod can start without DNS)

**Alternative:** Use cluster bootstrap ordering (sync-wave priority) to ensure MinIO starts before AdGuard.

**Status:** ✅ Fixed in production (3/3 Running)

**Related:** See production cluster health check (2026-03-10) for credential rotation details.
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
