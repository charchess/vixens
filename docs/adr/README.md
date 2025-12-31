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
  Renovate bot updates dev branch first for validation.

- **[ADR-008: Trunk-Based GitOps Workflow](008-trunk-based-gitops-workflow.md)**
  Two-branch strategy (dev/main) with trunk-based development.

- **[ADR-009: Simplified Two-Branch Workflow](009-simplified-two-branch-workflow.md)**
  Retire test/staging branches for simplified workflow.

### Architecture (Coming Soon)

- **ADR-010: Shared Resources Organization** 🚧
  Strategy for centralizing shared Kubernetes resources.

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
| 010 | Shared Resources Organization | Proposed | 2025-12 |
| 011 | Namespace Ownership Strategy | Proposed | 2025-12 |
| 012 | Middleware Management | Proposed | 2025-12 |

---

**Last Updated:** 2025-12-30
