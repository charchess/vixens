# Netbird VPN

## Informations de Déploiement
| Environnement | Déployé | Configuré | Testé | Version |
|---------------|---------|-----------|-------|---------|
| Dev           | [x]     | [x]       | [x]   | 0.62.3  |
| Prod          | [x]     | [x]       | [x]   | 0.62.3  |

## Description

Netbird est une solution VPN moderne basée sur WireGuard qui crée un réseau mesh sécurisé entre vos appareils. Il offre une alternative open-source à des solutions comme Tailscale ou ZeroTier.

### Composants déployés

1. **Management Service** : Service central de coordination (API, gestion des peers)
2. **Signal Service** : Facilite la négociation des connexions P2P entre peers
3. **Relay Service** : Service TURN/relay pour les connexions fallback
4. **Dashboard** : Interface web de gestion

## Architecture

```
┌─────────────────┐
│   Dashboard     │ ← Interface Web (https://netbird.[env].truxonline.com)
└────────┬────────┘
         │
┌────────▼────────┐
│   Management    │ ← API centrale, authentification, gestion réseau
│    Service      │   (https://netbird-api.[env].truxonline.com)
└────────┬────────┘
         │
         ├─────→ PostgreSQL (CloudNativePG shared)
         │
┌────────▼────────┐
│  Signal Service │ ← Négociation P2P
│                 │   (https://netbird-signal.[env].truxonline.com)
└─────────────────┘

┌─────────────────┐
│  Relay Service  │ ← TURN/relay fallback
│                 │   (https://netbird-relay.[env].truxonline.com)
└─────────────────┘
```

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

### Méthode Automatique (Curl)

```bash
# 1. Vérifier la redirection HTTP -> HTTPS
curl -I http://netbird.dev.truxonline.com
# Attendu: HTTP 301/302/308

# 2. Vérifier l'accès HTTPS au dashboard
curl -L -k https://netbird.dev.truxonline.com | grep -i "netbird"
# Attendu: Présence de "netbird" dans la page

# 3. Vérifier l'API Management
curl -k https://netbird-api.dev.truxonline.com/api/health
# Attendu: {"status":"ok"} ou code 200
```

### Méthode Manuelle

1. **Dashboard**: Accéder à l'URL, vérifier que l'interface s'affiche
2. **Login**: Tester l'authentification via Authentik
3. **API**: Vérifier que l'API répond (endpoint /api/health ou /api/peers)

### Client Netbird

Pour tester la connexion VPN complète:

```bash
# Installer le client netbird
curl -fsSL https://pkgs.netbird.io/install.sh | sh

# Se connecter au management server
netbird up --management-url https://netbird-api.dev.truxonline.com
```

## Notes Techniques

- **Namespace**: `networking`
- **Chart Helm**: `totmicro/netbird` v1.8.2 (community chart de facto standard)
- **Dépendances**:
  - PostgreSQL: Utilise `postgresql-shared-rw.databases.svc.cluster.local` (CloudNativePG)
  - Authentik: Authentification OAuth2/OIDC via Authentik
  - Traefik: Ingress controller avec certificats Let's Encrypt

### Base de données

**Dev**: SQLite avec PVC persistant (5Gi, RWO, `synology-iscsi-retain`)
- Stockage: `/var/lib/netbird` dans le pod management
- Strategy: `Recreate` (required pour PVC RWO)
- Backup: Peut être étendu avec Litestream si nécessaire

**Prod**: PostgreSQL partagé CloudNativePG (recommandé pour HA)
- Cluster: `postgresql-shared-rw.databases.svc.cluster.local`
- Base `netbird` à créer manuellement:
  ```sql
  CREATE DATABASE netbird;
  CREATE USER netbird WITH ENCRYPTED PASSWORD 'password_from_infisical';
  GRANT ALL PRIVILEGES ON DATABASE netbird TO netbird;
  ```

### Secrets Infisical

Les secrets suivants doivent être configurés dans Infisical (`/netbird` path):

**Dev** (`vixens/dev/netbird`):
```yaml
TURN_SECRET: "random_secret_for_turn_auth"
RELAY_SECRET: "random_secret_for_relay_auth"
AUTH_CLIENT_ID: "netbird"  # OAuth2 Client ID from Authentik
```

**Prod** (`vixens/prod/netbird`):
```yaml
POSTGRES_DSN: "postgres://netbird:password@postgresql-shared-rw.databases.svc.cluster.local:5432/netbird?sslmode=disable"
TURN_SECRET: "random_secret_for_turn_auth"
RELAY_SECRET: "random_secret_for_relay_auth"
AUTH_CLIENT_ID: "netbird"  # OAuth2 Client ID from Authentik
```

> **Note:** Les secrets PostgreSQL (`POSTGRES_DSN`) sont synchronisés dans le namespace `networking` via un `InfisicalSecret` dédié (`netbird-postgresql-credentials-sync`) qui pointe correctement vers l'environnement `prod` ou `dev` d'Infisical.

### CORS & API Access

- **Middleware:** `netbird-api-cors` (apiVersion: `traefik.io/v1alpha1`)
- **Allowed Origin:** `https://netbird.[env].truxonline.com`
- **Credentials:** `access-control-allow-credentials: true` (requis pour le dashboard)

### Configuration Authentik

1. Créer une application OAuth2/OpenID dans Authentik:
   - Name: `Netbird`
   - Client ID: `netbird`
   - Redirect URIs:
     - `https://netbird.dev.truxonline.com` (dev)
     - `https://netbird.truxonline.com` (prod)
   - Scopes: `openid`, `profile`, `email`, `offline_access`

2. Configurer l'autorisation:
   - Authorization URL: `https://authentik.[env].truxonline.com/application/o/authorize/`
   - Token URL: `https://authentik.[env].truxonline.com/application/o/token/`
   - Issuer: `https://authentik.[env].truxonline.com/application/o/netbird/`

### Particularités

- **WireGuard**: Utilise WireGuard sous le capot pour le tunneling
- **Mesh Network**: Architecture P2P avec fallback relay
- **NAT Traversal**: Signal service pour négociation ICE/STUN
- **Multi-platform**: Clients disponibles pour Linux, Windows, macOS, iOS, Android

### Resources

**Dev:**
- Management: 256Mi / 100m CPU (limit: 512Mi / 500m)
- Signal: 128Mi / 50m CPU (limit: 256Mi / 200m)
- Relay: 128Mi / 50m CPU (limit: 256Mi / 200m)
- Dashboard: 128Mi / 50m CPU (limit: 256Mi / 200m)

**Prod:** Resources augmentées pour production
- Management: 512Mi / 200m CPU (limit: 1Gi / 1000m)
- Signal: 256Mi / 100m CPU (limit: 512Mi / 500m)
- Relay: 256Mi / 100m CPU (limit: 512Mi / 500m)
- Dashboard: 256Mi / 100m CPU (limit: 512Mi / 500m)

## Références

- [Documentation officielle Netbird](https://docs.netbird.io/)
- [GitHub Netbird](https://github.com/netbirdio/netbird)
- [Helm Chart community (totmicro)](https://github.com/totmicro/helms)
- [Guide Kubernetes + Keycloak](https://olav.ninja/netbird-on-kubernetes-with-keycloak)

## Troubleshooting

### Dashboard ne charge pas

1. Vérifier que l'API Management répond: `curl -k https://netbird-api.[env].truxonline.com/api/health`
2. Vérifier les logs du dashboard: `kubectl logs -n networking netbird-dashboard-xxx`
3. Vérifier la configuration Authentik (client ID, redirect URIs)

### Clients ne peuvent pas se connecter

1. Vérifier le Signal service: `kubectl logs -n networking netbird-signal-xxx`
2. Vérifier le Relay service (fallback): `kubectl logs -n networking netbird-relay-xxx`
3. Vérifier les règles firewall (UDP 51820 pour WireGuard, ports TURN)

### Base de données PostgreSQL

1. Vérifier que la base existe: `kubectl exec -it -n databases postgresql-shared-1 -- psql -U postgres -c "\l"`
2. Vérifier la connexion: `kubectl logs -n networking netbird-management-xxx | grep postgres`
3. Vérifier le DSN dans le secret: `kubectl get secret -n networking netbird-secrets -o jsonpath='{.data.POSTGRES_DSN}' | base64 -d`
