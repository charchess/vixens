# Vixens Documentation

Welcome to the Vixens Kubernetes homelab documentation! This is your central hub for all project documentation.

---

## 🚀 Quick Start

**New to the project?**
1. Read [CLAUDE.md](../CLAUDE.md) for Claude Code guidance
2. Read [WORKFLOW.md](../WORKFLOW.md) for workflow rules
3. Follow [guides/adding-new-application.md](guides/adding-new-application.md) to deploy your first app

**Need to push to production?**
→ See [guides/gitops-workflow.md](guides/gitops-workflow.md)

**Managing tasks with Beads?**
→ See [guides/task-management.md](guides/task-management.md)

---

## 📊 Dashboards & Status

- **[Master Architecture (V4.0)](architecture.md)** - The technical blueprint for Goldification.
- **[Application Status Dashboard](STATUS.md)** - Real-time deployment status across environments.
- **[Functional Validation](reports/validation/RECETTE-FONCTIONNELLE.md)** - Latest functional test results.
- **[Technical Validation](reports/validation/RECETTE-TECHNIQUE.md)** - Latest technical audit results.

---

## 📚 Documentation Structure

### [guides/](guides/) - How-To Guides
Practical step-by-step guides for common tasks:
- **[Adding a New Application](guides/adding-new-application.md)** - Deploy apps to the cluster
- **[GitOps Workflow](guides/gitops-workflow.md)** - Push changes to production
- **[Production Promotion Workflow](guides/promotion-workflow.md)** - ⭐ Automated prod promotion (`just SendToProd`)
- **[Quality Reports](guides/quality-reports.md)** - ⭐ Complete reporting system (`just reports`)
- **[Task Management](guides/task-management.md)** - Beads workflow and formalism
- **[Secret Management](guides/secret-management.md)** - Working with Infisical
- **[Terraform Workflow](guides/terraform-workflow.md)** - Infrastructure changes

### [reference/](reference/) - Technical References
Detailed technical documentation:
- **[Workflow State Machine](reference/workflow-state-machine.md)** - Guide to Justfile Phases 0-6.
- **[Quality Standards](reference/quality-standards.md)** - Application quality tiers (Elite V4.0).
- **[Metrics Exemption](reference/metrics-exemption.md)** - How to exempt apps from metrics requirement (Gold tier).
- **[PodDisruptionBudget Components](reference/poddisruptionbudget-components.md)** - Standardized PDB configuration with Kustomize components.
- **[Multi-Agent Orchestration](reference/multi-agent-orchestration.md)** - Agent task assignment logic.
- **[ArgoCD Sync Waves](reference/argocd-sync-waves.md)** - Sync wave patterns.
- **[Resource Standards](reference/RESOURCE_STANDARDS.md)** - CPU/Memory allocation rules.
- **[Naming Conventions](reference/naming-conventions.md)** - Files, resources, namespaces.
Detailed technical documentation:
- **[Workflow State Machine](reference/workflow-state-machine.md)** - Guide to Justfile Phases 0-6.
- **[Quality Standards](reference/quality-standards.md)** - Application quality tiers (Elite V4.0).
- **[Metrics Exemption](reference/metrics-exemption.md)** - How to exempt apps from metrics requirement (Gold tier).
- **[Multi-Agent Orchestration](reference/multi-agent-orchestration.md)** - Agent task assignment logic.
- **[ArgoCD Sync Waves](reference/argocd-sync-waves.md)** - Sync wave patterns.
- **[Resource Standards](reference/RESOURCE_STANDARDS.md)** - CPU/Memory allocation rules.
- **[Naming Conventions](reference/naming-conventions.md)** - Files, resources, namespaces.
Detailed technical documentation:
- **[Workflow State Machine](reference/workflow-state-machine.md)** - Guide to Justfile Phases 0-6.
- **[Quality Standards](reference/quality-standards.md)** - Application quality tiers (Elite V4.0).
- **[Multi-Agent Orchestration](reference/multi-agent-orchestration.md)** - Agent task assignment logic.
- **[ArgoCD Sync Waves](reference/argocd-sync-waves.md)** - Sync wave patterns.
- **[Resource Standards](reference/RESOURCE_STANDARDS.md)** - CPU/Memory allocation rules.
- **[Naming Conventions](reference/naming-conventions.md)** - Files, resources, namespaces.

---

## 🔍 Finding What You Need

| I want to... | Go to... |
|--------------|----------|
| Add a new application | [guides/adding-new-application.md](guides/adding-new-application.md) |
| Push changes to production | [guides/gitops-workflow.md](guides/gitops-workflow.md) |
| **⭐ Promote to prod (automated)** | **[guides/promotion-workflow.md](guides/promotion-workflow.md)** |
| **⭐ Generate all reports** | **[guides/quality-reports.md](guides/quality-reports.md)** |
| Create/manage Beads tasks | [guides/task-management.md](guides/task-management.md) |
| Orchestrate multi-agent work | [reference/multi-agent-orchestration.md](reference/multi-agent-orchestration.md) |
| Understand sync waves | [reference/argocd-sync-waves.md](reference/argocd-sync-waves.md) |
| See naming conventions | [reference/naming-conventions.md](reference/naming-conventions.md) |
| Find app documentation | [applications/](applications/) |
| Understand an architecture decision | [adr/](adr/) |
| Troubleshoot an issue | [troubleshooting/common-issues.md](troubleshooting/common-issues.md) |
| Understand quality standards | [reference/quality-standards.md](reference/quality-standards.md) |
| Test multiple applications | [procedures/application-testing.md](procedures/application-testing.md) |
| Hibernate dev applications | [procedures/dev-hibernation.md](procedures/dev-hibernation.md) |
| See cluster status | [STATUS.md](STATUS.md) |
| Check validation results | [reports/validation/](reports/validation/) |

---

## 📖 Documentation Principles

1. **Single Source of Truth:** Information lives in ONE place, others LINK to it
2. **Discoverability First:** Find what you need in < 30 seconds
3. **Actionable:** Every guide has clear steps and acceptance criteria
4. **Up-to-Date:** Update docs when code changes (mandatory in PR reviews)
5. **Template-Driven:** Use templates for consistency

---

## 🤝 Contributing to Documentation

1. Use templates from [templates/](templates/)
2. Follow naming conventions in [reference/naming-conventions.md](reference/naming-conventions.md)
3. Link to existing docs instead of duplicating
4. Update this README if adding new major sections
5. Keep guides practical and procedures operational

---

## 📞 Getting Help

- **Claude Code:** Read [CLAUDE.md](../CLAUDE.md)
- **Workflow:** Read [WORKFLOW.md](../WORKFLOW.md)
- **Common Issues:** See [troubleshooting/common-issues.md](troubleshooting/common-issues.md)
- **Architecture Questions:** Check [adr/](adr/)
- **Task Management:** See [guides/task-management.md](guides/task-management.md)

---

**Last Updated:** 2026-01-11
