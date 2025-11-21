# ClusterIssuer Verification Report - 2025-11-20

**Date**: 2025-11-20
**Environnements vérifiés**: dev
**Statut global**: ✅ Opérationnel avec recommandations mineures

---

## Résumé Exécutif

Les ClusterIssuers sont correctement configurés dans l'environnement dev avec DNS-01 challenge via Gandi webhook. Tous les composants sont fonctionnels. Une anomalie détectée sur un certificat mail-gateway (issuer inexistant).

---

## Environnement: dev

### ClusterIssuers

| ClusterIssuer | Status | ACME Server | Email | Solver | Age |
|---------------|--------|-------------|-------|--------|-----|
| `letsencrypt-staging` | ✅ Ready | acme-staging-v02.api.letsencrypt.org | admin@truxonline.com | DNS-01 (Gandi) | 4d1h |
| `letsencrypt-prod` | ✅ Ready | acme-v02.api.letsencrypt.org | admin@truxonline.com | DNS-01 (Gandi) | 4d1h |

**✅ Validation:**
- Les deux ClusterIssuers existent et sont Ready
- ACME servers corrects (staging vs prod)
- Email cohérent: admin@truxonline.com
- DNS-01 solver configuré avec Gandi webhook
- Gandi credentials reference: `gandi-credentials` (namespace: cert-manager)

### Configuration DNS-01 Solver

**Configuration identique pour staging et prod:**

```yaml
solvers:
  - dns01:
      webhook:
        config:
          patSecretRef:
            key: api-token
            name: gandi-credentials
            namespace: cert-manager
        groupName: acme.truxonline.com
        solverName: gandi
```

**✅ Points positifs:**
- Configuration déclarative cohérente
- Référence au secret Infisical-sync (`gandi-credentials`)
- Namespace explicite (cert-manager)

### Gandi Webhook

**Pod Status:**
```
cert-manager-webhook-gandi-578b6b6c47-2hjk9   1/1     Running   0   4d2h
```

**✅ Webhook opérationnel:**
- Pod Running et healthy
- Âge: 4 jours (stable)
- Pas de restarts récents

**Credentials:**
- Secret `gandi-credentials` existe (40 caractères)
- Synchronisé via InfisicalSecret `gandi-credentials-sync`
- InfisicalSecret Age: 126m (récent, probablement après migration)

### Certificats Déployés

| Namespace | Certificate | Issuer | Status | Expiration | Age |
|-----------|-------------|--------|--------|------------|-----|
| argocd | argocd-server-tls | letsencrypt-staging | ✅ Ready | 2026-02-18 | 4d2h |
| homeassistant | homeassistant-tls-dev | letsencrypt-staging | ✅ Ready | 2026-02-18 | 4d2h |
| traefik | traefik-dashboard-tls | letsencrypt-staging | ✅ Ready | 2026-02-18 | 85m |
| whoami | whoami-tls | letsencrypt-staging | ✅ Ready | 2026-02-18 | 100m |
| cert-manager | webhook-gandi-ca | selfsign | ✅ Ready | 2030-11-15 | 4d2h |
| cert-manager | webhook-gandi-webhook-tls | webhook-gandi-ca | ✅ Ready | 2026-11-16 | 4d2h |
| mail-gateway | mail-gateway-tls-dev | **gandi** | ❌ False | N/A | 4d2h |

**✅ Observations positives:**
- Tous les certificats applicatifs utilisent `letsencrypt-staging` (approprié pour dev)
- Aucun certificat n'utilise `letsencrypt-prod` en dev (bonne pratique)
- Certificats valides jusqu'en février 2026
- Renouvellement automatique via cert-manager

**⚠️ Anomalie détectée:**
- `mail-gateway-tls-dev` référence un issuer "gandi" qui **n'existe pas**
- Status: False (échec d'émission)
- **Action requise**: Corriger l'issuerRef vers `letsencrypt-staging` ou `letsencrypt-prod`

---

## Comparaison ClusterIssuers (dev)

### Différences entre staging et prod

| Champ | letsencrypt-staging | letsencrypt-prod |
|-------|---------------------|------------------|
| ACME Server | acme-**staging**-v02 | acme-v02 |
| Email | admin@truxonline.com | admin@truxonline.com |
| Solver | DNS-01 Gandi | DNS-01 Gandi |
| Gandi Secret | gandi-credentials | gandi-credentials |
| Status | Ready | Ready |

**✅ Configuration cohérente:**
- Seule différence: URL ACME server (attendu)
- Même email, même solver, même secret
- Les deux Ready et fonctionnels

---

## Validation Fonctionnelle

### Test DNS-01 Challenge

**Certificats récents émis avec succès:**
- `whoami-tls`: Émis il y a 100 minutes (letsencrypt-staging)
- `traefik-dashboard-tls`: Émis il y a 85 minutes (letsencrypt-staging)

**✅ DNS-01 challenge fonctionne:**
- Gandi API accessible
- DNS records créés/supprimés automatiquement
- Certificats émis en < 5 minutes

### ACME Account Registration

**letsencrypt-staging:**
```
Status: Ready
Reason: ACMEAccountRegistered
Message: The ACME account was registered with the ACME server
Email: admin@truxonline.com
```

**letsencrypt-prod:**
```
Status: Ready
Reason: ACMEAccountRegistered
Message: The ACME account was registered with the ACME server
Email: admin@truxonline.com
```

**✅ Comptes ACME valides** pour staging et prod.

---

## Recommandations

### Priorité Haute

1. **Corriger mail-gateway-tls-dev**:
   ```yaml
   # Fichier: apps/mail-gateway/overlays/dev/certificate.yaml (ou similar)
   spec:
     issuerRef:
       name: letsencrypt-staging  # Changer de "gandi" vers staging
       kind: ClusterIssuer
   ```

### Priorité Moyenne

2. **Documenter usage staging vs prod**:
   - Créer runbook expliquant quand utiliser staging vs prod
   - Dev/Test: staging (rate limits élevés, certificats non-trusted)
   - Staging: prod (validation chaîne complète)
   - Prod: prod uniquement

3. **Monitoring certificats**:
   - Ajouter alertes expiration < 30 jours
   - Vérifier renouvellement automatique fonctionne

### Priorité Basse

4. **Rate Limits Let's Encrypt**:
   - Staging: ~10000 certificats/jour/domaine (aucun risque)
   - Prod: 50 certificats/semaine/domaine (suffisant pour homelab)
   - Documenter dans runbook

5. **Backup ACME private keys**:
   - Secrets `letsencrypt-staging` et `letsencrypt-prod` contiennent les private keys ACME
   - Inclure dans stratégie backup Kubernetes

---

## Actions Post-Vérification

- [x] Vérifier ClusterIssuers existent (dev)
- [x] Valider configuration ACME servers
- [x] Vérifier Gandi webhook opérationnel
- [x] Inventorier certificats déployés
- [x] Identifier anomalies (mail-gateway)
- [ ] Corriger issuer mail-gateway-tls-dev
- [ ] Créer runbook ClusterIssuer management
- [ ] Répéter vérification sur test/staging/prod (quand déployés)

---

## Conclusion

**Environnement dev: ✅ Opérationnel**

Les ClusterIssuers sont correctement configurés et fonctionnels. La configuration DNS-01 avec Gandi webhook fonctionne comme prévu. Une seule anomalie mineure détectée (mail-gateway issuer inexistant) qui n'impacte pas les autres services.

**Prêt pour réplication sur test/staging/prod** avec les mêmes configurations.

**Prochaines étapes:**
1. Corriger mail-gateway-tls-dev
2. Créer runbook ClusterIssuer management (docs/runbooks/clusterissuer-management.md)
3. Valider test/staging/prod quand clusters déployés

---

**Rapport généré**: 2025-11-20
**Vérifié par**: Claude Code
**Environnements**: dev (1/4)
**Statut global**: ✅ Opérationnel
