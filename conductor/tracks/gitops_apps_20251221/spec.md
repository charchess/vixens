# Specification: Execute Pending GitOps & Application Tasks

## Goal
To stabilize the application layer and ensure the ArgoCD GitOps workflow is functioning correctly by resolving pending tasks assigned in Archon. The primary focus is on resolving storage/NFS issues and ensuring application health in the `dev` environment before propagation.

## Scope
- **Task Source:** Archon MCP (assigned to "Coding Agent").
- **Primary Focus:**
    -   Resolution of known NFS/Storage issues (e.g., failed volume mounts for applications like Sabnzbd).
    -   Verification of ArgoCD application sync status.
    -   Validation of application access via Ingress.
-   **Environment:** Dev (primary), with propagation to Test/Staging/Prod as per workflow.

## Requirements
-   Strict adherence to the `conductor/workflow.md` protocols.
-   Use of `serena` for all code modifications.
-   Validation using `playwright` or `curl`.
-   Deployment strategy modification (Recreate) for RWO volumes if applicable.
