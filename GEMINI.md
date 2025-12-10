# Vixens GitOps Project - Gemini CLI Context

This document provides an overview of the Vixens GitOps project, intended to serve as instructional context for the Gemini CLI.

## Project Overview

The Vixens project implements a GitOps approach for deploying and managing Kubernetes clusters based on Talos Linux. It encompasses the complete lifecycle, from infrastructure provisioning using Terraform to application management with ArgoCD.

**Key Technologies:**
- **Terraform:** For declarative infrastructure provisioning.
- **Talos Linux:** Immutable operating system for Kubernetes nodes.
- **Kubernetes:** Container orchestration platform.
- **ArgoCD:** GitOps continuous delivery tool for Kubernetes applications.
- **Kustomize:** Used for managing application configurations across environments.

**Architecture:**
The project utilizes a two-pronged approach, reflecting a "two control loops" architecture:

1.  **Infrastructure Loop (Terraform):**
    -   A 3-level Terraform architecture is employed for enhanced DRY (Don't Repeat Yourself) principles.
    -   Structure: `environments/{env}/main.tf` -> `modules/environment/` -> `modules/{shared, talos, cilium, argocd}`.
    -   The `terraform/base/` directory is deprecated and unused.
    -   `modules/environment/` acts as a central orchestration layer for core infrastructure components (Talos cluster, Cilium CNI, ArgoCD GitOps).
    -   `modules/shared/` centralizes global, reusable configurations such as:
        - Chart versions: Cilium 1.18.3, ArgoCD 7.7.7, Traefik 25.0.0, cert-manager v1.14.4
        - Control plane tolerations: Reusable across Cilium, ArgoCD, Hubble
        - Cilium capabilities: Validated set of 11 Linux capabilities for Talos
        - Network defaults: Pod subnet, service subnet
        - Security defaults: Common security contexts
        - Timeouts: Helm install (20min), upgrade (15min), API wait (5min)

2.  **Application Loop (ArgoCD):**
    -   ArgoCD operates on an "App-of-Apps" pattern.
    -   A root application (templated by Terraform for each environment) watches `argocd/overlays/${environment}/` to deploy all applications defined there.
    -   Application configurations are managed using Kustomize overlays within `apps/<app-name>/overlays/<env-name>` directories.

## Building and Running

### Prerequisites

To interact with the project, ensure the following tools are installed:
-   [Terraform](https://www.terraform.io/downloads.html)
-   [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)
-   [talosctl](https://www.talos.dev/v1.6/introduction/getting-started/#talosctl)

### Terraform (Infrastructure Management)

To provision or update the infrastructure for a specific environment (e.g., `dev`):

1.  Navigate to the environment's directory:
    ```bash
    cd terraform/environments/dev
    ```
2.  Initialize Terraform (run from the environment directory):
    ```bash
    terraform init -upgrade
    ```
3.  Apply the Terraform configuration (run from the environment directory):
    ```bash
    terraform apply -auto-approve
    ```
    *(Note: `terraform plan` can be used to review changes before applying.)*

### ArgoCD (Application Deployment)

ArgoCD is bootstrapped by Terraform. Applications are defined in Git and automatically synchronized by ArgoCD.

-   Root application definition: `argocd/base/root-app.yaml.tpl`
-   Application overlays for environments: `apps/<app-name>/overlays/<env-name>/kustomization.yaml`

### Testing

The project uses a custom Python-based test runner: `scripts/run_tests.py`.

**Usage:**
```bash
python3 scripts/run_tests.py <test_type> <environment> [--tags <tags>] [--output <format>] [--no-color]
```

-   `<test_type>`: `fonc` (functional tests) or `tech` (technical tests).
-   `<environment>`: `dev`, `test`, `staging`, or `prod`.
-   `--tags`: (Optional) Comma-separated list of tags to filter tests (e.g., `terraform,network`).
-   `--output`: (Optional) `console` (default) or `json`.
-   `--no-color`: (Optional) Disable colored output.

**Example:**
```bash
python3 scripts/run_tests.py fonc dev --tags terraform
```

**Pre-flight Checks for Testing:**
The test runner requires `terraform` and `talosctl` to be available in the system's PATH.

## Development Conventions

-   **GitOps Workflow:** All infrastructure and application configurations are stored in Git, acting as the single source of truth. Changes are applied via Pull Requests.
-   **Branch-Based Deployment:** Each environment (dev, test, prod, etc.) corresponds to a specific Git branch.
-   **Terraform 3-Level Architecture:** Strict adherence to the `environments/ → modules/environment/ → modules/{shared, ...}` pattern for managing infrastructure.
-   **Typed Variables:** Terraform variables are strictly typed and grouped into objects for clarity and validation.
-   **Kustomize Overlays:** Application configurations are customized per environment using Kustomize.
-   **Architectural Decision Records (ADRs):** Significant architectural decisions are documented in ADRs (e.g., `docs/adr/006-terraform-3-level-architecture-REVISED.md`).
-   **Secrets Management:** Secrets are managed separately per environment within the `.secrets/<environment>` directory.
-   **Conformity Scoring:** The project uses a conformity scoring grid (mentioned in ADRs).

## Work Process:

**1. Initialization:**
    - Retrieve all tasks assigned to "Coding Agent" with status "todo", "doing", or "review", ensuring to use a sufficiently large `per_page` value to get all tasks.

**2. Task Selection:**
    - **Priority 1:** Select tasks assigned to "Coding Agent" with status "review".
    - **Priority 2:** If no "review" tasks, select tasks assigned to "Coding Agent" with status "doing".
    - **Priority 3:** If no "review" or "doing" tasks, propose "todo" tasks that appear most critical/important.

**3. Working on a Task:**
    - Change task status to "doing" in Archon.
    - Prioritize Archon RAG for documentation and Serena for code access.
    - Proceed incrementally.
    - Keep `@docs/RECETTE-FONCTIONNELLE.md` and `@docs/RECETTE-TECHNIQUE.md` up to date.

**4. Task Completion:**
    - Change task status to "review" in Archon.
    - Validate results thoroughly using all available means, including Playwright/curl for application access, not just its state.
    - If successful, change assignee to "User", keep status as "review", and proceed to the next task.
    - If unsuccessful, revert status to "doing" and resume work.

**Important Notes:**
- Remember control-plane tolerations.
- If there is a PVC with `ReadWriteOnce` access mode, ensure the deployment strategy is `Recreate`.
- Implement HTTP to HTTPS redirection.
- Ensure correct `cert-manager` cluster issuer (`letsencrypt-staging` for dev/test/staging, `letsencrypt-prod` for prod).
- **IMPORTANT: DO NOT TOUCH COMMON RESOURCES WITHOUT ASKING THE USER!!**