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
curl -I -k https://frigate.dev.truxonline.com
# Attendu: HTTP 200
```

### Méthode Manuelle
1. Accéder à l'URL.
2. Vérifier le flux des caméras.

## Notes Techniques
- **Namespace :** `media-stack`
- **Dépendances :**
    - MQTT (`Mosquitto`)
    - NFS Storage (Recordings)
    - Google Coral TPU (USB passthrough) / GPU
- **Particularités :** NVR avec détection d'objets AI.
