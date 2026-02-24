# Architecture Decision Records - Index

**Last Updated:** 2026-01-17

---

## 🏆 Active Decisions (Accepted)

Ces documents représentent l'état actuel et obligatoire de l'architecture Vixens.

| ADR | Title | Date | Tags |
|-----|-------|------|------|
| [001](001-choix-architecture-initiale.md) | Choix Architecture Initiale | 2025-11-15 | infra, talos, cilium |
| [002](002-argocd-gitops.md) | ArgoCD GitOps | 2025-11-15 | gitops, argocd |
| [003](003-vlan-segmentation.md) | VLAN Segmentation | 2025-11-15 | networking, vlan |
| [004](004-cilium-cni.md) | Cilium CNI | 2025-11-15 | networking, cilium, cni |
| [005](005-cilium-l2-announcements.md) | Cilium L2 Announcements | 2025-11-15 | networking, cilium, lb |
| [006](006-terraform-3-level-architecture-REVISED.md) | Terraform 3-Level Architecture | 2025-11-15 | infra, terraform |
| [007](007-renovate-trunk-based-workflow.md) | Renovate Trunk-Based Workflow | 2026-01-11 | renovate, gitops |
| [011](011-infisical-secrets-management.md) | Infisical Secrets Management | 2025-11-16 | security, secrets |
| [012](012-monitoring-modular-approach.md) | Modular Monitoring Approach | 2025-12-04 | monitoring, prometheus |
| [013](013-layered-configuration-disaster-recovery.md) | Layered Config & Disaster Recovery | 2026-01-05 | backup, dr |
| [014](014-litestream-backup-profiles-and-recovery-patterns.md) | Litestream Backup Profiles | 2026-01-10 | backup, litestream, sqlite |
| [017](017-pure-trunk-based-single-branch.md) | **Pure Trunk-Based (Master Workflow)** | 2026-01-11 | gitops, workflow |
| [019](019-renovate-discord-approval-workflow.md) | Renovate Discord Approval | 2026-01-02 | automation, renovate |
| [020](020-automated-housekeeping-sanitization.md) | Automated Housekeeping | 2026-01-17 | infra, sanitization |
| [021](021-netbird-native-manifests.md) | Netbird Native Manifests | 2026-01-17 | networking, netbird |
| [022](022-7-tier-goldification-system.md) | 7-Tier Goldification System | 2026-02-24 | quality, goldification |

---

## 📜 Historical / Superseded Decisions

Documents conservés pour l'audit trail mais remplacés par des décisions plus récentes.

| ADR | Title | Status | Superseded by |
|-----|-------|--------|---------------|
| [008](008-trunk-based-gitops-workflow.md) | Trunk-Based Migration (4->2) | Superseded | [ADR-017](017-pure-trunk-based-single-branch.md) |
| [009](009-simplified-two-branch-workflow.md) | Simplified Two-Branch Workflow | Superseded | [ADR-017](017-pure-trunk-based-single-branch.md) |
| [016](016-workflow-master-reference.md) | Workflow Master Reference | Superseded | [WORKFLOW.md](../../WORKFLOW.md) |
| [018](018-netbird-deployment-architecture.md) | Netbird (Helm Chart) | Superseded | [ADR-021](021-netbird-native-manifests.md) |

---

## 🗑️ Deprecated Decisions

Décisions abandonnées ou inactives.

| ADR | Title | Status | Reason |
|-----|-------|--------|--------|
| [010](010-static-manifests-for-infrastructure-apps.md) | Static Manifests for Infra | Deprecated | Manual maintenance too high |
| [015](015-conformity-scoring-grid.md) | Conformity Scoring Grid | Deprecated | Superseded by [ADR-022](022-7-tier-goldification-system.md) |