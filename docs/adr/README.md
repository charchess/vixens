# Architecture Decision Records (ADRs)

Architecture decisions and their rationale for the Vixens project.

---

## What is an ADR?

An **Architecture Decision Record** documents an important architectural decision made along with its context and consequences.

**When to create an ADR:**
- Significant architectural changes
- Technology/tool selections
- Design pattern adoptions
- Infrastructure strategy decisions
- Workflow changes affecting the team

---

## Available ADRs

### GitOps & Workflow

- **[ADR-007: Renovate Dev-First Workflow](007-renovate-dev-first-workflow.md)**
  Renovate bot updates dev branch first for validation. (⚠️ Needs update for single-branch)

- **[ADR-017: Pure Trunk-Based Development (Single Branch)](017-pure-trunk-based-single-branch.md)** ⭐
  Single `main` branch workflow. Dev watches main HEAD, Prod watches prod-stable tag. **Supersedes ADR-008/009.**

- ~~**[ADR-008: Trunk-Based GitOps Workflow](008-trunk-based-gitops-workflow.md)**~~ (Superseded by ADR-017)
  Two-branch strategy (dev/main) with trunk-based development.

- ~~**[ADR-009: Simplified Two-Branch Workflow](009-simplified-two-branch-workflow.md)**~~ (Superseded by ADR-017)
  Retire test/staging branches for simplified workflow.

### Infrastructure & Disaster Recovery

- **[ADR-010: Static Manifests for Infrastructure Apps](010-static-manifests-for-infrastructure-apps.md)**
  Use static manifests (not Helm) for infrastructure components.

- **[ADR-013: Layered Configuration Disaster Recovery](013-layered-configuration-disaster-recovery.md)**
  Disaster recovery strategy with layered configuration backups.

- **[ADR-013: Renovate Discord Approval Workflow](013-renovate-discord-approval-workflow.md)**
  Discord integration for Renovate PR approvals.

- **[ADR-014: Litestream Backup Profiles and Recovery Patterns](014-litestream-backup-profiles-and-recovery-patterns.md)**
  Light vs Heavy backup profiles for SQLite databases with standardized recovery patterns.

### Future Architecture

- **ADR-011: Namespace Ownership Strategy** 🚧
  Rules for namespace creation and ownership.

- **ADR-012: Middleware Management** 🚧
  Centralized vs distributed Traefik middleware strategy.

---

## ADR Status

| Status | Meaning |
|--------|---------|
| **Proposed** | Under discussion |
| **Accepted** | Decision made, implementation pending |
| **Implemented** | Fully implemented |
| **Deprecated** | No longer valid |
| **Superseded** | Replaced by another ADR |

---

## Creating a New ADR

1. Use [templates/adr-template.md](../templates/adr-template.md)
2. Number sequentially (e.g., `013-...`)
3. Follow naming: `NNN-short-title.md`
4. Update this README with link
5. Link from main [docs/README.md](../README.md)

---

## ADR Index

| Number | Title | Status | Date |
|--------|-------|--------|------|
| 007 | Renovate Dev-First Workflow | Implemented | 2024-12 |
| 008 | Trunk-Based GitOps Workflow | Implemented | 2024-12 |
| 009 | Simplified Two-Branch Workflow | Implemented | 2024-12 |
| 010 | Static Manifests for Infrastructure Apps | Accepted | 2025-01 |
| 011 | Namespace Ownership Strategy | Proposed | TBD |
| 012 | Middleware Management | Proposed | TBD |
| 013 | Layered Configuration Disaster Recovery | Accepted | 2026-01 |
| 013 | Renovate Discord Approval Workflow | Accepted | 2026-01 |
| 014 | Litestream Backup Profiles and Recovery Patterns | Accepted | 2026-01 |

---

**Last Updated:** 2026-01-10
