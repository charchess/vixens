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