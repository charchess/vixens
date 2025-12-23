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
mosquitto_sub -h mqtt.dev.truxonline.com -p 1883 -t "#" -u <user> -P <pass> -C 1
# Attendu: Réception d'un message ou connexion réussie
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
