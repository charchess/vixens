# Frigate

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | latest  |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** https://frigate.[env].truxonline.com

### Méthode Automatique (Curl)
```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://frigate.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS
curl -L -k https://frigate.dev.truxonline.com | grep "Frigate"
# Attendu: Présence de "Frigate"
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier le flux des caméras et l'absence d'erreurs "MSE" ou de connexion.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - MQTT (`Mosquitto`)
    - NFS Storage (Recordings)
    - Google Coral TPU (USB passthrough) / GPU
- **Particularités :** NVR avec détection d'objets AI.