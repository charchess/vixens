# Architecture Decision Records — canonical index

This file is the registry for Vixens ADRs.

## Immutability rule

An ADR records the decision and rationale that existed at the time. Accepted, deprecated and superseded ADRs are historical records and are **not rewritten to match current architecture**.

When a decision changes, create a new ADR and mark the relationship here. Old records and incident history remain intact.

## Current high-level decisions

| ADR | Decision |
|---|---|
| [017](017-pure-trunk-based-single-branch.md) | Single long-lived `main`; dev tracks `main`, prod tracks `prod-stable` |
| [021](021-netbird-native-manifests.md) | Netbird uses native manifests rather than the former chart architecture |
| [023](023-7-tier-goldification-system-v2.md) | Current maturity model |
| [024](024-sso-debt-diamond-wave7.md) | SSO debt/bypass decision |
| [025](025-local-path-provisioner.md) | Node-local storage option |
| [026](026-keda-scale-to-zero.md) | KEDA scale-to-zero architecture |
| [027](027-retire-openclaw-and-ollama-telemetry.md) | Retire OpenClaw/Ollama telemetry |
| [028](028-retire-dataangel-prod-zfs.md) | Retire DataAngel automatic restore from production workloads |
| [029](029-openbao-external-secrets-and-nas-fqdn.md) | OpenBao + External Secrets and NAS FQDN are canonical |
| [030](030-argocd-anonymous-admin-accepted-risk.md) | Anonymous ArgoCD admin access is an explicitly accepted homelab risk |
| [031](031-gitops-only-repository-scope.md) | Vixens is a GitOps-only state repository with minimal supporting documentation/tooling |

## Supersession / deprecation map

| Record | Relationship |
|---|---|
| ADR-008 | superseded by ADR-017 |
| ADR-009 | superseded by ADR-017 |
| ADR-010 | deprecated in its own record |
| ADR-011 | historical Infisical decision; superseded by ADR-029 |
| ADR-015 | deprecated; maturity model replaced by ADR-022/023 |
| ADR-016 | superseded by `WORKFLOW.md` / ADR-017 |
| ADR-018 Netbird | superseded by ADR-021 |
| ADR-018 OpenBao | historical numbering collision; preserved verbatim and superseded by ADR-029 |
| ADR-022 | superseded by ADR-023 |

## Complete preserved record set

The repository intentionally preserves the original records:

- [001](001-choix-architecture-initiale.md)
- [002](002-argocd-gitops.md)
- [003](003-vlan-segmentation.md)
- [004](004-cilium-cni.md)
- [005](005-cilium-l2-announcements.md)
- [006](006-terraform-3-level-architecture-REVISED.md)
- [007](007-renovate-trunk-based-workflow.md)
- [008](008-trunk-based-gitops-workflow.md)
- [009](009-simplified-two-branch-workflow.md)
- [010](010-static-manifests-for-infrastructure-apps.md)
- [011](011-infisical-secrets-management.md)
- [012](012-monitoring-modular-approach.md)
- [013](013-layered-configuration-disaster-recovery.md)
- [014](014-litestream-backup-profiles-and-recovery-patterns.md)
- [015](015-conformity-scoring-grid.md)
- [016](016-workflow-master-reference.md)
- [017](017-pure-trunk-based-single-branch.md)
- [018 Netbird](018-netbird-deployment-architecture.md)
- [018 OpenBao legacy record](018-openbao-external-secrets-and-nas-fqdn.md)
- [019](019-renovate-discord-approval-workflow.md)
- [020](020-automated-housekeeping-sanitization.md)
- [021](021-netbird-native-manifests.md)
- [022](022-7-tier-goldification-system.md)
- [023](023-7-tier-goldification-system-v2.md)
- [024](024-sso-debt-diamond-wave7.md)
- [025](025-local-path-provisioner.md)
- [026](026-keda-scale-to-zero.md)
- [027](027-retire-openclaw-and-ollama-telemetry.md)
- [028](028-retire-dataangel-prod-zfs.md)
- [029](029-openbao-external-secrets-and-nas-fqdn.md)
- [030](030-argocd-anonymous-admin-accepted-risk.md)
- [031](031-gitops-only-repository-scope.md)

The two historical files numbered 018 are not renamed or edited: the collision itself is part of repository history. ADR-029 provides the canonical successor for the OpenBao decision.
