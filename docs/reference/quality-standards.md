# Quality Standards (Goldification)

## Overview

Ce document est un **résumé** des standards de qualité pour les applications Vixens.

> **⚠️ Source de vérité:** [ADR-023: 7-Tier Goldification System v2](../adr/023-7-tier-goldification-system-v2.md)
>
> En cas de divergence, ADR-023 fait autorité.

---

## Système 7-Tiers (ADR-023 v2)

| Niveau | Nom | Philosophie |
|--------|-----|-------------|
| 🥉 1 | **Bronze** | "Déployée" — L'app tourne et est accessible |
| 🥈 2 | **Silver** | "Production Ready" — Limits, probes, TLS, secrets |
| 🥇 3 | **Gold** | "Observable" — Métriques, ServiceMonitor, Goldilocks |
| 💎 4 | **Platinum** | "Reliable" — PriorityClass, sizing justifié, PDB |
| 🟢 5 | **Emerald** | "Data Durability" — Litestream, Config-Syncer, Velero |
| 💠 6 | **Diamond** | "Secure & Integrated" — PSA, NetworkPolicies, SSO |
| 🌟 7 | **Orichalcum** | "Parfaite" — 7j stabilité, 0 CVE, sizing validé |

---

## Résumé des Prérequis par Niveau

### 🥉 Bronze — "Déployée"

- Image valide (pas de `:latest`)
- CPU/Memory requests définis
- Service configuré
- Structure Kustomize correcte (base/ + overlays/)
- Ingress configuré (si exposée)

### 🥈 Silver — "Production Ready"

Bronze +
- CPU/Memory limits définis
- Readiness probe
- Liveness probe
- Startup probe (ou bypass `vixens.io/fast-start: "true"`)
- TLS/HTTPS activé
- Secrets via Infisical

### 🥇 Gold — "Observable"

Silver +
- Métriques exposées (prometheus.io/scrape ou ServiceMonitor)
- Goldilocks activé
- `revisionHistoryLimit: 3`
- Sync-wave ArgoCD configuré
- PrometheusRule (alerting) si applicable

### 💎 Platinum — "Reliable"

Gold +
- PriorityClass assigné
- Sizing mode justifié (annotation `vixens.io/sizing-rationale`)
- Sizing revu post-Goldilocks
- PodDisruptionBudget (si multi-replica)
- Graceful shutdown (preStop hook) ou bypass `vixens.io/no-long-connections: "true"`
- HPA/KEDA si charge variable

### 🟢 Emerald — "Data Durability"

Platinum +
- Backup profile défini (`vixens.io/backup-profile: critical|standard|relaxed|ephemeral`)
- Litestream sidecar + restore initContainer (si SQLite)
- Config-Syncer sidecar + restore initContainer (si config persistante)
- Velero backup confirmé (si PVC)
- Ressources sidecars définies

### 💠 Diamond — "Secure & Integrated"

Emerald +
- PSA labels namespace (`pod-security.kubernetes.io/enforce: baseline`)
- SecurityContext durci (runAsNonRoot, drop ALL capabilities)
- NetworkPolicies Cilium L3/L4
- Authentik SSO (si auth utilisateur)
- Image digest pinning (si Renovate)
- Velero restore testé
- Homepage widget

### 🌟 Orichalcum — "Parfaite"

Diamond +
- 7 jours de stabilité (0 restart, 0 OOMKill)
- Sizing validé (VPA recommendations appliquées + stable 7j)
- Zéro CVE HIGH/CRITICAL (Trivy clean ou bypass documenté)

---

## Bypasses (Annotations)

| Annotation | Check court-circuité | Signification |
|------------|---------------------|---------------|
| `vixens.io/fast-start: "true"` | Startup probe (Silver) | Container démarre < 5s |
| `vixens.io/no-long-connections: "true"` | preStop hook (Platinum) | Pas de connexions longues |
| `vixens.io/explicitly-allow-root: "true"` | SecurityContext (Diamond) | Root requis, risque accepté |
| `vixens.io/nometrics: "true"` | Métriques + ServiceMonitor (Gold) | App sans métriques |
| `vixens.io/nossoneeded: "true"` | Authentik SSO (Diamond) | Pas d'auth utilisateur |
| `vixens.io/nohomepage: "true"` | Homepage widget (Diamond) | Non pertinent |
| `vixens.io/noingressneeded: "true"` | Ingress (Bronze) | App interne |
| `vixens.io/cve-accepted: "true"` | Trivy CVE (Diamond) | CVE accepté |

---

## Health Check Requirements

### Startup Probe (Silver - Universel)

Protège les containers à démarrage lent.

```yaml
startupProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 0
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 30  # 300s total
```

### Liveness Probe (Silver)

Détermine si le container doit être redémarré.

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

### Readiness Probe (Silver)

Détermine si le container est prêt à recevoir du trafic.

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: http
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

---

## Progression

```
Bronze → Silver → Gold → Platinum → Emerald → Diamond → Orichalcum
```

**Règle:** Une application ne peut pas sauter un niveau. Tous les prérequis d'un niveau doivent être validés avant de passer au suivant.

---

## État Actuel du Cluster (2026-03-08)

| Niveau | Count |
|--------|-------|
| 🥉 Bronze | 3 |
| 🥈 Silver | 17 |
| 🥇 Gold | 48 |
| 💎 Platinum | 17 |
| 🟢 Emerald | 0 |
| 💠 Diamond | 0 |
| 🌟 Orichalcum | 0 |

**Blocages principaux vers Emerald/Diamond:**
- 317 violations check-backup
- 237 violations check-pdb
- 121 violations check-security-context

---

## Related Documentation

- **[ADR-023: 7-Tier Goldification System v2](../adr/023-7-tier-goldification-system-v2.md)** — Source de vérité
- **[Maturity Standards Matrix](maturity-standards-matrix.md)** — Matrice détaillée
- **[STATUS.md](../STATUS.md)** — État actuel des applications

---

**Last Updated:** 2026-03-08
