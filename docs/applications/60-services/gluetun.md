# Gluetun

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v3.39.1 |
| Prod          | [x]     | [x]       | [x]   | v3.39.1 |

## Validation
**URL :** N/A (Proxy Service)

### Méthode Automatique (Command Line)
```bash
# Vérifier l'IP publique de sortie du VPN
kubectl exec -it deploy/gluetun -n services -- curl -s https://ifconfig.io
# Attendu: IP publique différente de celle du FAI (ex: IP suisse)
```

### Méthode Manuelle
1. Configurer une application (ex: Prowlarr) pour utiliser le proxy HTTP (gluetun:8888).
2. Vérifier que la connexion fonctionne et passe bien par le VPN.
3. Vérifier les logs pour s'assurer que la connexion Wireguard est stable ("Healthy").

## Notes Techniques
- **Namespace :** `services` (A vérifier)
- **Dépendances :**
    - `Infisical` (Secrets Wireguard)
- **Particularités :** Client VPN (NordVPN/Wireguard). Sert de Gateway/Proxy pour les applications nécessitant l'anonymat (Prowlarr, *arr).
---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
