# Frigate

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | latest  |

## Validation
**URL :** https://frigate.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://frigate.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://frigate.truxonline.com | grep -i "Frigate"
# Attendu: Code 200 OK
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier que l'interface Frigate se charge.
3. Vérifier que le flux caméra (si configuré) est visible.

## Notes Techniques
- **Namespace :** `media`
- **Gestion de la Configuration :**
    - Le fichier `config.yml` est géré dans **Infisical** au chemin `/apps/20-media/frigate/config/config.yml`.
    - Un `InfisicalSecret` synchronise ce fichier vers un secret Kubernetes `frigate-config-secret`.
    - Le déploiement monte ce secret dans `/config/config.yml`.
    - Les secrets MQTT (User/Password) sont gérés via des variables d'environnement dans Infisical ou substitués directement dans le fichier de config.
- **Dépendances :**
    - MQTT : `mosquitto.mosquitto.svc.cluster.local`. Authentification requise (user `frigate` à créer dans Mosquitto via hash).
    - NFS Storage : `/media/frigate` -> `/volume3/Internal/frigate` (Stockage des clips).
- **Ressources & Performance :**
    - **CPU/RAM :** Configuré pour utiliser le CPU (détecteur `cpu`).
        - Requests: 500m / 1Gi
        - Limits: 2000m / 2Gi
    - **SHM :** Taille augmentée à **512Mi** (via patch) pour supporter la haute résolution et éviter les crashs de `go2rtc`.
- **Accélération Matérielle (Piste d'amélioration) :**
    - Actuellement non configurée (utilise CPU).
    - Sur les NUC avec iGPU Intel, il est recommandé d'installer le `intel-device-plugin` et de monter `/dev/dri` pour réduire la charge CPU drastiquement.
    - **Référence :** [Proxmox Passthrough to Talos](https://johanneskueber.com/posts/proxmox_passthrough_talos/) (utile même sur bare metal pour la configuration Talos).
