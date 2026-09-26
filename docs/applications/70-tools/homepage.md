# Homepage

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Prod          | [x]     | [x]       | [x]   | latest  |

## Validation
**URL :** https://homepage.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://homepage.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://homepage.truxonline.com | grep "Homepage"
# Attendu: Contenu de la page d'accueil
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que le dashboard d'accueil s'affiche avec les icônes des services.
3. Vérifier que les catégories (Infrastructure, Monitoring, Media, etc.) sont bien présentes.

## Notes Techniques
- **Namespace :** `tools`
- **Gestion de la configuration :**
    - OpenBao est la source de vérité ; External Secrets Operator utilise `ClusterSecretStore/openbao`.
    - `ExternalSecret/homepage-config-sync` matérialise `Secret/homepage-config-secret`.
    - Chemin dev : `vixens/dev/apps/70-tools/homepage/config`.
    - Chemin prod : `vixens/prod/apps/70-tools/homepage/config`.
    - L'initContainer `copy-initial-config` copie la configuration du Secret en lecture seule vers un `emptyDir` writable monté sur `/app/config`.
- **Secrets applicatifs :**
    - En prod, `ExternalSecret/homepage-secrets-sync` matérialise `Secret/homepage-secrets` depuis `vixens/prod/apps/70-tools/homepage`.
    - Le Deployment consomme notamment les clés Home Assistant et *arr depuis ce Secret.
- **Sécurité :**
    - `HOMEPAGE_ALLOWED_HOSTS` doit être défini dans les overlays pour valider l'accès.
    - Ne pas recréer de `InfisicalSecret` : l'ancienne intégration Infisical est retirée.

---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
