# Plan: Execute Pending GitOps & Application Tasks

## Phase 1: Task Initialization & Audit
- [ ] Task: Connect to Archon and retrieve tasks assigned to "Coding Agent".
- [ ] Task: Perform an audit of existing Ingresses in `dev` environment to identify those missing HTTPS redirection.
- [ ] Task: Identify applications currently in degraded states (CrashLoopBackOff, ImagePullBackOff, etc.).
- [ ] Task: Update selected task status to 'doing' in Archon.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Task Initialization & Audit' (Protocol in workflow.md)

## Phase 2: Implementation & Resolution
- [ ] Task: Update Ingress resources to include the `redirect-https` middleware.
- [ ] Task: Resolve storage/NFS issues (e.g., updating PVC specs to `Recreate` strategy where needed).
- [ ] Task: Fix configuration errors causing application crashes (Netbox, ArgoCD Server, etc.).
- [ ] Task: Apply changes to the `dev` environment via Git commit (to `dev` branch).
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Implementation & Resolution' (Protocol in workflow.md)

## Phase 3: Validation & Propagation
- [ ] Task: Use Playwright/Curl to verify HTTP -> HTTPS redirection for all apps.
- [ ] Task: Verify successful TLS certificate issuance via Cert-manager.
- [ ] Task: Validate overall application health in `dev`.
- [ ] Task: Update task status to 'review' in Archon and re-assign to User.
- [ ] Task: Propagate validated changes to `test`, `staging`, and `prod` overlays.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Validation & Propagation' (Protocol in workflow.md)