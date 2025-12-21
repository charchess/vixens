# Product Guidelines: Vixens GitOps Platform

## Operational Tone & Voice
- **Professional & Technical:** Documentation and agent communications must be concise, objective, and rigorously focused on implementation details, protocols, and technical verification.

## Core Design Principles
1.  **Immutability:** Follow the Talos Linux philosophy where infrastructure nodes are replaced rather than patched.
    -   *Crucial Exception:* Data persistence must be guaranteed. Persistent Volume Claims (PVCs) for both iSCSI and NFS must survive node replacements and upgrades to prevent any data loss.
2.  **Observability:** Comprehensive logging, monitoring, and tracing are integrated from the start to ensure visibility into the system's state and health.
3.  **Security:** Implemented progressively to ensure functionality before restriction.

## Development & Promotion Strategy
-   **Iterative Implementation Cycle:**
    1.  **Functional Implementation (Dev):** Focus first on achieving core functionality and verifying the "happy path" in the development environment.
    2.  **Security Hardening (Dev):** Once functional, layer on security constraints (Network Policies, strict RBAC, secrets management) within the development environment.
    3.  **Promotion:** Only after a feature is both functional and secured in `dev` is it promoted to `test`, `staging`, and finally `prod`.

## Repository & Architectural Standards
-   **Base/Overlay Pattern:**
    -   **Base:** Define the generic, "canonical" configuration for an application or module in a `base/` directory. This should be as environment-agnostic as possible.
    -   **Overlays:** Create specific configurations for each environment (`overlays/dev`, `overlays/prod`, etc.) that inherit from `base`. Use Kustomize patches or Terraform variable overrides to apply environment-specific differences (e.g., replica counts, ingress domains, resource limits).
    -   *Constraint:* Never duplicate code in overlays that belongs in the base.
-   **Directory Structure:** Adhere strictly to the existing hierarchy (e.g., `apps/<category>/<app-name>/overlays/<env>/`).
-   **Strict GitOps:** All state changes must originate from a Git commit to the `dev` branch.
