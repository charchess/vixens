# Mosquitto

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v2.0.20 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [x]     | [x]       | [x]   | v2.0.20 |

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

## Gestion des Utilisateurs

Les mots de passe Mosquitto sont stockés sous forme de hash (PBKDF2/SHA512) dans le secret Infisical `MOSQUITTO_PASSWD_FILE`.

### Générer un Hash de Mot de Passe
Pour ajouter un nouvel utilisateur (ex: `frigate`), vous devez générer son hash en utilisant l'utilitaire `mosquitto_passwd`. Comme l'outil n'est pas installé localement, utilisez le pod Mosquitto existant :

```bash
# Remplacer <USER> et <PASSWORD>
kubectl exec -n mosquitto mosquitto-0 -- sh -c "rm -f /tmp/p && touch /tmp/p && mosquitto_passwd -b /tmp/p <USER> <PASSWORD> && cat /tmp/p"
```

**Exemple de sortie :**
```
frigate:$7$101$aQfmqdgO+FgaVjV/$nVsVrxaYQBCX5m9rrkFtTpKJu6ysn59HrpblYVk2QbwqGbpK2B9aN3SSJzCAdsrYJuCU7aTfyZUD985Qpi2OHQ==
```

### Appliquer le Changement
1. Copier la ligne complète générée.
2. Ajouter cette ligne dans le secret Infisical `MOSQUITTO_PASSWD_FILE` (Projet `vixens`, Path `/apps/10-home/mosquitto`).
3. Redémarrer le pod Mosquitto pour que l'InitContainer mette à jour le fichier monté :
   ```bash
   kubectl rollout restart statefulset mosquitto -n mosquitto
   ```

## Notes Techniques
- **Namespace :** `mosquitto`
- **Dépendances :**
    - `Infisical` (Secret `mosquitto-password-file`)
    - `Traefik` (Entrée TCP dédiée `mqtt`)
- **Particularités :** Déployé via StatefulSet. Routage TCP (Layer 4) via `IngressRouteTCP`. Le fichier de mots de passe est géré par un InitContainer qui le copie depuis le Secret vers un volume `emptyDir` (car le Secret est ReadOnly).
