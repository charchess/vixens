# ADR-031: GitOps-only repository scope

**Date:** 2026-09-18  
**Status:** Accepted  
**Deciders:** Repository owner  
**Tags:** repository, gitops, documentation, tooling, workflow

## Context

Vixens accumulated several generations of tooling and documentation alongside the Kubernetes desired state: Beads task data, BMAD/Cline/Serena agent frameworks, agent-specific instructions, generated reports, legacy Infisical guidance, Terraform-era material and duplicate workflow descriptions.

Many of those artifacts contradicted the current GitHub/ArgoCD workflow. Because agents and humans could encounter them as apparently authoritative instructions, stale auxiliary content became an operational risk.

## Decision

Vixens is a GitOps desired-state repository.

Keep:

- `apps/` Kubernetes/Kustomize desired state;
- `argocd/` reconciliation definitions;
- GitHub workflows needed for validation, dependency maintenance and promotion;
- small static validation helpers;
- concise current documentation;
- immutable ADR and post-mortem history;
- application runbooks that add operational context;
- workload build context genuinely maintained by this repository.

Remove or keep elsewhere:

- Beads and other task databases;
- third-party generic agent frameworks/skills;
- Serena/project-local agent memory;
- Terraform/Talos provisioning material;
- hand-maintained live status and conformity reports;
- obsolete agent-specific workflow documents;
- local credentials and secret-management client configuration.

GitHub Issues are the task tracker. `WORKFLOW.md` is the workflow source of truth. Agent instructions reference canonical project documentation instead of copying it.

## Consequences

The repository becomes smaller and easier to audit, but some convenience automation may need to be recreated later when there is a demonstrated need.

Reintroduced tooling must serve the GitOps repository directly and must not establish a competing source of truth.
