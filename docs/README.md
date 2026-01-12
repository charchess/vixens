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

- **[Application Status Dashboard](STATUS.md)** - Real-time deployment status across environments
- **[Functional Validation](reports/validation/RECETTE-FONCTIONNELLE.md)** - Latest functional test results
- **[Technical Validation](reports/validation/RECETTE-TECHNIQUE.md)** - Latest technical audit results

---

## 📚 Documentation Structure

### [guides/](guides/) - How-To Guides
Practical step-by-step guides for common tasks:
- **[Adding a New Application](guides/adding-new-application.md)** - Deploy apps to the cluster
- **[GitOps Workflow](guides/gitops-workflow.md)** - Push changes to production
- **[Task Management](guides/task-management.md)** - Beads workflow and formalism
- **[Secret Management](guides/secret-management.md)** - Working with Infisical
- **[Terraform Workflow](guides/terraform-workflow.md)** - Infrastructure changes

### [reference/](reference/) - Technical References
Detailed technical documentation:
- **[Multi-Agent Orchestration](reference/multi-agent-orchestration.md)** - Multi-agent task orchestration (Claude, Gemini, coding-agent)
- **[ArgoCD Sync Waves](reference/argocd-sync-waves.md)** - Sync wave patterns
- **[Task Formalism](reference/task-formalism.md)** - Conventional commit task format
- **[Kustomize Patterns](reference/kustomize-patterns.md)** - Common Kustomize patterns
- **[Overlay Strategy](reference/overlay-strategy.md)** - Dev/Prod overlay strategy
- **[Naming Conventions](reference/naming-conventions.md)** - Files, resources, namespaces
- **[Quality Standards](reference/quality-standards.md)** - Application quality tiers (Bronze to Elite)

### [applications/](applications/) - Application Documentation
Documentation for all deployed applications, organized by category:
- **[00-infra/](applications/00-infra/)** - Infrastructure (ArgoCD, Traefik, Cilium, Cert-Manager)
- **[02-monitoring/](applications/02-monitoring/)** - Monitoring stack (Prometheus, Grafana, Loki)
- **[10-databases/](applications/10-databases/)** - Database services (PostgreSQL, Redis)
- **[20-media/](applications/20-media/)** - Media applications (Jellyfin, *arr stack)
- **[40-network/](applications/40-network/)** - Network services (AdGuard, External-DNS)
- **[50-services/](applications/50-services/)** - General services (Home Assistant, Vaultwarden)
- **[70-tools/](applications/70-tools/)** - Tools & utilities (Homepage, Linkwarden)

### [procedures/](procedures/) - Operational Procedures
Step-by-step operational procedures:
- **[Deployment Standard](procedures/deployment-standard.md)** - Standard deployment process
- **[Backup & Restore](procedures/backup-restore.md)** - Backup/restore procedures
- **[Disaster Recovery](procedures/disaster-recovery.md)** - DR procedures
- **[Cluster Upgrade](procedures/cluster-upgrade.md)** - Upgrade procedures
- **[Secret Rotation](procedures/secret-rotation.md)** - Secret rotation procedures
- **[Application Testing](procedures/application-testing.md)** - Testing new applications
- **[Dev Hibernation](procedures/dev-hibernation.md)** - Hibernating dev environment apps

### [adr/](adr/) - Architecture Decision Records
Architecture decisions and their rationale:
- **[ADR Index](adr/README.md)** - Complete list of ADRs
- **[ADR-007](adr/007-renovate-trunk-based-workflow.md)** - Renovate dev-first workflow
- **[ADR-008](adr/008-trunk-based-gitops-workflow.md)** - Trunk-based GitOps
- **[ADR-009](adr/009-simplified-two-branch-workflow.md)** - Two-branch workflow
- **[ADR-010](adr/010-shared-resources-organization.md)** - Shared resources organization
- **[ADR-011](adr/011-namespace-ownership-strategy.md)** - Namespace ownership
- **[ADR-012](adr/012-middleware-management.md)** - Middleware management

### [reports/](reports/) - Analysis Reports
Technical reports, audits, and validation:
- **[Validation Reports](reports/validation/)** - Functional and technical validation logs
- **[Audits & Analysis](reports/audits/)** - Resource, conformity, and architecture audits
- **[2025-12-30 Code Review](reports/2025-12-30-code-review.md)** - Architecture review
- **[2025-12-30 Archon Migration](reports/2025-12-30-archon-migration-summary.md)** - Task migration

### [troubleshooting/](troubleshooting/) - Troubleshooting
Incident logs, post-mortems and common issues:
- **[Common Issues](troubleshooting/common-issues.md)** - Quick fixes for common problems
- **[Post-Mortems](troubleshooting/post-mortems/)** - Detailed incident analyses (e.g., [Cluster Reset 2026-01-05](troubleshooting/post-mortems/2026-01-05-cluster-reset.md))

### [templates/](templates/) - Document Templates
Templates for creating new documentation:
- **[ADR Template](templates/adr-template.md)** - Architecture decision record
- **[Application Doc Template](templates/application-doc-template.md)** - Application documentation
- **[Procedure Template](templates/procedure-template.md)** - Operational procedure
- **[Troubleshooting Template](templates/troubleshooting-template.md)** - Troubleshooting guide

---

## 🔍 Finding What You Need

| I want to... | Go to... |
|--------------|----------|
| Add a new application | [guides/adding-new-application.md](guides/adding-new-application.md) |
| Push changes to production | [guides/gitops-workflow.md](guides/gitops-workflow.md) |
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
