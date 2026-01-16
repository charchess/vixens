# Netbird VPN

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [ ]   | 0.36.0  |
| Prod          | [x]     | [x]       | [ ]   | 0.36.0  |

## Description

Netbird est une solution VPN moderne basée sur WireGuard qui crée un réseau mesh sécurisé entre vos appareils.

### Composants déployés

1. **Management Service** : Service central de coordination.
2. **Signal Service** : Négociation P2P.
3. **Relay Service** : Relay TURN pour fallback.
4. **Dashboard** : Interface UI.

## Architecture

Migration effectuée vers le Chart Helm Officiel `netbirdio/netbird` (v1.9.0).

- **Namespace**: `networking`
- **Authentification**: OIDC via Authentik.
- **Stockage**: PostgreSQL externe (partagé).
- **Ingress**: Traefik (géré par le Chart Helm).

## Validation

### URLs

**Dev:**
- Dashboard: https://netbird.dev.truxonline.com
- API: https://netbird-api.dev.truxonline.com
- Signal: https://netbird-signal.dev.truxonline.com
- Relay: https://netbird-relay.dev.truxonline.com

**Prod:**
- Dashboard: https://netbird.truxonline.com
- API: https://netbird-api.truxonline.com
- Signal: https://netbird-signal.truxonline.com
- Relay: https://netbird-relay.truxonline.com

### Commandes

```bash
# Vérifier les pods
kubectl get pods -n networking -l app.kubernetes.io/instance=netbird

# Vérifier les Ingress
kubectl get ingress -n networking -l app.kubernetes.io/instance=netbird
```

## Troubleshooting

### Dashboard ne charge pas ou boucle "Unauthenticated"

1. Vérifier que l'API Management répond: `curl -k https://netbird-api.[env].truxonline.com/api/health` (devrait retourner 401 ou 200).
2. Vérifier la configuration Authentik (client ID, redirect URIs).
3. **IMPORTANT :** Vérifier que le `NETBIRD_AUTH_CLIENT_SECRET` dans Infisical correspond bien au Client Secret du provider `netbird` dans Authentik. Si le provider a été recréé, le secret a changé.

### Clients ne peuvent pas se connecter

1. Vérifier le Signal service: `kubectl logs -n networking -l app.kubernetes.io/name=netbird-signal`
2. Vérifier le Relay service (fallback): `kubectl logs -n networking -l app.kubernetes.io/name=netbird-relay`
3. Vérifier les règles firewall (UDP 51820 pour WireGuard, ports TURN).
