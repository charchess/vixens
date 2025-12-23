# Gluetun

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | v3.39.1 |
| Test          | [ ]     | [ ]       | [ ]   | -       |
| Staging       | [ ]     | [ ]       | [ ]   | -       |
| Prod          | [ ]     | [ ]       | [ ]   | -       |

## Validation
**URL :** N/A (Proxy Service)

### Méthode Automatique (Command Line)
```bash
kubectl exec -it deploy/gluetun -n services -- curl ifconfig.io
# Attendu: IP publique du serveur VPN (Suisse)
```

### Méthode Manuelle
1. Configurer une application (ex: Prowlarr) pour utiliser le proxy HTTP (gluetun:8888).

## Notes Techniques
- **Namespace :** `services` (A vérifier)
- **Dépendances :**
    - `Infisical` (Secrets Wireguard)
- **Particularités :** Client VPN (NordVPN/Wireguard). Sert de Gateway/Proxy pour les applications nécessitant l'anonymat (Prowlarr, *arr).
