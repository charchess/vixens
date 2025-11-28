# Gemini Instructions for Vixens Project

This document provides essential context and instructions for AI agents working on the Vixens GitOps project.

## 1. Project Overview

The Vixens project aims to create a **resilient homelab** for managing HomeAssistant and other personal services. It emphasizes applying enterprise best practices (Infrastructure as Code, GitOps, progressive security) with a focus on being both **production-ready** in design and **pedagogical** in approach, despite having a zero budget and a single developer.

### Key Technologies:
- **Operating System**: Talos Linux v1.7.6 (immutable, no SSH)
- **Kubernetes**: v1.31.4 (embedded in Talos)
- **Infrastructure Provisioning**: Terraform (version to be determined, managed via `terraform-version.tf` or similar) with Talos provider
- **Container Network Interface (CNI)**: Cilium 1.16.5 (with native Load Balancer)
- **Ingress Controller**: Traefik 33.0.0
- **GitOps**: ArgoCD v7.6.12 (3-layer approach: system → security → apps)
- **Storage**: Synology CSI (iSCSI) and NFS (deprecated)
- **Secrets Management**: Infisical Operator
- **Certificates**: cert-manager with Gandi webhook (DNS-01)
- **Configuration Management**: Kustomize for environment-specific overlays

### Architecture:
The project employs two distinct control loops:
- **Application Loop (Fast & Automated)**: Developers push to GitHub, ArgoCD manages Kubernetes applications on various environments (dev, test).
- **Infrastructure Loop (Slow & Manual)**: `terraform apply` provisions and manages cluster infrastructure for different environments.

### Hardware Summary:
- **Hypervisor**: Hyper-V on Windows Server 2022 (3 dev/test/staging nodes)
- **Bare Metal**: NiPoGi mini PCs, Intel N150, 16GB DDR4, 512GB NVMe (3 prod nodes)
- **NAS**: Synology DS1821+, DSM 7.2.2-72806 Update 4 (iSCSI + MinIO backend S3)
- **Network**: UniFi Dream Machine SE (VLANs 111, 200, 208-210)
- **UPS**: Infosec E3 (currently inoperative)

## 2. Building and Running

### Prerequisites:
- Terraform
- kubectl
- talosctl

### Installation & Usage (Example for `dev` environment):
1.  **Clone the repository**:
    ```bash
    git clone https://github.com/charchess/vixens.git
    cd vixens
    ```
2.  **Navigate to environment and review `terraform.tfvars` (if applicable)**:
    ```bash
    cd terraform/environments/dev
    # Review or create terraform.tfvars
    ```
3.  **Initialize and apply Terraform (from project root)**:
    ```bash
    terraform -chdir=terraform/environments/dev init -upgrade
    terraform -chdir=terraform/environments/dev apply -auto-approve
    ```

### ArgoCD Configuration Example (`dev` environment):
Application configurations are managed using Kustomize overlays. For instance, the ArgoCD `dev` environment's configuration typically includes an `ingress.yaml`:
```yaml
# apps/argocd/overlays/dev/kustomization.yaml
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ingress.yaml
```

## 3. Development Conventions

### Code Style & Philosophy:
-   **"Functional" Approach**: Prioritize simplicity and "it works" over over-engineered complexity.
- **Progressive Evolution**: Start with "insecure to go fast" (Phase 1-2) with a focus on functionality, then progressively add robust security (e.g., Authentik) and comprehensive monitoring in Phase 3+.
-   **Version Pinning**: Always pin versions for Helm charts, Docker images, Terraform providers.
-   **No Inline YAML**: All values in `values-{env}.yaml` files; no inline YAML.
-   **DRY**: Single base Terraform module, single Helm chart per application.

### Architecture Patterns:
-   **Talos Immutable**: Recreate on upgrade; no `terraform apply -replace`.
-   **100% GitOps**: All changes driven through Git; manual intervention only for initial ArgoCD bootstrap.
-   **3-Layer ArgoCD**: System, Security, and Applications.
-   **Kustomize Overlays**: `base/` + `overlays/{env}/` structure for environment differentiation.
-   **No SSH**: Talos's design intentionally restricts SSH access.
-   **Local Services**: HomeAssistant and mail should remain locally accessible even if the cluster is down.

### Testing & Validation Strategy:
-   **yamllint**: Automated validation of all YAML files via GitHub Actions.
-   **OpenSpec Validator**: All specifications must pass `openspec validate --strict`.
-   **Workflow Validation**: GitHub branch protection enforces linear progression (dev → test → staging → prod).
-   **AI Tests**: AI generates code, manual review before PR.
-   **User Tests**: Manual validation on `dev` environment before PR to `test`.

### Git Workflow:
-   **Branch Strategy**:
    -   `dev`: Active development (force-push allowed).
    -   `test`: Testing branch (PR from `dev` only).
    -   `staging`: Pre-production (PR from `test` only).
    -   `main (prod)`: Production branch (PR from `staging` only).
-   **GitHub Protection Rules**: Enforce status checks (yamllint, openspec-validate), linear history, and required reviews (for `staging` and `main`).

## 4. AI Agent Instructions

### Most Used Tools:
For general file system and code modification tasks, the following tools are frequently used and highly effective:
-   `run_shell_command`: To execute various shell commands.
-   `replace`: For precise, context-aware text replacement within files.
-   `write_file`: To create new files or overwrite existing ones.

### Issue Tracking with bd (beads):
**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.
-   **Workflow for AI Agents**:
    1.  **Check ready work**: `bd ready --json`
    2.  **Claim your task**: `bd update <id> --status in_progress --json`
    3.  **Work on it**: Implement, test, document.
    4.  **Discover new work?**: Create linked issue: `bd create "Found bug" -p 1 --deps discovered-from:<parent-id> --json`
    5.  **Complete**: `bd close <id> --reason "Done" --json`
    6.  **Commit together**: Always commit the `.beads/issues.jsonl` file with code changes to keep issue state synchronized.

### Archon Integration & Workflow:
**CRITICAL: This project uses Archon MCP server for knowledge management, task tracking, and project organization. ALWAYS start with Archon MCP server task management.**
-   **Core Workflow: Task-Driven Development**:
    1.  **Get Task** → `find_tasks(task_id="...")` or `find_tasks(filter_by="status", filter_value="todo")`
    2.  **Start Work** → `manage_task("update", task_id="...", status="doing")`
    3.  **Research** → Use knowledge base (RAG workflow)
    4.  **Implement** → Write code based on research
    5.  **Review** → `manage_task("update", task_id="...", status="review")`
    6.  **Next Task** → `find_tasks(filter_by="status", filter_value="todo")`
-   **NEVER skip task updates. NEVER code without checking current tasks first.**

### OpenSpec Instructions:
Always consult `openspec/AGENTS.md` when the request:
-   Mentions planning or proposals (e.g., proposal, spec, change, plan).
-   Introduces new capabilities, breaking changes, architecture shifts, or significant performance/security work.
-   Sounds ambiguous and requires an authoritative specification before coding.
-   **Three-Stage Workflow**:
    1.  **Creating Changes**: Review existing context, choose a unique `change-id`, scaffold `proposal.md`, `tasks.md`, optional `design.md`, and spec deltas. Validate with `openspec validate <id> --strict`.
    2.  **Implementing Changes**: Read `proposal.md`, `design.md` (if present), and `tasks.md`. Implement tasks sequentially, confirm completion, and update the checklist. **Do not start implementation until the proposal is approved.**
    3.  **Archiving Changes**: After deployment, archive changes using `openspec archive <change-id>`.

### Tool Selection Guide:
| Task                      | Tool                      | Why                       |
|---------------------------|---------------------------|---------------------------|
| Find files by pattern     | `glob`                    | Fast pattern matching     |
| Search code content       | `search_file_content`     | Optimized regex search    |
| Read specific files       | `read_file`               | Direct file access        |
| Explore unknown scope     | `codebase_investigator`   | Multi-step investigation  |

### Critical Note for AI Agents:
-   **Concurrency**: The `dev` and `test` environments cannot run simultaneously due to network connectivity issues during `terraform plan` when the respective environment is down.
-   **Host Format**: The host format is `<host>.<env>.truxonline.com`.
-   **Kubeconfig Path**: The kubeconfig path for the test environment is `/root/vixens/terraform/environments/test/kubeconfig-test`.
