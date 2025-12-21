# Specification: Execute Pending GitOps & Application Tasks

## Goal
To stabilize the application layer and ensure the ArgoCD GitOps workflow is functioning correctly by resolving pending tasks assigned in Archon. The primary focus is on resolving storage/NFS issues and ensuring application health in the `dev` environment.

**CRITICAL ADDITION:** Explicit validation of HTTPS access and redirection for all applications in `dev`.

## Scope
- **Task Source:** Archon MCP (assigned to "Coding Agent").
- **Primary Focus:**
    -   **Storage:** Resolution of known NFS/Storage issues (e.g., failed volume mounts for applications like Sabnzbd).
    -   **Access Security:** Verification of HTTP -> HTTPS redirection for ALL applications using Traefik `redirectScheme` middleware.
    -   **Sync:** Verification of ArgoCD application sync status.
    -   **Diagnostics:** Identification of root causes for any applications in degraded states (CrashLoopBackOff, Error, etc.).
-   **Environment:** Dev (primary), with propagation to Test/Staging/Prod as per workflow.

## Requirements
-   Strict adherence to the `conductor/workflow.md` protocols.
-   Use of `serena` for all code modifications.
-   **Validation:**
    -   Use `playwright` or `curl -I` to verify:
        1.  HTTP requests receive a 301/308 redirect to HTTPS.
        2.  HTTPS requests return 200 OK (or expected app status code).
        3.  Valid TLS certificate issuance.
-   **Configuration:** Ensure `ingress` resources utilize the common `redirect-https` middleware.