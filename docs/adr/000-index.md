# Architecture Decision Records - Index

**Last Updated**: 2025-11-21

---

## Active ADRs

| ADR | Title | Status | Date | Related OpenSpec |
|-----|-------|--------|------|------------------|
| [001](001-choix-architecture-initiale.md) | Choix Architecture Initiale | Active | 2025-11 | - |
| [002](002-argocd-gitops.md) | ArgoCD GitOps | Active | 2025-11 | - |
| [003](003-vlan-segmentation.md) | VLAN Segmentation | Active | 2025-11 | - |
| [004](004-cilium-cni.md) | Cilium CNI | Active | 2025-11 | - |
| [005](005-cilium-l2-announcements.md) | Cilium L2 Announcements | Active | 2025-11 | - |
| [006](006-terraform-3-level-architecture-REVISED.md) | Terraform 3-Level Architecture | Active | 2025-11 | - |
| [007](007-infisical-secrets-management.md) | Infisical Secrets Management | Active | 2025-11-20 | [propagate-infisical-multi-env](../../openspec/changes/propagate-infisical-multi-env/) |

---

## Superseded ADRs

None yet.

---

## ADR Categories

### Infrastructure
- ADR 001: Architecture initiale
- ADR 003: VLAN segmentation
- ADR 006: Terraform architecture

### GitOps & Deployment
- ADR 002: ArgoCD GitOps

### Networking
- ADR 004: Cilium CNI
- ADR 005: Cilium L2 Announcements

### Security & Secrets
- ADR 007: Infisical Secrets Management

---

## Creating New ADRs

See [Documentation Hierarchy](../DOCUMENTATION-HIERARCHY.md) for guidelines on when to create ADRs and how they relate to OpenSpecs.

**Template Structure**:
```markdown
# ADR XXX: Title

**Status**: Draft | Active | Superseded
**Date**: YYYY-MM-DD
**Related OpenSpec**: Link if applicable

## Context
Why this decision is needed

## Decision
What we decided

## Consequences
Impact and trade-offs

## Related
- Other ADRs
- OpenSpecs
- Documentation
```

---

## Reference

All ADRs follow the format defined in [Documentation Hierarchy](../DOCUMENTATION-HIERARCHY.md#documentation-docs).
