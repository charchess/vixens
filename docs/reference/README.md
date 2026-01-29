# Reference Documentation

Technical references and specifications for the Vixens project.

---

## Available References

### Architecture & Standards

- **[Vixens Master Architecture](../architecture.md)**
  The technical blueprint for the V4.0 Goldification campaign.

- **[Quality Standards](quality-standards.md)**
  Definition of Bronze, Silver, Gold, Platinum, and Elite tiers.

- **[Resource Standards](RESOURCE_STANDARDS.md)**
  Technical standards for CPU and Memory allocation.

- **[Application Scoring Model](APPLICATION_SCORING_MODEL.md)**
  The logic behind conformity scoring.

### Workflows & Orchestration

- **[Workflow State Machine](workflow-state-machine.md)**
  Detailed guide of the Phases 0-6 managed by `justfile`.

- **[Multi-Agent Orchestration](multi-agent-orchestration.md)**
  Agent selection and task assignment logic (Claude, Gemini, etc.).

- **[Task Formalism](task-formalism.md)**
  Conventional commit-based task format.

### K8s & GitOps Patterns

- **[ArgoCD Sync Waves](argocd-sync-waves.md)**
  Sync wave ordering and dependencies.

- **[Application Deployment Standard](application-deployment-standard.md)**
  The canonical way to structure an application manifest.

- **[Sync Waves Implementation Plan](sync-waves-implementation-plan.md)**
  Historical plan for sync wave rollout.

---

## Using References

References are **technical specifications** meant for lookup and deep understanding. For practical how-to guides, see [guides/](../guides/).

**Reference vs Guide:**
- **Reference:** "Here's how sync waves work technically"
- **Guide:** "Here's how to add sync waves to your app"

---

**Last Updated:** 2025-12-30
