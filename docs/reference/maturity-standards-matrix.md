# 📊 Matrice des Standards de Maturité (ADR-023 v2.1)

**Source autoritaire:** [ADR-023: 7-Tier Goldification System v2](../adr/023-7-tier-goldification-system-v2.md)

Ce document récapitule les exigences techniques pour chaque tier de maturité, leur type (Universel/Contextuel), et les mécanismes de bypass associés.

---

## 🥉 Niveau 1 : Bronze — *"Déployée"*

**Philosophie:** L'application existe, tourne, et est correctement structurée.

| Prérequis | Type | Description | Bypass |
|-----------|------|-------------|--------|
| Image valide, pas de `:latest` | Universel | Tag fixe ou digest SHA | - |
| CPU/Memory requests définis | Universel | Requests présents sur tous les containers | - |
| Service configuré | Universel | Accessible depuis le cluster | - |
| Structure Kustomize correcte | Universel | `base/` + `overlays/` respectés | - |
| Ingress configuré | Contextuel | Si accessible depuis l'extérieur du cluster | `vixens.io/noingressneeded: "true"` |

**Critère de passage:** L'application démarre et est joignable.

---

## 🥈 Niveau 2 : Silver — *"Production Ready"*

**Philosophie:** L'application peut vivre en production sans déstabiliser le cluster.

| Prérequis | Type | Description | Bypass |
|-----------|------|-------------|--------|
| CPU/Memory limits définis | Universel | Sizing label v2 présent (`vixens.io/sizing.<container>`) | - |
| Readiness probe | Universel | Le trafic n'est envoyé qu'à une app prête | - |
| Liveness probe | Universel | L'app redémarre automatiquement si elle se bloque | - |
| Startup probe | Universel | Requis sur tous les containers | `vixens.io/fast-start: "true"` si démarrage < 5s |
| TLS/HTTPS activé | Universel | cert-manager configuré | - |
| Secrets via Infisical | Universel | Aucun secret hardcodé en clair | - |
| PVC + update strategy cohérente | Contextuel | `strategy.type: Recreate` si iSCSI/Retain | - |

**Critère de passage:** L'application est prête pour la promotion en production.

---

## 🥇 Niveau 3 : Gold — *"Observable"*

**Philosophie:** On sait ce qui se passe. L'application est monitorée et le déploiement est maîtrisé.

| Prérequis | Type | Description | Bypass |
|-----------|------|-------------|--------|
| Métriques exposées | Universel | Annotations `prometheus.io/scrape` ou ServiceMonitor | `vixens.io/nometrics: "true"` |
| Goldilocks activé | Universel | Annotation `goldilocks.fairwinds.com/enabled: "true"` — valide le sizing même en mode G-* | - |
| `revisionHistoryLimit: 3` | Universel | Limite les ReplicaSets orphelins | - |
| Sync-wave ArgoCD configuré | Universel | Ordre de déploiement explicitement déclaré | - |
| ServiceMonitor | Contextuel | Si scrape Prometheus automatique requis | - |
| PrometheusRule (alerting) | Contextuel | Au moins une alerte critique définie pour l'application | - |

**Critère de passage:** L'application est observable et son déploiement est ordonné.

---

## 💎 Niveau 4 : Platinum — *"Reliable"*

**Philosophie:** L'application est robuste, prioritisée, et son sizing a été délibérément choisi et validé.

| Prérequis | Type | Description | Bypass |
|-----------|------|-------------|--------|
| PriorityClass assigné | Universel | `vixens-critical` / `vixens-high` / `vixens-medium` / `vixens-low` | - |
| Sizing mode justifié | Universel | Mode G/B/SB/V choisi avec intention documentée (annotation `vixens.io/sizing-rationale` ou commentaire dans le manifest) | - |
| Sizing revu post-Goldilocks | Universel | Recommandations Goldilocks consultées, sizing ajusté ou refus documenté | - |
| Graceful shutdown | Universel | `preStop` hook + `terminationGracePeriodSeconds` requis sur tous les containers | `vixens.io/no-long-connections: "true"` si pas de connexions longues |
| PodDisruptionBudget | Contextuel | Si multi-replica / haute-disponibilité | - |
| topologySpreadConstraints / podAntiAffinity | Contextuel | Si multi-replica : éviter la concentration sur un seul nœud | - |
| HPA / KEDA | Contextuel | Si la charge est variable et le scaling automatique est pertinent | Opt-in: `vixens.io/needs-autoscaling: "true"` |

**Critère de passage:** L'application a une gestion explicite et réfléchie des ressources et des priorités.

---

## 🟢 Niveau 5 : Emerald — *"Data Durability"*

**Philosophie:** Les données de l'application survivent à une panne, un restart, un désastre.

| Prérequis | Type | Description | Bypass |
|-----------|------|-------------|--------|
| Backup profile défini | Universel | Label `vixens.io/backup-profile: critical\|standard\|relaxed\|ephemeral` | - |
| Ressources sidecars définies | Universel | CPU/RAM limités sur tous les sidecars si présents | - |
| Litestream sidecar + restore initContainer | Contextuel | Si SQLite — backup continu + restore automatique au démarrage | - |
| Config-Syncer sidecar + restore initContainer | Contextuel | Si configuration persistante non-SQLite | - |
| Velero backup confirmé | Contextuel | Si PVC — namespace inclus dans un schedule Velero actif | - |

**Critère de passage:** L'application peut être restaurée après sinistre.

---

## 💠 Niveau 6 : Diamond — *"Secure & Integrated"*

**Philosophie:** L'application est isolée, durcie et intégrée à l'écosystème. La sécurité arrive ici volontairement (homelab : faire fonctionner avant de sécuriser).

| Prérequis | Type | Description | Bypass |
|-----------|------|-------------|--------|
| PSA labels namespace (`baseline`) | Universel | `pod-security.kubernetes.io/enforce: baseline` sur le namespace | - |
| SecurityContext durci | Universel | `runAsNonRoot: true`, `allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]`, `seccompProfile: RuntimeDefault` | `vixens.io/explicitly-allow-root: "true"` |
| NetworkPolicies Cilium L3/L4 | Universel | Isolation réseau explicite — deny-all + allow sélectif | - |
| Authentik SSO | Contextuel | Si authentification utilisateur requise | `vixens.io/nossoneeded: "true"` |
| Cilium L7 policies | Contextuel | Si filtrage HTTP/gRPC requis | - |
| Image digest pinning (SHA) | Contextuel | Tag `image@sha256:...` — supply chain sécurisée (Renovate gère) | Opt-in: `vixens.io/digest-pinned: "true"` |
| Velero restore testé | Contextuel | Restore effectivement validé en conditions réelles (pas juste "configuré") | - |
| Homepage widget | Contextuel | Widget Dashboard Homepage configuré et fonctionnel | `vixens.io/nohomepage: "true"` |

**Critère de passage:** L'application est isolée, durcie et intégrée au système d'authentification.

---

## 🌟 Niveau 7 : Orichalcum — *"Parfaite"*

**Philosophie:** Il n'y a plus rien à faire. Tier "vision ultime" — une app Orichalcum est éprouvée, autonome et documentée à un niveau opérationnel professionnel.

| Prérequis | Type | Description | Bypass |
|-----------|------|-------------|--------|
| 7 jours de stabilité | Universel | Zéro restart, zéro OOMKill sur 7 jours consécutifs | - |
| Sizing validé | Universel | VPA recommendations appliquées + stabilité 7j, **OU** mode G-* stable 7j sans OOMKill | - |
| Zéro CVE HIGH/CRITICAL | Universel | Scan Trivy propre ou bypass documenté | `vixens.io/cve-accepted: "true"` |

**Critère de passage:** L'application est éprouvée, documentée et autonome.

---

## 💡 Notes Importantes

### Philosophie des bypasses

> Chaque check **bloque par défaut**. Passer un check = implémenter la fonctionnalité OU poser une annotation de bypass explicite. Un bypass est une **déclaration intentionnelle**, pas une fuite.

### Universal vs Contextuel

- **Universel** : S'applique à TOUTES les applications, quel que soit leur rôle
- **Contextuel** : S'applique uniquement si la condition est vraie pour l'application
  - Exemple : PVC + update strategy → applicable seulement si l'app a du stockage persistant
  - Exemple : HPA/KEDA → applicable seulement si charge variable identifiée

### Opt-in vs Opt-out

- **Opt-out** (défaut): Check actif par défaut, bypass pour désactiver (ex: `vixens.io/nometrics: "true"`)
- **Opt-in**: Check inactif par défaut, annotation pour activer (ex: `vixens.io/needs-autoscaling: "true"`)

### Progression séquentielle

```
Bronze → Silver → Gold → Platinum → Emerald → Diamond → Orichalcum
```

**Une application ne peut pas sauter un niveau.** Tous les prérequis du niveau N doivent être satisfaits avant de progresser au niveau N+1.

---

## 🔗 Références

- **[ADR-023: 7-Tier Goldification System v2](../adr/023-7-tier-goldification-system-v2.md)** — Source autoritaire
- **[Quality Standards](quality-standards.md)** — Vue d'ensemble du système de maturité
- **[Litestream Backup Profiles](../adr/014-litestream-backup-profiles-and-recovery-patterns.md)** — Stratégie backup SQLite

---

**Dernière mise à jour:** 2026-03-13  
**Version ADR-023:** v2.1 (2026-03-05)
