# Loki

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v3.0.0  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | v3.0.0  |

## Validation
**URL :** https://loki.[env].truxonline.com (API)

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://loki.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'état "Ready" de l'API
curl -L -k https://loki.dev.truxonline.com/ready
# Attendu: HTTP 200 (ready)
```

### Méthode Manuelle
1. Vérifier la source de données Loki dans Grafana (Explore > Loki).
2. Exécuter une requête simple (`{namespace="monitoring"}`) et vérifier l'affichage des logs.

## Notes Techniques
- **Namespace :** `monitoring`
- **Dépendances :**
    - `Synology-CSI` (Stockage logs via PVC `synelia-iscsi-retain`)
- **Particularités :** Déployé via Manifestes (StatefulSet). Mode monolithique (Single Binary). Stockage sur disque persistent.
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
