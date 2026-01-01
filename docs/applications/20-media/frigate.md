# Frigate

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | latest  |

## Validation

### Accès Web (Interface Utilisateur)
**URL :** https://frigate.[env].truxonline.com

#### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://frigate.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://frigate.truxonline.com | grep -i "Frigate"
# Attendu: Code 200 OK
```

#### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Frigate se charge.
3. Vérifier que les flux caméras sont visibles.

### Accès Direct (Home Assistant Integration)
Frigate expose également des ports TCP directs via le LoadBalancer Traefik (192.168.201.70 en prod):

**Ports exposés:**
- **5000**: HTTP API (sans TLS) - Accès: `http://192.168.201.70:5000`
- **8554**: RTSP streaming - Accès: `rtsp://192.168.201.70:8554`
- **8555**: WebRTC streaming - Accès: Port 8555

**⚠️ Important:** Les ports 5000/8554/8555 sont du TCP brut (pas de HTTPS). Ne pas accéder via `https://frigate.truxonline.com:5000`.

**Test des ports TCP:**
```bash
# Vérifier que les ports sont exposés
kubectl -n media get svc frigate -o yaml | grep -A 5 "ports:"

# Tester l'API HTTP (port 5000)
curl http://192.168.201.70:5000/api/config | jq .cameras

# Vérifier les IngressRouteTCP
kubectl -n media get ingressroutetcp
```

## Notes Techniques

### Namespace & Configuration
- **Namespace :** `media`
- **Gestion de la Configuration :**
    - Le fichier `config.yml` est géré dans **Infisical** au chemin `/apps/20-media/frigate/config/config.yml`.
    - Un `InfisicalSecret` synchronise ce fichier vers un secret Kubernetes `frigate-config-secret`.
    - Un **InitContainer** (`copy-config`) copie le contenu du secret vers le volume PVC `/config` au démarrage.
    - Cela permet au fichier d'être inscriptible (non read-only) par Frigate tout en conservant Infisical comme source de vérité.
    - Les secrets MQTT (User/Password) sont gérés via des variables d'environnement dans Infisical.

### Dépendances
- **MQTT :** `mosquitto.mosquitto.svc.cluster.local:1883`. Authentification requise (user `frigate` à créer dans Mosquitto via hash).
- **NFS Storage :** `/media/frigate` -> `/volume3/Internal/frigate` (Synology NAS) - Stockage des clips vidéo.
- **Traefik LoadBalancer :** Ports 5000/8554/8555 exposés via IngressRouteTCP pour Home Assistant.

### Ressources & Performance
- **CPU/RAM (Production) :**
    - Requests: 1000m / 2Gi
    - Limits: 4000m / 8Gi
- **SHM :** Taille augmentée à **4Gi** (production) pour supporter la haute résolution et éviter les crashs de `go2rtc`.
- **Cache :** Volume emptyDir (Memory) à `/tmp/cache` pour performances.

### Accélération Matérielle GPU ✅
- **Status :** **ACTIVÉE** (Intel iGPU)
- **Configuration :**
    - `securityContext.privileged: true` (requis sur Talos/NUC 12 pour accès GPU complet)
    - Volume hostPath `/dev/dri` monté dans le conteneur
    - Variable `LIBVA_DRIVER_NAME=iHD` (driver moderne pour Intel 12ème Gen+)
    - `ffmpeg.hwaccel_args: preset-vaapi` dans config Frigate
- **Bénéfice :** Réduction drastique de la charge CPU lors du décodage/encodage vidéo.
- **Référence :** [Proxmox Passthrough to Talos](https://johanneskueber.com/posts/proxmox_passthrough_talos/)

### Flux Caméras (RTSP)
- **Format recommandé :** RTSP (pas HTTP FLV/RTMP)
- **Reolink Duo 2 :**
    - Main stream: `rtsp://user:pass@ip:554/Preview_01_main` (H.265/HEVC 8MP)
    - Sub stream: `rtsp://user:pass@ip:554/Preview_01_sub` (détection)
- **Tapo C200/C210 :**
    - Main stream: `rtsp://user:pass@ip:554/stream1`
    - Sub stream: `rtsp://user:pass@ip:554/stream2`
- **⚠️ Note :** HTTP FLV ne fonctionne pas avec les caméras H.265/HEVC.

### Exposition des Ports (Home Assistant)
- **Architecture :** Ports exposés via **Traefik LoadBalancer** (même IP que Traefik)
- **IP LoadBalancer (prod) :** 192.168.201.70
- **Ports TCP exposés :**
    - **5000** : HTTP API (frigate-api entrypoint)
    - **8554** : RTSP streaming (frigate-rtsp entrypoint)
    - **8555** : WebRTC streaming (frigate-webrtc entrypoint)
- **Mécanisme :** IngressRouteTCP (Traefik CRD) route le trafic de Traefik → service ClusterIP `frigate`
- **Fichiers :**
    - `apps/00-infra/traefik/values/common.yaml` - Définition des entrypoints
    - `apps/20-media/frigate/base/ingressroute-tcp.yaml` - Routes TCP

---
> ⚠️ **HIBERNATION DEV**
> Cette application est désactivée dans l'environnement `dev` pour économiser les ressources.
> Pour tester des évolutions, décommentez-la dans `argocd/overlays/dev/kustomization.yaml` avant de déployer.
