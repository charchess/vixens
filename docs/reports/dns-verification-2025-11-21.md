# DNS Redirections Verification Report - 2025-11-21

**Date**: 2025-11-21
**Environnements vérifiés**: dev, test (DNS only), staging (DNS only), prod (DNS only)
**Statut global**: Partiellement opérationnel - Recommandations importantes

---

## Résumé Exécutif

La configuration DNS actuelle utilise une architecture **CNAME vers A record central** par environnement. Le dev est entièrement opérationnel. Test a partiellement des DNS configurés. Staging et prod n'ont pas d'enregistrements DNS.

**Architecture DNS découverte:**
- Chaque service utilise un CNAME pointant vers `vixens-{env}.truxonline.com`
- L'enregistrement A `vixens-{env}.truxonline.com` pointe vers le LoadBalancer Traefik
- Approche flexible: un seul A record à mettre à jour si l'IP change

---

## Environnement: Dev

### LoadBalancer IP

| Service | Type | External-IP | Ports |
|---------|------|-------------|-------|
| traefik | LoadBalancer | **192.168.208.70** | 80, 443 |

### DNS Records (Gandi LiveDNS)

| Record | Type | Target | Status |
|--------|------|--------|--------|
| `vixens-dev.truxonline.com` | A | 192.168.208.70 | ✅ Configuré |
| `mail.dev.truxonline.com` | CNAME | vixens-dev.truxonline.com | ✅ Configuré |
| `homeassistant.dev.truxonline.com` | CNAME | vixens-dev.truxonline.com | ✅ Configuré |
| `traefik.dev.truxonline.com` | CNAME | vixens-dev.truxonline.com | ✅ Configuré |
| `argocd.dev.truxonline.com` | CNAME | vixens-dev.truxonline.com | ✅ Configuré |
| `whoami.dev.truxonline.com` | CNAME | vixens-dev.truxonline.com | ✅ Configuré |

### Ingress Resources

| Namespace | Ingress Name | Host | IngressClass | TLS |
|-----------|--------------|------|--------------|-----|
| argocd | argocd-server-ingress | argocd.dev.truxonline.com | traefik | ✅ |
| argocd | argocd-server-http-redirect | argocd.dev.truxonline.com | traefik | ❌ (redirect) |
| homeassistant | homeassistant-ingress | homeassistant.dev.truxonline.com | none | ✅ |
| mail-gateway | mail-gateway-ingress | mail.dev.truxonline.com | none | ✅ |
| whoami | whoami | whoami.dev.truxonline.com | traefik | ✅ |
| whoami | whoami-http-redirect | whoami.dev.truxonline.com | traefik | ❌ (redirect) |

**Note:** `traefik.dev.truxonline.com` n'a pas d'Ingress dédié - le dashboard est exposé via les valeurs Helm Traefik.

### Tests de Connectivité

| Service | URL | HTTP Code | Status |
|---------|-----|-----------|--------|
| whoami | https://whoami.dev.truxonline.com | 200 | ✅ OK |
| argocd | https://argocd.dev.truxonline.com | 200 | ✅ OK |
| homeassistant | https://homeassistant.dev.truxonline.com | 200 | ✅ OK |
| mail-gateway | https://mail.dev.truxonline.com | 200 | ✅ OK |
| traefik | https://traefik.dev.truxonline.com/dashboard/ | 200 | ✅ OK |
| HTTP Redirect | http://whoami.dev.truxonline.com | 301 | ✅ Redirect vers HTTPS |

---

## Environnement: Test

### LoadBalancer IP Attendue

| VLAN | Traefik LB IP (expected) |
|------|-------------------------|
| 209 | 192.168.209.70 |

### DNS Records (Gandi LiveDNS)

| Record | Type | Target | Status |
|--------|------|--------|--------|
| `vixens-test.truxonline.com` | A | 192.168.209.70 | ✅ Configuré |
| `homeassistant.test.truxonline.com` | CNAME | vixens-test.truxonline.com | ✅ Configuré |
| `traefik.test.truxonline.com` | CNAME | vixens-test.truxonline.com | ✅ Configuré |
| `mail.test.truxonline.com` | CNAME | - | ❌ **MANQUANT** |
| `argocd.test.truxonline.com` | CNAME | - | ⚠️ Non vérifié |
| `whoami.test.truxonline.com` | CNAME | - | ⚠️ Non vérifié |

**Note:** Cluster test non déployé - DNS partiellement préconfiguré.

---

## Environnement: Staging

### LoadBalancer IP Attendue

| VLAN | Traefik LB IP (expected) |
|------|-------------------------|
| 210 | 192.168.210.70 |

### DNS Records (Gandi LiveDNS)

| Record | Type | Target | Status |
|--------|------|--------|--------|
| `vixens-staging.truxonline.com` | A | - | ❌ **NON CONFIGURÉ** |
| `*.staging.truxonline.com` | CNAME | - | ❌ **NON CONFIGURÉ** |

**Recommandation:** Créer les DNS records avant le déploiement du cluster staging.

---

## Environnement: Prod

### Cluster Configuration

| Parameter | Value |
|-----------|-------|
| **VIP Kubernetes API** | 192.168.111.190 |
| **VLAN Internal** | 111 (192.168.111.0/24) |
| **VLAN Services** | 200 (192.168.200.0/24) |
| **Traefik LB IP (expected)** | 192.168.200.70 |

### DNS Records (Gandi LiveDNS)

| Record | Type | Target | Status |
|--------|------|--------|--------|
| `vixens-prod.truxonline.com` | A | - | ❌ **NON CONFIGURÉ** |
| `mail.truxonline.com` | CNAME | - | ⚠️ Non vérifié |
| `homeassistant.truxonline.com` | CNAME | - | ⚠️ Non vérifié |

**Note:** Production n'utilise pas le pattern `{service}.prod.truxonline.com` mais directement `{service}.truxonline.com`.

---

## Architecture DNS Actuelle

### Pattern CNAME → A Record

```
┌─────────────────────────────────────┐
│  Service DNS Record                 │
│  mail.dev.truxonline.com            │
│           │                         │
│           ▼ CNAME                   │
│  vixens-dev.truxonline.com          │
│           │                         │
│           ▼ A Record                │
│  192.168.208.70                     │
│  (Traefik LoadBalancer)             │
│           │                         │
│           ▼ Ingress Routing         │
│  Kubernetes Service                 │
└─────────────────────────────────────┘
```

### Avantages de cette Architecture

1. **Maintenance simplifiée**: Un seul A record à modifier si l'IP du LoadBalancer change
2. **Scalabilité**: Ajout de nouveaux services = nouveau CNAME uniquement
3. **Cohérence**: Pattern uniforme pour tous les environnements
4. **TTL optimisé**: Les CNAMEs peuvent avoir TTL court, A record stable

### Comparaison avec Alternatives

| Approche | Avantages | Inconvénients | Status |
|----------|-----------|---------------|--------|
| **CNAME → A (Actuel)** | Un seul A à modifier, scalable | Dépendance centrale | ✅ Adopté |
| Wildcard DNS | Simple, automatique | Moins de contrôle, expose tout | ❌ Non adopté |
| External-DNS | Full automation, GitOps | Complexité, API credentials | 📅 Future |
| A Records explicites | Granulaire | Maintenance lourde | ❌ Non adopté |

---

## Recommandations

### Priorité Haute

1. **Créer DNS pour staging/prod** avant déploiement:

   **Staging (à créer dans Gandi):**
   ```
   vixens-staging.truxonline.com.  A      192.168.210.70
   mail.staging.truxonline.com.    CNAME  vixens-staging.truxonline.com.
   homeassistant.staging.truxonline.com. CNAME vixens-staging.truxonline.com.
   traefik.staging.truxonline.com. CNAME  vixens-staging.truxonline.com.
   argocd.staging.truxonline.com.  CNAME  vixens-staging.truxonline.com.
   whoami.staging.truxonline.com.  CNAME  vixens-staging.truxonline.com.
   ```

   **Prod (à créer dans Gandi):**
   ```
   vixens-prod.truxonline.com.     A      192.168.200.70
   mail.truxonline.com.            CNAME  vixens-prod.truxonline.com.
   homeassistant.truxonline.com.   CNAME  vixens-prod.truxonline.com.
   traefik.truxonline.com.         CNAME  vixens-prod.truxonline.com.
   argocd.truxonline.com.          CNAME  vixens-prod.truxonline.com.
   ```

2. **Compléter DNS test** - Ajouter les CNAMEs manquants:
   ```
   mail.test.truxonline.com.       CNAME  vixens-test.truxonline.com.
   argocd.test.truxonline.com.     CNAME  vixens-test.truxonline.com.
   whoami.test.truxonline.com.     CNAME  vixens-test.truxonline.com.
   ```

### Priorité Moyenne

3. **Standardiser IngressClass**: Certains Ingress utilisent `<none>` au lieu de `traefik`
   - Vérifier si intentionnel (fonctionnalité)
   - Sinon, ajouter `ingressClassName: traefik`

4. **TTL Optimization**:
   - A records: TTL 300-600s (stable mais permettant changements)
   - CNAME: TTL 300s (flexibilité)

### Priorité Basse

5. **External-DNS pour Phase 3+**:
   - Automatiser création DNS depuis Ingress annotations
   - Nécessite credentials Gandi API
   - ADR à créer pour décision

6. **Documentation DNS Runbook**:
   - Procédure ajout nouveau service
   - Procédure changement IP LoadBalancer
   - Troubleshooting DNS

---

## Validation Checklist

### Dev Environment

- [x] Traefik LoadBalancer IP: 192.168.208.70
- [x] DNS A record vixens-dev: Configuré
- [x] DNS CNAMEs services: Tous configurés
- [x] Ingress resources: 6 Ingress actifs
- [x] HTTPS connectivity: 5/5 services OK
- [x] HTTP → HTTPS redirect: Fonctionnel

### Test Environment

- [x] DNS A record vixens-test: Configuré (192.168.209.70)
- [x] DNS CNAME homeassistant.test: Configuré
- [x] DNS CNAME traefik.test: Configuré
- [ ] DNS CNAME mail.test: **MANQUANT**
- [ ] DNS CNAME argocd.test: À vérifier
- [ ] DNS CNAME whoami.test: À vérifier
- [ ] Cluster déployé: Non

### Staging Environment

- [ ] DNS A record vixens-staging: **À CRÉER**
- [ ] DNS CNAMEs services: **À CRÉER**
- [ ] Cluster déployé: Non

### Prod Environment

- [ ] DNS A record vixens-prod: **À CRÉER**
- [ ] DNS CNAMEs services (*.truxonline.com): **À VÉRIFIER/CRÉER**
- [ ] Cluster déployé: Non

---

## Actions Post-Vérification

- [x] Vérifier DNS records dev
- [x] Tester connectivité HTTPS dev
- [x] Vérifier DNS records test
- [x] Identifier patterns DNS
- [x] Documenter architecture CNAME → A
- [x] Configurer VIP prod à 192.168.111.190 (2025-11-21)
- [ ] DNS non-prod: DNS local suffisant (pas d'action Gandi requise)
- [ ] Créer DNS prod dans Gandi (avant déploiement cluster)
- [ ] Créer runbook DNS

## Corrections Appliquées (2025-11-21)

### Production VIP Update

**Changement:** VIP Kubernetes API mise à jour pour cohérence.

**Avant:** `192.168.111.170`
**Après:** `192.168.111.190` ✅

**Fichier modifié:**
- `terraform/environments/prod/terraform.tfvars` (cluster.endpoint et cluster.vip)

### DNS Strategy Clarification

**Dev/Test/Staging:** DNS local suffisant, pas besoin de configuration Gandi.
**Prod:** DNS Gandi requis avant déploiement cluster (pattern CNAME → A record).

---

## Conclusion

**Environnement dev: ✅ Entièrement Opérationnel**

L'architecture DNS CNAME → A record est bien implémentée et fonctionnelle pour l'environnement dev. Tous les services sont accessibles via HTTPS avec les certificats Let's Encrypt.

**Environnements test/staging/prod: ⚠️ Action Requise**

Les DNS records doivent être créés dans Gandi LiveDNS avant le déploiement des clusters. Le pattern est établi et documenté, l'implémentation est straightforward.

**Prochaines étapes:**
1. Créer DNS records dans Gandi UI pour staging/prod
2. Compléter DNS test (CNAMEs manquants)
3. Valider après déploiement cluster test (Sprint 9)

---

**Rapport généré**: 2025-11-21
**Vérifié par**: Claude Code
**Environnements testés**: dev (4/4), test (DNS only), staging (DNS only), prod (DNS only)
**Statut global**: ✅ Dev OK | ⚠️ Test Partiel | ❌ Staging/Prod À Configurer
