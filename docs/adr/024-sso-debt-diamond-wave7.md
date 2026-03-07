# ADR-024 — SSO Debt Register: Diamond Wave 7

**Status:** Active  
**Date:** 2026-03-07  
**Updated:** 2026-03-07 (post-W7 fix: changedetection + lazylibrarian → nossoneeded)  
**Context:** Diamond tier requires Authentik ForwardAuth or explicit bypass

---

## Contexte

Le niveau Diamond (💠) de la maturity grid exige que chaque application exposée
via Ingress soit protégée par Authentik SSO (ForwardAuth middleware Traefik), **ou**
porte l'annotation `vixens.io/nossoneeded: "true"` documentant pourquoi le bypass
est justifié.

Ce document trace l'état de chaque application après le labeling Diamond (W7 + fix post-W7).

---

## Décision finale: toutes les apps passent en `nossoneeded`

La décision utilisateur (post-W7) est la suivante :

> Toutes les apps Diamond portent `nossoneeded: "true"` maintenant.
> La vraie dette SSO est trackée via des Beads issues individuelles,
> une par app, assignées @user. Le workflow : l'utilisateur configure
> l'Application Authentik dans l'UI, puis demande à l'agent de configurer
> l'Ingress côté cluster.

Cela permet :
1. D'atteindre Diamond sans bloquer sur la configuration SSO
2. De tracer la dette de façon granulaire (une Beads issue = une app)
3. De traiter les apps une par une, à la demande, en collaboration user ↔ agent

---

## État Wave 7 + fix: toutes les apps avec `nossoneeded`

| App | Justification bypass |
|-----|----------------------|
| authentik | Fournisseur SSO lui-même — ne peut pas être derrière lui-même |
| jellyfin | Auth native Jellyfin (comptes utilisateurs propres) |
| jellyseerr | Auth native + liaison optionnelle Jellyfin |
| homeassistant | Auth native Home Assistant (comptes locaux + MFA) |
| mealie | Auth native Mealie |
| music-assistant | Auth native Music Assistant |
| booklore | Auth native BookLore |
| sabnzbd | Auth native SABnzbd (API key + password) |
| vaultwarden | Auth native Vaultwarden — password manager, ForwardAuth casserait les clients Bitwarden |
| linkwarden | Auth native Linkwarden |
| netbox | Auth native NetBox (Django RBAC) |
| headlamp | Auth via tokens Kubernetes (OIDC configuré) |
| docspell | Auth native Docspell |
| openclaw | API privée — pas d'utilisateurs finaux externes |
| loki | API monitoring interne — pas d'utilisateurs finaux |
| netvisor | Interface admin réseau interne |
| netbird | OIDC natif via Authentik (SSO déjà configuré dans management config) |
| changedetection | Décision: nossoneeded acceptable — SSO configurable via Beads issue dédiée |
| lazylibrarian | Décision: nossoneeded acceptable — SSO configurable via Beads issue dédiée |

---

## Roadmap SSO: dette trackée par Beads issues

La configuration ForwardAuth Authentik pour chaque app est tracée individuellement
via des Beads issues assignées à `@user`. Le workflow pour chaque app :

1. **@user** : ouvrir la Beads issue correspondante
2. **@user** : créer l'Application Authentik dans l'UI (slug + provider OIDC/proxy)
3. **@user** : indiquer à l'agent le slug de l'application
4. **Agent** : mettre à jour l'Ingress prod pour ajouter `auth-authentik-forward-auth@kubernetescrd`
   et retirer `vixens.io/nossoneeded: "true"`

### Pattern cible Ingress (après configuration SSO)

```yaml
annotations:
  traefik.ingress.kubernetes.io/router.middlewares: >-
    traefik-redirect-https@kubernetescrd,
    auth-authentik-forward-auth@kubernetescrd
  # nossoneeded retiré
```

### Apps prioritaires pour SSO (recommandation)

| App | Raison | Risque sans SSO |
|-----|--------|-----------------|
| changedetection | Pas d'auth par défaut | Accès libre depuis Internet |
| lazylibrarian | Auth basique configurable | Accès libre depuis Internet |
| headlamp | Tokens k8s, ForwardAuth ajoute couche SSO | Accès k8s depuis Internet |
| netvisor | Interface admin réseau | Accès réseau depuis Internet |

### Apps où ForwardAuth peut casser des intégrations (traiter avec précaution)

| App | Risque |
|-----|--------|
| jellyfin | Clients mobiles/API pourraient ne plus fonctionner |
| jellyseerr | Idem — clients *arr |
| music-assistant | Clients audio |
| sabnzbd | Clients download managers |
| homeassistant | Intégrations domotique pourraient casser |
| netbird | Déjà OIDC natif, ForwardAuth redondant + risque conflit |
| vaultwarden | NE PAS FAIRE — casserait tous les clients Bitwarden |

---

## Apps hors scope (pas d'Ingress HTTP)

| App | Raison |
|-----|--------|
| mariadb-shared | Pas d'Ingress |
| redis-shared | Pas d'Ingress |
| mosquitto | Pas d'Ingress HTTP |
| gluetun | Proxy interne, pas d'Ingress utilisateur |
| penpot | Pas d'Ingress prod (à créer séparément — voir ci-dessous) |

---

## Penpot — Ingress prod manquant

Penpot (`apps/70-tools/penpot`) n'a pas d'Ingress prod. L'app est déployée mais non accessible
via HTTPS. Il faut créer `apps/70-tools/penpot/overlays/prod/ingress.yaml`.
Penpot a son propre système d'auth (comptes Penpot) → `nossoneeded` applicable.

---

## Étapes suivantes

1. ~~**Traiter changedetection et lazylibrarian**~~ → **FAIT** (nossoneeded ajouté, fix post-W7)

2. **Créer Ingress prod pour Penpot** avec `nossoneeded`

3. **Beads issues SSO debt** — une par app, assignée @user :
   - Consulter `.beads/issues.jsonl` pour les issues créées dans cette session
   - Pattern: `sso(authentik): configure ForwardAuth for <app>`

4. **Vérifier les violations** `check-authentik-sso` (doit être 0 désormais) :
   ```bash
   kubectl get policyreport -A | grep sso
   ```

---

## Références

- [ADR-023 v2 — 7-Tier Goldification System](023-7-tier-goldification-system-v2.md)
- [check-authentik-sso policy](../../apps/00-infra/kyverno/base/policies/check-authentik-sso.yaml)
- [PR #1907 — Wave 6 NetworkPolicies](https://github.com/charchess/vixens/pull/1907)
- [PR #1908 — Wave 7 tier labels + SSO bypasses](https://github.com/charchess/vixens/pull/1908)
