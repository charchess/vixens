# ADR-024 — SSO Debt Register: Diamond Wave 7

**Status:** Active  
**Date:** 2026-03-07  
**Context:** Diamond tier requires Authentik ForwardAuth or explicit bypass

---

## Contexte

Le niveau Diamond (💠) de la maturity grid exige que chaque application exposée
via Ingress soit protégée par Authentik SSO (ForwardAuth middleware Traefik), **ou**
porte l'annotation `vixens.io/nossoneeded: "true"` documentant pourquoi le bypass
est justifié.

Ce document trace l'état de chaque application Wave 6 après le labeling Diamond.

---

## Décision Wave 7

### Apps avec bypass `nossoneeded` (auth propre ou sans utilisateur externe)

| App | Justification |
|-----|---------------|
| authentik | Fournisseur SSO lui-même — ne peut pas être derrière lui-même |
| jellyfin | Auth native Jellyfin (comptes utilisateurs propres) |
| jellyseerr | Auth native + liaison optionnelle Jellyfin |
| homeassistant | Auth native Home Assistant (comptes locaux + MFA) |
| mealie | Auth native Mealie |
| music-assistant | Auth native Music Assistant |
| booklore | Auth native BookLore |
| sabnzbd | Auth native SABnzbd (API key + password) |
| vaultwarden | Auth native Vaultwarden — password manager, ne PAS mettre ForwardAuth |
| linkwarden | Auth native Linkwarden |
| netbox | Auth native NetBox (Django RBAC) |
| headlamp | Auth via tokens Kubernetes (OIDC configuré) |
| docspell | Auth native Docspell |
| openclaw | API privée — pas d'utilisateurs finaux externes |
| loki | API monitoring interne — pas d'utilisateurs finaux |
| netvisor | Interface admin réseau interne |
| netbird | OIDC natif via Authentik (SSO configuré dans management config) |

### Apps SSO-dette (tier label présent, ForwardAuth manquant)

Ces apps portent `vixens.io/tier: "diamond"` sur leur Ingress mais **n'ont pas**
`nossoneeded` ni `authentik` dans leurs middlewares. La policy `check-authentik-sso`
va générer des violations Audit. Elles doivent être traitées.

| App | URL | Priorité | Action requise |
|-----|-----|----------|----------------|
| changedetection | changedetection.truxonline.com | Medium | Créer Authentik Application + ajouter middleware ForwardAuth |
| lazylibrarian | lazylibrarian.truxonline.com | Low | Idem — ou décider si `nossoneeded` acceptable |

**Note:** Ces apps sont accessibles depuis Internet sans authentification. Risque réel.
Traiter avant de considérer le Diamond complet pour ces apps.

### Apps hors scope Wave 7 (namespaces privileged / pas d'Ingress HTTP)

| App | Raison |
|-----|--------|
| mariadb-shared | Pas d'Ingress |
| redis-shared | Pas d'Ingress |
| mosquitto | Pas d'Ingress HTTP |
| gluetun | Proxy interne, pas d'Ingress utilisateur |
| penpot | Pas d'Ingress prod (à créer séparément) |

---

## Penpot — Ingress manquant

Penpot (tools/penpot) n'a pas d'Ingress prod. L'app est déployée mais non accessible
via HTTPS. Il faut créer `apps/70-tools/penpot/overlays/prod/ingress.yaml`.
Penpot a son propre système d'auth (comptes Penpot) → `nossoneeded` applicable.

---

## Étapes suivantes

1. **Traiter changedetection et lazylibrarian** :
   - Option A : Créer Application Authentik + ajouter `auth-authentik-forward-auth@kubernetescrd`
     au middleware Traefik dans l'Ingress prod
   - Option B : Décider que `nossoneeded` est acceptable (accès interne réseau uniquement)
     et documenter la décision dans ce fichier

2. **Créer Ingress prod pour Penpot** avec `nossoneeded`

3. **Vérifier les violations** `check-authentik-sso` en Audit mode :
   ```bash
   kubectl get policyreport -A | grep sso
   ```

---

## Références

- [ADR-023 v2 — 7-Tier Goldification System](023-7-tier-goldification-system-v2.md)
- [check-authentik-sso policy](../../apps/00-infra/kyverno/base/policies/check-authentik-sso.yaml)
- [PR #1907 — Wave 6 NetworkPolicies](https://github.com/charchess/vixens/pull/1907)
