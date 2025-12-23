# Mosquitto

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v2.0.20 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** mqtt.[env].truxonline.com (ou IP du LB sur port 1883)

### Méthode Automatique (Command Line)
```bash
# Vérifier la connexion TCP/MQTT (nécessite le client mosquitto)
mosquitto_sub -h mqtt.dev.truxonline.com -p 1883 -t "#" -u <user> -P <pass> -C 1
# Attendu: Connexion réussie, réception d'un message ou timeout (mais pas Connection Refused)

# Alternative simple (netcat)
nc -zv mqtt.dev.truxonline.com 1883
# Attendu: Connection to mqtt.dev.truxonline.com 1883 port [tcp/*] succeeded!
```

### Méthode Manuelle
1. Utiliser MQTT Explorer.
2. Connexion à l'hôte avec les identifiants.

## Notes Techniques
- **Namespace :** `mosquitto`
- **Dépendances :**
    - `Infisical` (Secret `mosquitto-password-file`)
    - `Traefik` (Entrée TCP dédiée `mqtt`)
- **Particularités :** Déployé via StatefulSet. Routage TCP (Layer 4) via `IngressRouteTCP`.