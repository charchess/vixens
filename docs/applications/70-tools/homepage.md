# Homepage

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
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
- **Gestion de la Configuration :**
    - Les fichiers de configuration (`services.yaml`, `settings.yaml`, `widgets.yaml`, `bookmarks.yaml`) sont gérés dans **Infisical** au chemin `/apps/70-tools/homepage/config`.
    - Un `InfisicalSecret` synchronise ces fichiers vers un secret Kubernetes `homepage-config-secret`.
    - Un `initContainer` (`copy-initial-config`) copie le contenu du secret vers le PVC persistant `/app/config` au démarrage.
- **Dépendances :**
    - `Infisical` pour la configuration et les secrets.
- **Sécurité :**
    - `HOMEPAGE_ALLOWED_HOSTS` doit être défini dans les overlays pour valider l'accès.

---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
