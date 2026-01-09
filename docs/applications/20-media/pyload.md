# PyLoad

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [ ]     | [ ]       | [ ]   | latest  |
| Prod          | [x]     | [x]       | [x]   | latest  |

## Architecture
**Type :** Application (Kustomize)
**Namespace :** `downloads`

PyLoad est un gestionnaire de téléchargement polyvalent. Il est configuré pour router son trafic sortant via le proxy VPN **Gluetun**.

## Configuration

### Secrets Infisical
**Chemin :** `/apps/20-media/pyload`
**Variables requises :**
- `PYLOAD__ADMIN_USER` - Utilisateur administrateur.
- `PYLOAD__ADMIN_PASS` - Mot de passe administrateur.

### Proxy Gluetun
L'application utilise Gluetun comme proxy HTTP :
- **Host :** `gluetun.services.svc.cluster.local`
- **Port :** `8888`

## Stockage
- `/config` : PVC RWO (iSCSI) - iSCSI Retain.
- `/downloads` : Montage NFS - `192.168.111.69:/volume3/Downloads/pyload`.

## Validation

### Méthode Automatique (Command Line)
```bash
# Vérifier que le pod est Running
kubectl get pods -n downloads -l app.kubernetes.io/name=pyload

# Vérifier l'accès réseau (Ingress)
curl -I https://pyload.dev.truxonline.com
```

### Méthode Manuelle
1. Se connecter à l'interface web.
2. Vérifier dans les paramètres que le proxy est actif.
3. Tester un téléchargement de test.

## Notes Techniques
- **Image :** `lscr.io/linuxserver/pyload-ng:latest`
- **Port :** 8000
- **Stratégie :** Recreate (à cause du PVC RWO).

---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
