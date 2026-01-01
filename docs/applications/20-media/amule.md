# aMule

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [ ]     | [ ]       | [ ]   | latest  |
| Prod          | [ ]     | [ ]       | [ ]   | latest  |

## Architecture
**Type :** Application (Kustomize)
**Namespace :** `downloads`

aMule est un client P2P compatible eMule. Il est configuré pour router son trafic sortant via le proxy VPN **Gluetun**.

## Configuration

### Secrets Infisical
**Chemin :** `/apps/20-media/amule`
**Variables requises :**
- `AMULE__WEBUI_PWD` - Mot de passe pour l'interface web (format MD5 souvent requis).

### Proxy Gluetun
L'application utilise Gluetun comme proxy HTTP/SOCKS :
- **Host :** `gluetun.services.svc.cluster.local`
- **Port :** `8888` (HTTP) / `1080` (SOCKS)

## Stockage
- `/config` : PVC RWO (iSCSI) - iSCSI Retain.
- `/downloads` : Montage NFS - `192.168.111.69:/volume3/Downloads/amule`.

## Validation

### Méthode Automatique (Command Line)
```bash
# Vérifier que le pod est Running
kubectl get pods -n downloads -l app.kubernetes.io/name=amule

# Vérifier l'accès réseau (Ingress)
curl -I https://amule.dev.truxonline.com
```

### Méthode Manuelle
1. Se connecter à l'interface web (port 4711).
2. Configurer le proxy dans les préférences si nécessaire.

## Notes Techniques
- **Image :** `lscr.io/linuxserver/amule:latest`
- **Port UI :** 4711
- **Stratégie :** Recreate.
