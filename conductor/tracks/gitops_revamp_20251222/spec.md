# Specification: GitOps Structure Revamp & Infisical-Driven Configuration

## 1. Overview
This track aims to modernize the GitOps workflow by decoupling environment definitions from specific Git branches and moving application configurations into Infisical. This allows for dynamic environment targeting (e.g., testing a feature branch on a dev environment) and faster configuration iterations without triggering full CI/CD pipelines.

## 2. Functional Requirements

### 2.1 Dynamic Git Revision Strategy
*   **Default Behavior:**
    *   `prod` environment targets `main` branch.
    *   `dev`, `test`, `staging` environments target `dev` branch.
*   **Override Mechanism:**
    *   The target revision (Git branch/tag) for an environment/application must be definable in **Infisical**.
    *   A custom controller or operator must watch these Infisical values (synced to K8s Secrets) and automatically patch the ArgoCD Application's `targetRevision`.
    *   **Goal:** Changing a value in Infisical (e.g., `TARGET_REVISION=feature/login`) instantly redirects the ArgoCD app to that branch.

### 2.2 Infisical-Based Application Configuration
*   **Storage:** Full configuration files (e.g., `configuration.yaml` for Home Assistant) will be stored as secrets/values within Infisical.
*   **Injection:**
    *   Use the **Infisical Operator** to sync these values into Kubernetes Secrets.
    *   Update Application Deployments to mount these Secrets as files (replacing previous ConfigMap mounts).
*   **Updates:**
    *   Implement `stakater/Reloader` (or similar) to automatically restart application pods when the synced Secret changes, ensuring immediate configuration application.

## 3. Technical Implementation Strategy

### 3.1 Git Revision Controller
*   Develop a lightweight Kubernetes Operator (using Python/Kopf or Go).
*   **Watches:** Secrets managed by Infisical (specifically looking for a defined key like `ARGOCD_TARGET_REVISION`).
*   **Actions:** Patches the corresponding ArgoCD `Application` resource.

### 3.2 Configuration Migration
*   Refactor `homeassistant` (as the pilot) to remove `ConfigMap` generation from Kustomize.
*   Add `InfisicalSecret` definition to fetch the configuration file content.
*   Update `Deployment` to mount the Secret volume instead of ConfigMap.
*   Deploy `Reloader` to the cluster.

## 4. Acceptance Criteria
*   [ ] Changing the target branch in Infisical triggers an ArgoCD sync to that branch within 2 minutes.
*   [ ] Home Assistant configuration can be edited in Infisical and is reflected in the running pod without any Git commits.
*   [ ] Default environment branching (`prod`->`main`, `dev`->`dev`) works without overrides.
*   [ ] `Reloader` automatically restarts pods upon config change.

## 5. Out of Scope
*   Full migration of *all* applications (focus on Home Assistant as pilot and framework setup).
