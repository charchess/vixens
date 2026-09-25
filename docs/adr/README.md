# Architecture Decision Records (ADRs)

Architecture decisions and their rationale for the Vixens project.

## Usage

An ADR records a durable architectural decision. Active operational instructions should live in the current guides/runbooks; historical ADRs are preserved even when superseded.

When an ADR and a newer accepted decision conflict, the newer decision wins. Current manifests remain the implementation source of truth.

## Important current decisions

### GitOps & workflow

- **[ADR-017: Pure Trunk-Based Development](017-pure-trunk-based-single-branch.md)** — single `main`; dev follows main; prod follows the promoted stable ref.
- Historical ADR-008/009 are superseded by ADR-017.

### Application maturity

- **[ADR-023: 7-Tier Goldification System v2](023-7-tier-goldification-system-v2.md)** — maturity tier intent and scoring model.
- **[ADR-029: Align application maturity with the current platform](029-align-maturity-with-current-platform.md)** — amends ADR-023 for OpenBao/ESO secrets and explicit resource fallbacks.

### Secrets

- **[ADR-018: OpenBao / External Secrets and NAS FQDN](018-openbao-external-secrets-and-nas-fqdn.md)** — current secret architecture.
- **[ADR-011: Infisical Secrets Management](011-infisical-secrets-management.md)** — historical/superseded secret architecture.

### Storage / recovery

- **[ADR-013: Layered Configuration Disaster Recovery](013-layered-configuration-disaster-recovery.md)**
- **[ADR-014: Litestream Backup Profiles and Recovery Patterns](014-litestream-backup-profiles-and-recovery-patterns.md)**
- **[ADR-025: Local Path Provisioner](025-local-path-provisioner.md)**
- **[ADR-028: Retire DataAngel Restore Init from Production Workloads](028-retire-dataangel-prod-zfs.md)**

### Networking / applications

- **[ADR-021: Netbird Native Manifests](021-netbird-native-manifests.md)** — supersedes the older Netbird Helm design.
- **[ADR-026: KEDA Scale-to-Zero](026-keda-scale-to-zero.md)**
- **[ADR-027: Retire OpenClaw and Ollama Telemetry](027-retire-openclaw-and-ollama-telemetry.md)**

## Historical numbering note

The repository contains older ADR numbering collisions (notably `013` and `018`). They are preserved to avoid rewriting history and breaking existing links. New ADRs continue from the highest allocated number.

Do not infer chronology or precedence from the number alone; use each ADR's date/status and explicit supersession links.

## Status values

| Status | Meaning |
|---|---|
| Proposed | Under discussion |
| Accepted | Decision made/current unless superseded |
| Active | Current operational architecture/standard |
| Implemented | Decision fully implemented |
| Deprecated | No longer recommended |
| Superseded | Replaced by a newer decision |

## Creating a new ADR

1. Start from the current `main` and check for concurrent PRs.
2. Use `docs/templates/adr-template.md` when appropriate.
3. Allocate the next unused number; do not renumber historical ADRs.
4. State status, date, context, decision and consequences.
5. Explicitly link decisions being amended/superseded.
6. Update this README.
7. Submit through the normal branch → PR → CI workflow.

## Compact index

| Number | Decision | Status |
|---:|---|---|
| 007 | Renovate Trunk-Based Workflow | historical/current only where not superseded |
| 008 | Trunk-Based GitOps Workflow | Superseded |
| 009 | Simplified Two-Branch Workflow | Superseded |
| 010 | Static Manifests for Infrastructure Apps | Accepted |
| 011 | Infisical Secrets Management | Superseded by current OpenBao/ESO architecture |
| 013 | Layered Configuration Disaster Recovery | Accepted |
| 013 | Renovate Discord Approval Workflow | Accepted (historical numbering collision) |
| 014 | Litestream Backup Profiles and Recovery Patterns | Accepted |
| 017 | Pure Trunk-Based Development | Active |
| 018 | Netbird Helm deployment architecture | Superseded by ADR-021 |
| 018 | OpenBao / External Secrets and NAS FQDN | Accepted (historical numbering collision) |
| 020 | Automated Housekeeping | Accepted |
| 021 | Netbird Native Manifests | Accepted |
| 022 | 7-Tier Goldification v1 | Superseded by ADR-023 |
| 023 | 7-Tier Goldification v2 | Active |
| 024 | SSO Debt Diamond Wave 7 | Active/Accepted |
| 025 | Local Path Provisioner | Accepted |
| 026 | KEDA Scale-to-Zero | Accepted |
| 027 | Retire OpenClaw and Ollama Telemetry | Accepted |
| 028 | Retire DataAngel Restore Init from Production Workloads | Accepted |
| 029 | Align application maturity with current platform | Accepted |

**Last Updated:** 2026-09-25
