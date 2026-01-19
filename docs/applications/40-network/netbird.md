# Netbird VPN

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version | Tier |
|---------------|---------|-----------|-------|---------|------|
| Dev           | [x]     | [x]       | [x]   | v0.63.0 | Elite|
| Prod          | [x]     | [x]       | [x]   | v0.63.0 | Elite|

## Description

Netbird est une solution VPN moderne basée sur WireGuard qui crée un réseau mesh sécurisé entre vos appareils.

### Composants déployés

1. **Management Service** : Service central de coordination (Go).
2. **Signal Service** : Négociation P2P (Go).
3. **Relay Service** : Relay TURN pour fallback (Go).
4. **Dashboard v2** : Interface UI (SPA).

## Architecture

Déployé via des manifestes Kubernetes natifs (Kustomize) pour une meilleure stabilité.

- **Namespace**: `networking`
- **Authentification**: OIDC via Authentik (Client Type: Public pour le Dashboard).
- **Stockage**: PostgreSQL externe (partagé).
- **Ingress**: Traefik avec redirection HTTPS.
- **Sécurité**: JWKS backend récupéré via URL interne (Authentik Service) pour bypasser les problèmes de confiance TLS en Dev.

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