# Vaultwarden

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
| Dev           | [x]     | [x]       | [x]   | 1.34.3  |
| Prod          | [x]     | [x]       | [x]   | 1.34.3  |

## Validation
**URL :** https://vaultwarden.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://vaultwarden.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://vaultwarden.dev.truxonline.com | grep "Vaultwarden"
# Attendu: Présence de "Vaultwarden" (ou "Bitwarden")
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface de login s'affiche.
3. Tenter une création de compte (si activé) ou un login.

## Notes Techniques
- **Namespace :** `services`
- **Dépendances :**
    - OpenBao + External Secrets Operator (`ExternalSecret/vaultwarden-secrets` → `Secret/vaultwarden-secrets`)
- **Particularités :** Serveur Bitwarden léger (Rust). Utilise SQLite (sur PVC) par défaut. Standard **🏆 Elite** :
    - **Priorité :** `vixens-medium`.
    - **Profil :** Small (50m/256Mi).
    - **Stockage :** Stratégie `Recreate` pour PVC RWO (réparé le 07/01/2026).
    - **Backup :** Litestream S3 (MinIO) configuré.
    - **Stabilité :** Liveness probe corrigée (path `/alive`, 60s delay) pour l'initialisation Litestream.
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
