# qBittorrent

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [ ]     | [ ]       | [ ]   | latest  |
| Prod          | [ ]     | [ ]       | [ ]   | latest  |

## Architecture
**Type :** Application (Kustomize)
**Namespace :** `downloads`

qBittorrent est un client BitTorrent open-source. Il est configuré pour router son trafic sortant via le proxy VPN **Gluetun**.

## Configuration

### Secrets Infisical
**Chemin :** `/apps/20-media/qbittorrent`
**Variables requises :**
- `QBITTORRENT__WEBUI_USERNAME` - Utilisateur pour l'interface web.
- `QBITTORRENT__WEBUI_PASSWORD` - Mot de passe pour l'interface web.

### Proxy Gluetun
L'application utilise Gluetun comme proxy HTTP :
- **Host :** `gluetun.services.svc.cluster.local`
- **Port :** `8888`

## Stockage
- `/config` : PVC RWO (iSCSI) - iSCSI Retain.
- `/downloads` : Montage NFS - `192.168.111.69:/volume3/Downloads/torrents`.
- `/blackhole` : Montage NFS - `192.168.111.69:/volume3/Downloads/blackhole/torrent`.

## Validation

### Méthode Automatique (Command Line)
```bash
# Vérifier que le pod est Running
kubectl get pods -n downloads -l app.kubernetes.io/name=qbittorrent

# Vérifier l'accès réseau (Ingress)
curl -I https://qbittorrent.dev.truxonline.com
```

### Méthode Manuelle
1. Se connecter à l'interface web.
2. Configurer le proxy HTTP dans les paramètres de connexion de qBittorrent.
3. Vérifier l'IP publique via un torrent de test.

## Notes Techniques
- **Image :** `lscr.io/linuxserver/qbittorrent:latest`
- **Port UI :** 8080
- **Stratégie :** Recreate (à cause du PVC RWO).

---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
