# Project Workflow

## 1. Archon-First Task Management (CRITICAL)
- **Primary Source:** Archon MCP server is the single source of truth for tasks.
- **Workflow Cycle:**
    1.  **Initialization:** Retrieve tasks assigned to "Coding Agent" (status: `todo`, `doing`, `review`).
    2.  **Selection:**
        -   Priority 1: Resume `review` tasks.
        -   Priority 2: Continue `doing` tasks.
        -   Priority 3: Pick critical `todo` tasks.
    3.  **Status Update:** IMMEDIATE update in Archon to `doing` before starting work.
    4.  **Completion:** Update status to `review` and re-assign to "User" ONLY after successful validation.

## 2. Research & Implementation
- **Documentation:** Use Archon RAG (`rag_search_knowledge_base`) as the primary research tool.
- **Code Access:** Use **Serena** tools (`serena_read_file`, etc.) for reading and manipulating the codebase. Run `onboarding` if Serena instructions are needed.
- **Incremental Work:** Proceed in small, verifiable steps.

## 3. Deployment & Validation Protocol
- **GitOps Enforcement:**
    -   All commits must be pushed to the **`dev` branch ONLY**.
    -   Infrastructure changes via Terraform; Application changes via ArgoCD overlays.
-   **Environment Propagation:**
    -   Implement and validate in **Dev** first.
    -   Once validated, port changes to `test`, `staging`, and `prod` overlays.
-   **Validation (Definition of Done):**
    -   **Technical Check:** `terraform plan/apply` success, ArgoCD sync healthy.
    -   **Functional Check:** Verify application access using **Playwright** (preferred for web) or `curl`.
    -   **Certificates:** Verify TLS issuance (Let's Encrypt Staging for non-prod).

## 4. Phase Completion Verification
- **Protocol:** At the end of each phase (group of tasks), perform a comprehensive health check of the `dev` environment to ensure no regression before proceeding.

## 5. Specific Constraints
-   **Ingress:** `<app>.<env>.truxonline.com` (except Prod: `<app>.truxonline.com`).
-   **Storage:** If PVC is RWO, Deployment strategy must be `Recreate`.
-   **Networking:** Ensure HTTP -> HTTPS redirection.
