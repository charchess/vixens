# Booklore (Grimmory)

> **Migration:** Le projet original `booklore-app/booklore` a fermé. Le successeur est
> **[grimmory-tools/grimmory](https://github.com/grimmory-tools/grimmory)**.
> L'image est `ghcr.io/grimmory-tools/grimmory` depuis v2.3.0 (mars 2026).
> Les noms Kubernetes (deployment, service, ingress, namespace) restent `booklore` pour l'instant.

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Image |
|---------------|---------|-----------|-------|-------|
| Dev           | [x]     | [x]       | [x]   | ghcr.io/grimmory-tools/grimmory:v2.3.0 |
| Prod          | [x]     | [x]       | [x]   | ghcr.io/grimmory-tools/grimmory:v2.3.0 |

## Validation
**URL :** https://booklore.truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://booklore.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://booklore.truxonline.com | grep -i "booklore\|grimmory"
# Attendu: Page d'accueil de l'application
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que les fichiers dans `/bookdrop` (montage NFS) sont scannés.
3. Vérifier les logs du pod pour les évènements `BookdropMonitoringService`.

## Notes Techniques
- **Namespace :** `media`
- **Image :** `ghcr.io/grimmory-tools/grimmory:v2.3.0`
- **Port :** 6060 (Spring Boot / Tomcat)
- **Base de données :** MariaDB (sidecar `booklore-mariadb`, PVC iSCSI)
- **Backup :** DataAngel sidecar (FS-only, bucket `vixens-prod-booklore`, sizing B-small)
- **Stockage (NFS) :**
    - `/books` : `/volume3/Content/ebooks`
    - `/bookdrop` : `/volume3/Internal/incoming/ebooks`
- **Startup probe :** 30 min (failureThreshold: 180, period: 10s) — Spring Boot + NFS scan lent

## Historique
- **2026-03-22 :** Migration booklore → grimmory v2.3.0 (PR #2378), startup probe 30min (PR #2379), dataangel sizing B-small (PR #2380)
- **Cause :** GHCR `booklore-app/booklore` retourne 401 (projet fermé)

---
> **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
