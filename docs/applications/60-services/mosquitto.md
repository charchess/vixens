# Mosquitto

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v2.0.20 |
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

Les mots de passe Mosquitto sont stockés sous forme de hash (PBKDF2/SHA512) dans OpenBao. `ExternalSecret/mosquitto-password-sync` lit `vixens/dev/apps/10-home/mosquitto` en dev (chemin prod dans l'overlay) via `ClusterSecretStore/openbao` et matérialise `Secret/mosquitto-password-file`.

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
2. Mettre à jour la propriété `MOSQUITTO_PASSWD_FILE` dans le chemin OpenBao de
   l'environnement (`vixens/dev/apps/10-home/mosquitto` en dev).
3. Vérifier que `ExternalSecret/mosquitto-password-sync` est Ready et que
   `Secret/mosquitto-password-file` a été resynchronisé.
4. Le fichier est préparé par l'InitContainer au démarrage. Si un redémarrage
   opérationnel immédiat est réellement requis, le faire explicitement comme action
   runtime contrôlée ; ne pas modifier le Secret Kubernetes à la main.

## Notes Techniques
- **Namespace :** `mosquitto`
- **Dépendances :**
    - OpenBao + External Secrets Operator (`ExternalSecret/mosquitto-password-sync` → `Secret/mosquitto-password-file`)
    - `Traefik` (Entrée TCP dédiée `mqtt`)
- **Particularités :** Déployé via StatefulSet. Routage TCP (Layer 4) via `IngressRouteTCP`. Le fichier de mots de passe est géré par un InitContainer qui le copie depuis le Secret vers un volume `emptyDir` (car le Secret est ReadOnly).

---
> ⚠️ **HIBERNATION DEV**
> L'environnement `dev` peut être hiberné pour économiser les ressources ; vérifiez l'overlay dev courant avant de conclure qu'il est actif ou inactif.
> Pour réactiver durablement l'application, suivez `docs/procedures/dev-hibernation.md` via une branche/PR ; ne décommentez pas directement l'Application ArgoCD comme mécanisme de réveil.
