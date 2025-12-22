# Specification: Global Application Migration & Infisical Hierarchy Refactor

## 1. Overview
This track focuses on standardizing the Infisical secret hierarchy to strictly mirror the Git repository's application structure. This ensures predictability, simplifies secret management, and aligns security boundaries with the codebase structure. The migration will be performed incrementally using a parallel run strategy to minimize downtime.

## 2. Functional Requirements

### 2.1 Hierarchy Standardization
*   **Structure:** Infisical paths MUST mirror the `apps/` directory structure in Git.
    *   Example: `apps/70-tools/linkwarden` -> Infisical Path: `/apps/70-tools/linkwarden`
*   **Shared Secrets:** Secrets used by multiple applications (e.g., Database credentials) MUST be duplicated into the specific application's path.
    *   No cross-referencing global paths for application logic.

### 2.2 Migration Process
*   **Audit:** Identify all current `InfisicalSecret` resources and their source paths.
*   **Map:** Generate a mapping table of `Old Path` -> `New Path`.
*   **Parallel Creation:** User manually creates secrets at the `New Path`.
*   **Update:** Update Kubernetes manifests (`InfisicalSecret`, `Deployment`, `Kustomization`) to point to the `New Path`.
*   **Cleanup:** Remove old secret paths after successful verification.

## 3. Technical Implementation Strategy

### 3.1 Mapping & Audit Phase
*   Script or manual audit to list all `kind: InfisicalSecret` in the codebase.
*   Extract `spec.authentication.universalAuth.secretsScope.secretsPath`.
*   Determine the correct target path based on the file location.

### 3.2 Execution Strategy (Per Application Group)
*   Batch applications by their category folder (e.g., `00-infra`, `70-tools`).
*   For each batch:
    1.  **Request User Action:** Provide a clear list of secrets to be created at the new paths.
    2.  **Wait for Confirmation:** Pause track execution until user confirms creation.
    3.  **Refactor Code:** Update `infisical-secret.yaml` files.
    4.  **Verify:** Check ArgoCD sync status (if environment is available) or dry-run.

## 4. Acceptance Criteria
*   [ ] All `InfisicalSecret` manifests point to a path starting with `/apps/XX-category/app-name`.
*   [ ] No application relies on a shared/global secret path (except system-level components if strictly necessary).
*   [ ] Application configuration files (ConfigMaps) are successfully generated from the new secret paths.

## 5. Out of Scope
*   Automated creation of Infisical secrets (User action required).
*   Changing the internal content/keys of the secrets (unless required for format standardization).
