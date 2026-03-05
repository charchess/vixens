# ADR-022: 7-Tier Goldification System (Maturité)

**Date:** 2026-02-24
**Status:** Superseded by [ADR-023](023-7-tier-goldification-system-v2.md)
**Deciders:** User, Coding Agent
**Tags:** quality, goldification, maturity

---

## Contexte

Le projet Vixens utilise un système de classification par niveaux de maturité pour évaluer et améliorer la qualité des applications déployées. Ce système remplace les anciennes grilles de notation (ADR-015) et les systèmes incohérents existants.

### Problèmes résolus

- **Incohérence des noms** : Platinum/Emerald/Elite/Diamond utilisés de façon différente selon les sources
- **Conflation criticité/maturité** : Les niveaux représentent la maturité opérationnelle, pas l'importance métier
- **Score 0-100** : Trop complexe, difficile à interpréter
- **Prérequis flous** : Pas de distinction entre universel et contextuel

---

## Décision

Adopter un système de **7 niveaux de maturité** avec distinction claire entre prérequis universels et contextuels.

---

## Niveaux de Maturité

### 🥉 Niveau 1: Bronze - "L'Application Existe"

**Philosophie** : L'application est déployée et fonctionnelle en développement.

| Prérequis | Type | Description |
|-----------|------|-------------|
| Application déployée | Universel | Image conteneur valide, pas de `:latest` |
| Resources requests définis | Universel | CPU et Memory request configurés |
| Ingress/Service configuré | Universel | Accessible via le cluster |
| Structure Kustomize correcte | Universel | base/overlays respecté |

**Critère de passage** : L'application démarre et est accessible en dev.

---

### 🥈 Niveau 2: Silver - "Production Ready"

**Philosophie** : L'application est prête pour la production avec les fondamentaux de sécurité.

| Prérequis | Type | Description |
|-----------|------|-------------|
| **Tous Bronze** | - | |
| Resource limits définis | Universel | CPU et Memory limit configurés |
| Readiness probe | Universel | Traffic refusé si pas prête |
| TLS/HTTPS activé | Universel | Cert-manager configuré |
| Secrets via Infisical | Universel | Pas de secrets hardcodés |
| Persistent storage (PVC) | Contextuel | Si données persistantes requises |

**Critère de passage** : L'application peut être promue en production.

---

### 🥇 Niveau 3: Gold - "Standard Quality"

**Philosophie** : L'application est observable et optimisée grâce aux outils de monitoring.

| Prérequis | Type | Description |
|-----------|------|-------------|
| **Tous Silver** | - | |
| Liveness probe | Universel | Redémarrage auto si故障 |
| Goldilocks activé | Universel | Annotation `goldilocks.fairwinds.com/enabled: "true"` |
| VPA annotations | Universel | Recommandations de ressources |
| Métriques exposées | Universel | Prometheus `/metrics` ou annotations |
| ServiceMonitor | Contextuel | Si scrape automatique requis |

**Critère de passage** : L'application est observable et optimisable via Goldilocks.

---

### 💎 Niveau 4: Platinum - "Reliability"

**Philosophie** : L'application est fiable avec gestion des ressources etpriorités.

| Prérequis | Type | Description |
|-----------|------|-------------|
| **Tous Gold** | - | |
| QoS class défini | Universel | Guaranteed ou Burstable |
| PriorityClass assigné | Universel | vixens-critical/high/medium/low |
| revisionHistoryLimit: 3 | Universel | Limite les replicasets |
| Sync-wave configuré | Universel | Ordre de déploiement ArgoCD |
| PodDisruptionBudget | Contextuel | Si haute-disponibilité requise |

**Critère de passage** : L'application a une gestion stricte des ressources etpriorités.

---

### 🟢 Niveau 5: Emerald - "Data Durability"

**Philosophie** : Les données de l'application sont protégées et récupérables.

| Prérequis | Type | Description |
|-----------|------|-------------|
| **Tous Platinum** | - | |
| Backup profile défini | Universel | Critical/Standard/Relaxed/Ephemeral |
| Litestream sidecar | Contextuel | Si base SQLite |
| Config-Syncer sidecar | Contextuel | Si configuration persistante |
| InitContainer restore | Contextuel | Si restauration automatique au démarrage |
| Sidecar resources définis | Universel | CPU/RAM limités pour sidecars |

**Critère de passage** : L'application peut être restaurée après sinistre.

---

### 💠 Niveau 6: Diamond - "Security & Integration"

**Philosophie** : L'application est sécurisée et intégrée à l'écosystème.

| Prérequis | Type | Description |
|-----------|------|-------------|
| **Tous Emerald** | - | |
| PSA labels (baseline/restricted) | Universel | `pod-security.kubernetes.io/enforce` |
| NetworkPolicies (L3/L4) | Universel | Cilium isolation réseau |
| SecurityContext durci | Universel | runAsNonRoot, seccompProfile, etc. |
| Authentik SSO | Contextuel | Si authentification requise |
| Homepage widgets | Contextuel | Si dashboard souhaité |
| Velero backup confirmé | Contextuel | Si PVC et backup requis |
| L7 policies (HTTP) | Contextuel | Si filtrage HTTP requis |

**Critère de passage** : L'application est isolée et intégrée au système d'authentification.

---

### 🌟 Niveau 7: Orichalcum - "Perfection"

**Philosophie** : L'application est irréprochable et opérationnellement autonome.

| Prérequis | Type | Description |
|-----------|------|-------------|
| **Tous Diamond** | - | |
| 7 jours de stabilité | Universel | Zero restart, zero crash |
| Guaranteed QoS | Universel | requests == limits (pas de throttling) |
| Runbooks documentés | Universel | Procédures d'opération |
| SLO/SLI définis | Universel | Objectifs de service |
| Policy Kyverno compliant | Universel | Respecte toutes les politiques |
| DR testing validé | Contextuel | Test de restauration réussi |
| Chaos engineering | Contextuel | Si criticité le justifie |

**Critère de passage** : L'application est éprouvée et autonome.

---

## Système d'Évaluation

### Méthode de calcul

L'évaluation se fait par **checklist binaire** :
1. Lister tous les prérequis du niveau actuel
2. Vérifier chaque prérequis (OK / KO)
3. Si TOUS les prérequis universels + contextuels applicables sont OK → Niveau atteint
4. Si un prérequis manquant → Niveau précédent

### Progression

Une application ne peut sauter un niveau :
```
Bronze → Silver → Gold → Platinum → Emerald → Diamond → Orichalcum
```

### Évaluation contextuelle

Les prérequis "Contextuel" sont évalués seulement si applicables :
- **Litestream** : Applicable si SQLite detecté → ignoré sinon
- **Authentik SSO** : Applicable si auth requise → ignoré sinon
- **PVC** : Applicable si données persistantes → ignoré sinon

---

## Mise à jour des outils

### Scripts impactés

| Script | Action |
|--------|--------|
| `scripts/reports/conformity_checker.py` | Réécrire pour afficher les 7 niveaux |
| `scripts/reports/calculate_tier.py` | Nouveau script d'évaluation |

### Rapports impactés

| Rapport | Action |
|---------|--------|
| `CONFORMITY-prod.md` | Mettre à jour après phase 2 |
| `CONFORMITY-dev.md` | Mettre à jour après phase 2 |

---

## Historique des révisions

- **2026-02-24** : Version initiale - Système 7-tiers avec distinction Universel/Contextuel

---

## Références

- [ADR-015: Conformity Scoring Grid](015-conformity-scoring-grid.md) - Déprécié
- [ADR-014: Litestream Backup Profiles](014-litestream-backup-profiles-and-recovery-patterns.md)
- [ADR-013: Layered Config & Disaster Recovery](013-layered-configuration-disaster-recovery.md)
- [docs/reference/quality-standards.md](../reference/quality-standards.md) - À mettre à jour
