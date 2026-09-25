# Guides

Practical how-to guides for common tasks in the Vixens project.

---

## Available Guides

### Core Workflows

- **[Adding a New Application](adding-new-application.md)** ⭐
  Deploy new applications to the cluster using the current GitOps conventions.

- **[GitOps Workflow](gitops-workflow.md)** ⭐
  Complementary GitOps details. `WORKFLOW.md` is authoritative for the current Issue → PR → dev → production workflow.

- **[Task Management](task-management.md)** ⭐
  Track work with GitHub Issues, labels, pull requests, and explicit dependencies.

### Infrastructure & Operations

- **[Secret Management](secret-management.md)** ⭐
  OpenBao + External Secrets Operator (`ClusterSecretStore/openbao` → `ExternalSecret` → Kubernetes `Secret`).

- **[Terraform Workflow](terraform-workflow.md)**
  Infrastructure changes in TerraVixens. Keep Vixens focused on the application/GitOps state unless a cross-repo change is explicitly required.

- **[Production Hibernation](prod-hibernation.md)**
  Production workload hibernation policy and operational constraints.

- **[Troubleshooting Guide](troubleshooting-guide.md)**
  Common diagnostic paths. Prefer current manifests and runbooks over historical examples when they disagree.

---

## Canonical Sources

Avoid duplicating workflow rules across guides:

- `WORKFLOW.md` — repository workflow and production promotion;
- `AGENTS.md` — constraints for humans and automation agents;
- `docs/guides/secret-management.md` — current secrets architecture;
- `docs/adr/` — architectural decisions and superseded history;
- GitHub Issues / PRs — task state and current change history.

Historical ADRs, audits and troubleshooting reports may intentionally mention retired tooling such as Infisical, Beads or Serena. They are not current operating instructions.

---

## Guide Structure

Each guide should contain, when relevant:
1. **Overview** — what the guide covers
2. **Prerequisites** — what is required before starting
3. **Step-by-step instructions** — reproducible workflow
4. **Validation** — how to verify success
5. **Troubleshooting** — common failure modes
6. **Related documentation** — links to canonical sources

---

## Contributing

When creating or updating guides:
1. Start from the current `main` branch and inspect open PRs first.
2. Reuse current repository patterns instead of copying historical examples.
3. Do not create a second source of truth for workflow or task state.
4. Include clear acceptance/validation criteria for operational changes.
5. Add the guide here when it is intended as active documentation.

---

**Last Updated:** 2026-09-25
