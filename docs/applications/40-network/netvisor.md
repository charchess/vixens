# Netvisor

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Prod          | [x]     | [x]       | [x]   | latest  |

## Validation
**URL :** https://netvisor.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://netvisor.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://netvisor.dev.truxonline.com | grep "Netvisor"
# Attendu: Présence de "Netvisor"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que les métriques réseau s'affichent et se mettent à jour.

## Notes Techniques
- **Namespace :** `networking`
- **Architecture :**
  - **Server** : Deployment (1 replica) exposant l'interface web sur port 60072
  - **Daemon** : DaemonSet déployé sur tous les nœuds (3/3) pour collecter métriques réseau
  - Communication : Daemon → Server via service interne `netvisor-server.networking.svc.cluster.local`
- **Dépendances :**
  - PostgreSQL (Cluster partagé via `postgresql-shared`)
  - Redis (Cluster partagé via `redis-shared.databases.svc.cluster.local`)
  - Infisical (Secrets DATABASE_URL)
- **Configuration par environnement :**
  - `SCANOPY_PUBLIC_URL` : URL publique varie par environnement (patchée via kustomize)
    - dev: `https://netvisor.dev.truxonline.com`
    - test: `https://netvisor.test.truxonline.com`
    - staging: `https://netvisor.staging.truxonline.com`
    - prod: `https://netvisor.truxonline.com`
- **Sécurité :**
  - Daemon nécessite `privileged: true` et `hostNetwork: true` pour accès réseau
  - Toleration control-plane activée sur daemon et server
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
