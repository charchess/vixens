# ADR-023: 7-Tier Goldification System v2 (Maturité)

**Date:** 2026-03-05
**Status:** Active
**Deciders:** User, Coding Agent
**Tags:** quality, goldification, maturity
**Supersedes:** [ADR-022](022-7-tier-goldification-system.md)

---

## Contexte

ADR-022 définissait le système 7-tiers initial. Suite à une revue critique, plusieurs ajustements sont nécessaires :

- **Probes mal placées** : liveness et readiness séparées sur deux tiers sans justification opérationnelle
- **Sizing "QoS class défini"** trop vague, ne capte pas l'intention derrière le choix G/B/SB/V
- **Guaranteed QoS en Orichalcum** contradictoire avec VPA (Gold) et trop restrictif pour un homelab aux ressources limitées
- **Critères manquants** : startup probe, update strategy, alerting, topology spread, graceful shutdown, HPA/KEDA, image digest, restore testé
- **"Kyverno compliant" en Orichalcum** est tautologique : atteindre les niveaux précédents implique déjà la compliance
- **SLO/SLI et Runbooks** insuffisamment définis pour être actionnables

### Rappel de la philosophie

> La **maturité** dans ce système ne désigne pas la maturité au sens "industrie standard" mais la **complétude de la configuration**. Une app Orichalcum est une app parfaitement configurée sur laquelle il n'y a plus rien à faire. C'est un tier "vision ultime", atteignable un jour.

---

### Philosophie des bypasses

> Chaque check **bloque par défaut**. Passer un check = implémenter la fonctionnalité OU poser une annotation de bypass explicite. Un bypass est une **déclaration intentionnelle**, pas une fuite.

| Annotation | Check court-circuité | Signification |
|---|---|---|
| `vixens.io/fast-start: "true"` | Startup probe (Silver) | Container démarre en < 5s — probe inutile |
| `vixens.io/no-long-connections: "true"` | preStop hook (Platinum) | Pas de connexions longues ni d'état — shutdown immédiat OK |
| `vixens.io/explicitly-allow-root: "true"` | SecurityContext durci (Diamond) | App requiert root — risque accepté explicitement |
| `vixens.io/nometrics: "true"` | Métriques + ServiceMonitor (Gold) | App sans métriques exposables |
| `vixens.io/nossoneeded: "true"` | Authentik SSO (Diamond) | App sans authentification utilisateur |
| `vixens.io/nohomepage: "true"` | Homepage widget (Diamond) | Non pertinent pour le dashboard |
| `vixens.io/noingressneeded: "true"` | Ingress (Bronze) | App interne, non exposée |
| `vixens.io/cve-accepted: "true"` | Trivy CVE critique (Diamond) | CVE accepté, risque documenté |
| `vixens.io/digest-pinned: "true"` | Image digest (Diamond) | Opt-in : active le check (Renovate gère tag+digest) |
| `vixens.io/needs-autoscaling: "true"` | HPA/KEDA (Platinum) | Opt-in : active le check (charge variable identifiée) |

## Décision

Adopter la **version 2** du système 7-tiers avec les corrections ci-dessous.

---

## Niveaux de Maturité

### 🥉 Niveau 1 : Bronze — *"Déployée"*

**Philosophie** : L'application existe, tourne, et est correctement structurée.

| Prérequis | Type | Description |
|-----------|------|-------------|
| Image valide, pas de `:latest` | Universel | Tag fixe ou digest SHA |
| CPU/Memory requests définis | Universel | Requests présents sur tous les containers |
| Service configuré | Universel | Accessible depuis le cluster |
| Structure Kustomize correcte | Universel | base/ + overlays/ respectés |
| Ingress configuré | Contextuel | Si accessible depuis l'extérieur du cluster |

**Critère de passage** : L'application démarre et est joignable.

---

### 🥈 Niveau 2 : Silver — *"Production Ready"*

**Philosophie** : L'application peut vivre en production sans déstabiliser le cluster.

| Prérequis | Type | Description |
|-----------|------|-------------|
| CPU/Memory limits définis | Universel | Sizing label v2 présent (`vixens.io/sizing.<container>`) |
| Readiness probe | Universel | Le trafic n'est envoyé qu'à une app prête |
| Liveness probe | Universel | L'app redémarre automatiquement si elle se bloque |
| TLS/HTTPS activé | Universel | cert-manager configuré |
| Secrets via Infisical | Universel | Aucun secret hardcodé en clair |
| PVC + update strategy cohérente | Contextuel | `strategy.type: Recreate` si iSCSI/Retain |
|| Startup probe | Universel | Requis sur tous les containers. Bypass : `vixens.io/fast-start: "true"` si démarrage < 5s |

**Critère de passage** : L'application est prête pour la promotion en production.

---

### 🥇 Niveau 3 : Gold — *"Observable"*

**Philosophie** : On sait ce qui se passe. L'application est monitorée et le déploiement est maîtrisé.

| Prérequis | Type | Description |
|-----------|------|-------------|
| Métriques exposées | Universel | Annotations `prometheus.io/scrape` ou ServiceMonitor |
| Goldilocks activé | Universel | Annotation `goldilocks.fairwinds.com/enabled: "true"` — valide le sizing même en mode G-* |
| `revisionHistoryLimit: 3` | Universel | Gate : limite les ReplicaSets orphelins |
| Sync-wave ArgoCD configuré | Universel | Gate : ordre de déploiement explicitement déclaré |
| ServiceMonitor | Contextuel | Si scrape Prometheus automatique requis |
| PrometheusRule (alerting) | Contextuel | Au moins une alerte critique définie pour l'application |

**Critère de passage** : L'application est observable et son déploiement est ordonné.

---

### 💎 Niveau 4 : Platinum — *"Reliable"*

**Philosophie** : L'application est robuste, prioritisée, et son sizing a été délibérément choisi et validé.

| Prérequis | Type | Description |
|-----------|------|-------------|
| PriorityClass assigné | Universel | `vixens-critical` / `vixens-high` / `vixens-medium` / `vixens-low` |
| Sizing mode justifié | Universel | Mode G/B/SB/V choisi avec intention documentée (annotation `vixens.io/sizing-rationale` ou commentaire dans le manifest) |
| Sizing revu post-Goldilocks | Universel | Recommandations Goldilocks consultées, sizing ajusté ou refus documenté |
| PodDisruptionBudget | Contextuel | Si multi-replica / haute-disponibilité |
| topologySpreadConstraints / podAntiAffinity | Contextuel | Si multi-replica : éviter la concentration sur un seul nœud |
|| Graceful shutdown | Universel | `preStop` hook + `terminationGracePeriodSeconds` requis sur tous les containers. Bypass : `vixens.io/no-long-connections: "true"` si pas de connexions longues |
| HPA / KEDA | Contextuel | Si la charge est variable et le scaling automatique est pertinent |

**Critère de passage** : L'application a une gestion explicite et réfléchie des ressources et des priorités.

---

### 🟢 Niveau 5 : Emerald — *"Data Durability"*

**Philosophie** : Les données de l'application survivent à une panne, un restart, un désastre.

| Prérequis | Type | Description |
|-----------|------|-------------|
| Backup profile défini | Universel | Label `vixens.io/backup-profile: critical\|standard\|relaxed\|ephemeral` |
| Ressources sidecars définies | Universel | CPU/RAM limités sur tous les sidecars si présents |
| Litestream sidecar + restore initContainer | Contextuel | Si SQLite — backup continu + restore automatique au démarrage |
| Config-Syncer sidecar + restore initContainer | Contextuel | Si configuration persistante non-SQLite |
| Velero backup confirmé | Contextuel | Si PVC — namespace inclus dans un schedule Velero actif |

**Critère de passage** : L'application peut être restaurée après sinistre.

---

### 💠 Niveau 6 : Diamond — *"Secure & Integrated"*

**Philosophie** : L'application est isolée, durcie et intégrée à l'écosystème. La sécurité arrive ici volontairement (homelab : faire fonctionner avant de sécuriser).

| Prérequis | Type | Description |
|-----------|------|-------------|
| PSA labels namespace (`baseline`) | Universel | `pod-security.kubernetes.io/enforce: baseline` sur le namespace |
|| SecurityContext durci | Universel | `runAsNonRoot: true`, `allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]`, `seccompProfile: RuntimeDefault`. Bypass : `vixens.io/explicitly-allow-root: "true"` |
| NetworkPolicies Cilium L3/L4 | Universel | Isolation réseau explicite — deny-all + allow sélectif |
| Authentik SSO | Contextuel | Si authentification utilisateur requise |
| Cilium L7 policies | Contextuel | Si filtrage HTTP/gRPC requis |
| Image digest pinning (SHA) | Contextuel | Tag `image@sha256:...` — supply chain sécurisée (Renovate gère) |
| Velero restore testé | Contextuel | Restore effectivement validé en conditions réelles (pas juste "configuré") |
| Homepage widget | Contextuel | Widget Dashboard Homepage configuré et fonctionnel |

**Critère de passage** : L'application est isolée, durcie et intégrée au système d'authentification.

---

### 🌟 Niveau 7 : Orichalcum — *"Parfaite"*

**Philosophie** : Il n'y a plus rien à faire. Tier "vision ultime" — une app Orichalcum est éprouvée, autonome et documentée à un niveau opérationnel professionnel.

| Prérequis | Type | Description |
|-----------|------|-------------|
| 7 jours de stabilité | Universel | Zéro restart, zéro OOMKill sur 7 jours consécutifs |
| Sizing validé | Universel | VPA recommendations appliquées + stabilité 7j, **OU** mode G-* stable 7j sans OOMKill |
| Zéro CVE HIGH/CRITICAL | Universel | Scan Trivy propre ou bypass `vixens.io/cve-accepted: "true"` documenté |

**Critère de passage** : L'application est éprouvée, documentée et autonome.

---

## Définitions

### Runbook opérationnel *(documentaire — non scoring)*

Un runbook est un document opérationnel vivant. Le fichier `docs/applications/<category>/<app>.md` peut contenir les sections suivantes (recommandé, non obligatoire pour le scoring) :

```markdown
## Runbook opérationnel

### Vue d'ensemble
- Rôle de l'application dans le cluster
- Tier de criticité (Critical / High / Medium / Low)
- Dépendances directes (upstream / downstream)

### Procédures opérationnelles

#### Redémarrage contrôlé
Étapes pour redémarrer proprement (rollout restart, drain, etc.)

#### Restauration depuis backup
Étapes complètes pour restaurer l'état depuis le dernier backup connu.
Référencer les procédures Litestream / Velero / Config-Syncer selon le cas.

#### Mise à jour / Upgrade
Procédure d'upgrade de version (image bump, migration DB si applicable).

### Alerting & réponse

| Alerte | Sévérité | Cause probable | Action |
|--------|----------|----------------|--------|
| PodOOMKilled | Warning | Sizing insuffisant | Augmenter tier sizing |
| ...    | ...      | ...            | ...    |

### Troubleshooting

| Symptôme | Cause probable | Résolution |
|----------|----------------|------------|
| Pod en CrashLoop au démarrage | ... | ... |
| ...      | ...            | ...        |
```

### SLO/SLI *(documentaire — non scoring)*

Un **SLI** (*Service Level Indicator*) est la métrique qui mesure la santé du service.
Un **SLO** (*Service Level Objective*) est la cible sur ce SLI.

Définir dans le runbook si souhaité (non obligatoire pour le scoring) :

| Dimension | SLI | SLO homelab |
|-----------|-----|-------------|
| Disponibilité | `(1 - taux d'erreur) * 100` | Ex : > 99% sur 30 jours |
| RTO (Recovery Time Objective) | Temps entre incident et service restauré | Ex : < 4h |
| RPO (Recovery Point Objective) | Perte de données maximale acceptable | Ex : < 24h (dernier backup) |

**Note homelab** : les SLO n'ont pas besoin d'être mesurés automatiquement — les définir et les documenter suffit. La mesure automatique est un bonus.

---

## Système d'Évaluation

### Méthode

L'évaluation est une **checklist binaire** :

1. Lister tous les prérequis du niveau visé
2. Vérifier chaque prérequis universel → doit être **OK**
3. Pour chaque prérequis contextuel : déterminer s'il est applicable → si oui, doit être **OK** ; si non applicable, **ignoré**
4. Si TOUS les universels + contextuels applicables sont OK → niveau atteint
5. Un niveau manquant bloque les niveaux supérieurs (progression séquentielle)

### Progression

```
Bronze → Silver → Gold → Platinum → Emerald → Diamond → Orichalcum
```

Une application ne peut pas sauter un niveau.

### Évaluation contextuelle

Les prérequis "Contextuel" sont évalués uniquement si applicables :

- **Startup probe** : universel par défaut — bypass `vixens.io/fast-start: "true"` si démarrage < 5s
- **PVC + update strategy** : applicable si l'app utilise du stockage persistant
- **Litestream** : applicable si SQLite détecté (ConfigMap litestream.yml ou sidecar litestream)
- **Velero** : applicable si PVC présent
- **HPA/KEDA** : applicable si charge variable identifiée
- **Authentik SSO** : applicable si authentification utilisateur requise
- **Image digest** : applicable si l'image est gérée par Renovate avec digest support activé

---

## Historique des révisions

- **2026-03-05** : Version 2 — Corrections post-revue critique
  - Readiness + Liveness probe → Silver (depuis Gold)
  - Ajout : startup probe, update strategy, alerting, topology spread, graceful shutdown, HPA/KEDA, image digest, velero restore testé
  - "QoS class défini" → "Sizing mode justifié + revu post-Goldilocks"
  - Suppression de "Guaranteed QoS" en Orichalcum (remplacé par "Sizing validé")
  - Suppression de "Kyverno compliant" en Orichalcum (tautologique)
  - Définitions formelles Runbook et SLO/SLI ajoutées
  - Homepage widget → Diamond contextuel
- **2026-03-05** : Version 2.1 — Philosophie bypass + inversions defaults
  - Ajout section "Philosophie des bypasses" avec table complète des annotations
  - Startup probe : Contextuel → **Universel** (bypass `fast-start`)
  - Graceful shutdown (preStop hook) : Contextuel → **Universel** (bypass `no-long-connections`)
  - SecurityContext : mention du bypass `explicitly-allow-root`
  - Orichalcum : suppression du scoring Runbook, SLO, DR testing, Chaos engineering
    (conservés en documentation non-scoring)
  - Ajout check Orichalcum : Zéro CVE HIGH/CRITICAL
  - Définition contextuelle mise à jour pour refléter la philosophie bypass

---

## Références

- [ADR-022: 7-Tier Goldification System v1](022-7-tier-goldification-system.md) — Superseded
- [ADR-015: Conformity Scoring Grid](015-conformity-scoring-grid.md) — Deprecated
- [ADR-014: Litestream Backup Profiles](014-litestream-backup-profiles-and-recovery-patterns.md)
- [ADR-013: Layered Config & Disaster Recovery](013-layered-configuration-disaster-recovery.md)
- [docs/reference/quality-standards.md](../reference/quality-standards.md)
