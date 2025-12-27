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

## Notes Techniques
- **Namespace :** `media`
- **Gestion de la Configuration :**
    - Le fichier `config.yml` est géré dans **Infisical** au chemin `/apps/20-media/frigate/config/config.yml`.
    - Un `InfisicalSecret` synchronise ce fichier vers un secret Kubernetes `frigate-config-secret`.
    - Le déploiement monte ce secret dans `/config/config.yml`.
- **Dépendances :**
    - MQTT (`Mosquitto` interne : `mosquitto.home.svc.cluster.local`)
    - NFS Storage (`/media/frigate` -> `/volume3/Internal/frigate`)
- **Matériel :**
    - Configuré par défaut pour utiliser le CPU (détecteur `cpu1`).
