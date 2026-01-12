# ADR-018: Netbird VPN Deployment Architecture

## Status
Accepted

## Date
2026-01-12

## Context

Nous avons besoin d'une solution VPN moderne pour :
- Connecter de manière sécurisée les appareils personnels au homelab
- Permettre l'accès distant aux services internes
- Créer un réseau mesh entre plusieurs sites/appareils
- Alternative self-hosted à des solutions SaaS (Tailscale, ZeroTier)

Netbird a été choisi comme solution car :
- Open-source, basé sur WireGuard (performant, moderne)
- Architecture P2P avec fallback relay
- Supporte l'authentification via OAuth2/OIDC (intégration Authentik)
- Self-hostable avec tous les composants

## Decision

### 1. Utilisation du Helm Chart Community

**Décision** : Utiliser le Helm chart community `totmicro/netbird` v1.8.2 via Kustomize helmCharts.

**Alternatives considérées** :
- Créer des manifestes Kubernetes custom
- Utiliser le Kubernetes Operator officiel (pour clients uniquement, pas serveur)
- Chart communautaire concurrent (LarkTechnologies) - jugé plus complexe

**Justification** :
- Chart `totmicro/netbird` est devenu le standard de facto de la communauté
- Maintenu activement avec support des dernières versions
- Inclut les 4 composants (management, signal, relay, dashboard)
- Compatible avec notre approche Kustomize (via helmCharts inline)
- Évite le travail de maintenance de manifestes custom
- Facilite les mises à jour (bump de version)

### 2. Base de Données: SQLite (Dev) / PostgreSQL (Prod)

**Décision** : Utiliser SQLite pour dev/validation, PostgreSQL partagé CloudNativePG pour prod.

**Alternatives considérées** :
- PostgreSQL partout (dev + prod)
- SQLite partout avec Litestream
- PostgreSQL dédié dans le namespace networking

**Justification** :

**Dev (SQLite):**
- Simplifie grandement les prérequis de validation
- Pas de dépendance PostgreSQL à configurer
- PVC 5Gi avec strategy Recreate (RWO)
- Suffisant pour tests et développement (<10 devices)
- Pattern éprouvé (AdGuard Home, autres apps)

**Prod (PostgreSQL):**
- CloudNativePG offre HA, backups automatiques, point-in-time recovery
- Mutualisation des ressources (déjà utilisé par Netbox, Docspell, etc.)
- Meilleure performance pour production (>50 devices)
- Cohérence avec notre stratégie de bases de données centralisées
- Netbird supporte nativement PostgreSQL (via POSTGRES_DSN)

**Configuration** :

**Dev:**
- PVC: `netbird-data` (5Gi, RWO, synology-iscsi-retain)
- Path: `/var/lib/netbird`
- Strategy: `Recreate` (requis pour RWO)

**Prod:**
```sql
-- Base créée manuellement sur postgresql-shared
CREATE DATABASE netbird;
CREATE USER netbird WITH ENCRYPTED PASSWORD 'xxx';
GRANT ALL PRIVILEGES ON DATABASE netbird TO netbird;
```

DSN: `postgres://netbird:password@postgresql-shared-rw.databases.svc.cluster.local:5432/netbird?sslmode=disable`

### 3. Authentification via Authentik

**Décision** : Intégrer Netbird avec notre instance Authentik pour l'authentification.

**Alternatives considérées** :
- Auth0 (SaaS, payant)
- Keycloak (plus lourd, redondant avec Authentik)
- Authentification locale Netbird (moins flexible)

**Justification** :
- Authentik déjà déployé et utilisé dans le homelab
- SSO unifié pour tous les services
- Support OIDC/OAuth2 natif dans Netbird
- Gestion centralisée des utilisateurs
- Support MFA via Authentik

**Configuration** :
- Application OAuth2 créée dans Authentik
- Client ID: `netbird`
- Scopes: `openid profile email offline_access`
- Redirect URIs: `https://netbird.[env].truxonline.com`

### 4. Architecture Réseau et Ingress

**Décision** : Exposer 4 endpoints distincts via Traefik Ingress avec Let's Encrypt.

**Endpoints** :
- `netbird.[env].truxonline.com` - Dashboard (UI)
- `netbird-api.[env].truxonline.com` - Management API
- `netbird-signal.[env].truxonline.com` - Signal service (négociation P2P)
- `netbird-relay.[env].truxonline.com` - Relay service (TURN fallback)

**Alternatives considérées** :
- Endpoint unique avec path-based routing
- LoadBalancer services (sans Ingress)

**Justification** :
- Séparation claire des services
- Facilite le debugging (logs par endpoint)
- Compatible avec les clients Netbird (nécessitent des endpoints spécifiques)
- Let's Encrypt staging pour dev, prod pour production
- HTTP → HTTPS redirect sur dashboard

### 5. Pas de Coturn Dédié

**Décision** : Ne pas déployer Coturn séparé, utiliser le relay service intégré Netbird.

**Alternatives considérées** :
- Déployer Coturn comme TURN/STUN server séparé
- Utiliser un service TURN public

**Justification** :
- Netbird relay service fournit les fonctionnalités TURN nécessaires
- Architecture moderne consolidée (depuis v0.29+)
- Réduit la complexité du déploiement
- Évite la maintenance d'un service additionnel
- Suffisant pour nos besoins (homelab, pas production enterprise)

### 6. Resources et Scaling

**Décision** : Déployer avec 1 replica par composant, resources adaptées à l'environnement.

**Dev** :
- Management: 256Mi / 100m CPU
- Signal: 128Mi / 50m CPU  
- Relay: 128Mi / 50m CPU
- Dashboard: 128Mi / 50m CPU

**Prod** : Resources doublées

**Justification** :
- 1 replica suffisant pour homelab (pas de HA critique)
- Augmentation possible via patches overlay si besoin
- Resources conservatrices pour dev (économies)
- Resources augmentées pour prod (robustesse)

## Consequences

### Positives

- Déploiement rapide via Helm chart éprouvé
- Cohérence avec notre stack (PostgreSQL shared, Authentik, Traefik)
- Maintenance simplifiée (chart community maintenu)
- SSO unifié via Authentik
- Certificats automatiques (Let's Encrypt)
- GitOps complet (ArgoCD auto-sync)

### Négatives

- Dépendance à un chart community (pas officiel)
- Migration nécessaire si chart abandonné
- Configuration PostgreSQL manuelle initiale (CREATE DATABASE)
- Pas de HA native (1 replica par composant)

### Risques

**Chart abandonné** (Faible) :
- Chart activement maintenu
- Migration vers manifestes custom possible
- Code simple à reprendre si nécessaire

**Performance PostgreSQL shared** (Faible) :
- Monitoring via Prometheus/Grafana
- Migration vers instance dédiée possible si saturé

**Scalabilité** (Moyen) :
- Solution acceptable pour homelab (<50 devices)
- Scaling horizontal possible (augmenter replicas)

## Implementation Notes

### Prérequis

1. PostgreSQL database créée manuellement
2. Secrets Infisical configurés (`/netbird` path)
3. Application OAuth2 créée dans Authentik
4. DNS records configurés (4 endpoints)

### Secrets Infisical

Paths:
- Dev: `vixens/dev/netbird`
- Prod: `vixens/prod/netbird`

Keys:
- `POSTGRES_DSN` - Connection string PostgreSQL
- `TURN_SECRET` - Secret pour authentification TURN
- `RELAY_SECRET` - Secret pour relay service
- `AUTH_CLIENT_ID` - OAuth2 Client ID (Authentik)

### Déploiement GitOps

1. Code dans `apps/40-network/netbird/`
2. ArgoCD Application : `argocd/overlays/{dev,prod}/apps/netbird.yaml`
3. Sync wave: 15 (après databases, infra)
4. Auto-sync enabled avec prune et selfHeal

### Validation

```bash
# Dashboard
curl -I https://netbird.dev.truxonline.com

# API Management
curl -k https://netbird-api.dev.truxonline.com/api/health

# Client connection
netbird up --management-url https://netbird-api.dev.truxonline.com
```

## References

- [Netbird Documentation](https://docs.netbird.io/)
- [Helm Chart Community (totmicro)](https://github.com/totmicro/helms)
- [Netbird GitHub Issue #853](https://github.com/netbirdio/netbird/issues/853) - Helm chart discussion
- [Kubernetes deployment guide](https://olav.ninja/netbird-on-kubernetes-with-keycloak)
- ADR-017: Pure Trunk-Based Single-Branch (GitOps workflow)

## Related

- ADR-017: Trunk-based workflow (main branch)
- ADR-015: Conformity scoring (quality standards)
- docs/applications/40-network/netbird.md (operational documentation)
