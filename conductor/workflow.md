# Project Workflow

## 1. Archon & Conductor Synchronization (Agent-Managed Loop)
- **Loop A: Archon -> Conductor (Promotion)**
    - Identify high-priority tasks (`task_order` > 80) or complex requirements in Archon.
    - Promote these to a **Track** in Conductor for detailed architectural planning.
    - Initialize `spec.md` and `plan.md` in the new Track directory.
- **Loop B: Conductor -> Archon (Expansion)**
    - After generating or updating a `plan.md`, manually create the corresponding granular tasks in Archon using `manage_task`.
    - **Priority Mapping Calculation:**
        - Phase 1 Tasks: `task_order` = 90
        - Phase 2 Tasks: `task_order` = 70
        - Phase 3 Tasks: `task_order` = 50 (and below)
    - Ensure each Archon task description references the Track ID.

## 2. Archon-First Task Execution (CRITICAL)
- **Primary Source:** Archon MCP server is the single source of truth for execution.
- **Workflow Cycle:**
    1.  **Initialization:** Retrieve tasks assigned to "Coding Agent" (status: `todo`, `doing`, `review`).
    2.  **Selection:**
        -   Priority 1: Resume `review` tasks.
        -   Priority 2: Continue `doing` tasks.
        -   Priority 3: Pick critical `todo` tasks (highest `task_order`).
    3.  **Status Update:** IMMEDIATE update in Archon to `doing` before starting work.
    4.  **Completion:** Update status to `review` and re-assign to "User" ONLY after successful validation.

## 3. Research & Implementation
- **Documentation:** Use Archon RAG (`rag_search_knowledge_base`) as the primary research tool.
- **Code Access:** Use **Serena** tools for reading and manipulating the codebase.
- **Incremental Work:** Proceed in small, verifiable steps.

## 4. Deployment & Validation Protocol
- **GitOps Enforcement:** Commits to **`dev` branch ONLY**.
- **Environment Propagation:** Validate in **Dev** first, then port to `test`, `staging`, and `prod`.
- **Validation (Definition of Done):**
    - Technical: `terraform plan/apply`, ArgoCD sync healthy.
    - Functional: Verify via **Playwright** (web) or `curl`.
    - Certificates: Verify TLS issuance (Let's Encrypt Staging for non-prod).

## 5. Specific Constraints
-   **Ingress:** `<app>.<env>.truxonline.com` (except Prod: `<app>.truxonline.com`).
-   **Storage:** If PVC is RWO, Deployment strategy must be `Recreate`.
-   **Networking:** Ensure HTTP -> HTTPS redirection.
