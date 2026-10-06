# Architecture Decision Records — authoritative index

**Last Updated:** 2026-10-02  
**Governance:** [ADR-030](030-project-scoped-adr-governance.md)  
**Usage guide:** [README.md](README.md)

This catalog is the authoritative inventory of ADR files in the repository. It normalizes status and assigns architectural scope/planning ownership without rewriting historical records merely to add metadata.

## Scope legend

- **Repository** — cross-monorepo development/governance decisions
- **Core** — shared infrastructure/platform foundation
- **AIaaS** — TXO Fabric tenant/agent product architecture
- **Personal** — personal/application architecture consuming Core

Planning projects are `vixens core`, `vixens roadmap`, and `vixens perso`.

## Current decisions

| File / ADR | Decision | Catalog status | Scope | Related Project |
|---|---|---|---|---|
| [001](001-choix-architecture-initiale.md) | Choix Architecture Initiale | Accepted | Core | vixens core |
| [002](002-argocd-gitops.md) | ArgoCD GitOps | Accepted | Core | vixens core |
| [003](003-vlan-segmentation.md) | VLAN Segmentation | Accepted | Core | vixens core |
| [004](004-cilium-cni.md) | Cilium CNI | Accepted | Core | vixens core |
| [005](005-cilium-l2-announcements.md) | Cilium L2 Announcements | Accepted | Core | vixens core |
| [006](006-terraform-3-level-architecture-REVISED.md) | Terraform 3-Level Architecture | Accepted | Core | vixens core |
| [007](007-renovate-trunk-based-workflow.md) | Renovate Trunk-Based Workflow | Accepted | Repository | vixens core |
| [012](012-monitoring-modular-approach.md) | Modular Monitoring Approach | Accepted | Core | vixens core |
| [013](013-layered-configuration-disaster-recovery.md) | Layered Configuration & Disaster Recovery | Accepted | Core | vixens core |
| [014](014-litestream-backup-profiles-and-recovery-patterns.md) | Litestream Backup Profiles & Recovery Patterns | Accepted | Core | vixens core |
| [017](017-pure-trunk-based-single-branch.md) | Pure Trunk-Based Development | Accepted | Repository | vixens core |
| [018](018-openbao-external-secrets-and-nas-fqdn.md) † | OpenBao / External Secrets and NAS FQDN | Accepted | Core | vixens core |
| [019 file](019-renovate-discord-approval-workflow.md) ‡ | Renovate Discord Approval Workflow | Deprecated | Repository | vixens core |
| [020](020-automated-housekeeping-sanitization.md) | Automated Housekeeping | Accepted | Repository | vixens core |
| [021](021-netbird-native-manifests.md) | Netbird Native Manifests | Accepted | Core | vixens core |
| [023](023-7-tier-goldification-system-v2.md) | 7-Tier Goldification System v2 | Accepted (`Active` in historical file) | Repository | vixens core |
| [024](024-sso-debt-diamond-wave7.md) | SSO Debt Register: Diamond Wave 7 | Accepted (`Active` in historical file) | Core | vixens core |
| [025](025-local-path-provisioner.md) | Local Path Provisioner | Accepted | Core | vixens core |
| [026](026-keda-scale-to-zero.md) | KEDA HTTP Add-on for Scale-to-Zero | Accepted | Core | vixens core |
| [027](027-retire-openclaw-and-ollama-telemetry.md) | Retire OpenClaw and Ollama Telemetry | Accepted | Core | vixens core |
| [028](028-retire-dataangel-prod-zfs.md) | Retire DataAngel Restore Init from Production Workloads | Accepted | Core | vixens core |
| [029](029-align-maturity-with-current-platform.md) | Align application maturity with the current platform | Accepted | Repository | vixens core |
| [030](030-project-scoped-adr-governance.md) | Project-scoped ADR governance | Accepted | Repository | all three Projects |
| [031](031-agentidentity-is-the-fabric-runtime-identity.md) | AgentIdentity is the Fabric runtime agent identity | Accepted | AIaaS | vixens roadmap |
| [032](032-fabric-is-tenant-neutral.md) | TXO Fabric is tenant-neutral and valid with zero clients | Accepted | AIaaS | vixens roadmap |
| [033](033-separate-agent-state-workspace-and-skills.md) | Separate private agent state, shared business data and shared skills | Accepted | AIaaS | vixens roadmap |
| [034](034-immutable-hermes-runtime-supply-chain.md) | Build Hermes extensions into an immutable TXO runtime image | Accepted | AIaaS | vixens roadmap |
| [035](035-centralized-fabric-inference-boundary.md) | Centralize Fabric external inference behind the TXO AI Gateway | Accepted | AIaaS | vixens roadmap |
| [036](036-hindsight-external-durable-memory-boundary.md) | Keep durable agent memory behind the tenant Hindsight service boundary | Accepted | AIaaS | vixens roadmap |
| [037](037-skills-vs-platform-governed-executable-capabilities.md) | Separate agent-extensible skills from platform-governed executable capabilities | Accepted | AIaaS | vixens roadmap |

## Historical / superseded decisions

| File / ADR | Decision | Catalog status | Replaced by / note | Scope | Related Project |
|---|---|---|---|---|---|
| [008](008-trunk-based-gitops-workflow.md) | Trunk-Based GitOps Workflow | Superseded | [ADR-017](017-pure-trunk-based-single-branch.md) | Repository | vixens core |
| [009](009-simplified-two-branch-workflow.md) | Simplified Two-Branch Workflow | Superseded | [ADR-017](017-pure-trunk-based-single-branch.md) | Repository | vixens core |
| [011 file](011-infisical-secrets-management.md) § | Infisical Secrets Management | Superseded | current OpenBao/ESO architecture ([historical ADR-018 collision](018-openbao-external-secrets-and-nas-fqdn.md)) | Core | vixens core |
| [016](016-workflow-master-reference.md) | Workflow Master Reference | Superseded | [`WORKFLOW.md`](../../WORKFLOW.md) for living workflow contract | Repository | vixens core |
| [018](018-netbird-deployment-architecture.md) † | Netbird Helm deployment architecture | Superseded | [ADR-021](021-netbird-native-manifests.md) | Core | vixens core |
| [022](022-7-tier-goldification-system.md) | 7-Tier Goldification System v1 | Superseded | [ADR-023](023-7-tier-goldification-system-v2.md) | Repository | vixens core |

## Deprecated decisions

| File / ADR | Decision | Catalog status | Reason / replacement | Scope | Related Project |
|---|---|---|---|---|---|
| [010](010-static-manifests-for-infrastructure-apps.md) | Static/Hydrated Manifests for Infrastructure Apps | Deprecated | historical approach no longer canonical | Core | vixens core |
| [015](015-conformity-scoring-grid.md) | Conformity Scoring Grid | Deprecated | replaced by the 7-tier maturity model | Repository | vixens core |

## Historical identity anomalies

These anomalies predate ADR-030 and are preserved so existing links/history remain valid. They are **not** precedents for new ADR numbering.

- **† ADR-018 collision:** both `018-netbird-deployment-architecture.md` and `018-openbao-external-secrets-and-nas-fqdn.md` carry the number 018. The Netbird ADR is superseded by ADR-021; the OpenBao/ESO ADR is the current secrets architecture.
- **‡ Renovate Discord header collision:** `019-renovate-discord-approval-workflow.md` is the repository file identity, but its historical document header says `ADR-013`. The separate `013-layered-configuration-disaster-recovery.md` also exists. The file is cataloged here by its stable path and is not renumbered retroactively.
- **§ Infisical restored-file mismatch:** `011-infisical-secrets-management.md` is the stable repository path, while its restored historical header says `ADR 007`. It is superseded and retained only as history.

No new ADR may reuse an allocated number. The next ADR after this catalog is **038** unless another PR allocates it first.

## Project view

The catalog currently contains Core and Repository decisions plus the first AIaaS/Fabric decisions. No durable Personal-scoped ADR is created merely for symmetry; `vixens perso` receives ADRs when an actual personal-application architecture decision warrants one.

Unresolved TXO Fabric candidates remain with their owning issues (notably #3687, #3689 and later #3688). A proposal in an issue is not promoted into this Accepted catalog until implementation/acceptance establishes the decision.
