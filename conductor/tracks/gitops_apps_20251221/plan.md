# Plan: Execute Pending GitOps & Application Tasks

## Phase 1: Task Initialization & Analysis
- [ ] Task: Connect to Archon and retrieve tasks assigned to "Coding Agent" (Status: review, doing, todo).
- [ ] Task: Select the highest priority task according to the workflow (Review > Doing > Critical Todo).
- [ ] Task: Update selected task status to 'doing' in Archon.
- [ ] Task: Analyze the specific task requirements (e.g., NFS issue details, PVC configurations).
- [ ] Task: Conductor - User Manual Verification 'Task Initialization & Analysis' (Protocol in workflow.md)

## Phase 2: Implementation & Resolution
- [ ] Task: Research solution using Archon RAG (e.g., Synology CSI docs, Talos NFS requirements).
- [ ] Task: Implement fixes using Serena (e.g., updating PVC specs, debugging network policies, checking secret configurations).
- [ ] Task: Apply changes to the `dev` environment via Git commit (to `dev` branch).
- [ ] Task: Verify deployment success (Terraform/ArgoCD sync).
- [ ] Task: Conductor - User Manual Verification 'Implementation & Resolution' (Protocol in workflow.md)

## Phase 3: Validation & Propagation
- [ ] Task: Validate application health using Playwright/Curl (Functional Check).
- [ ] Task: Verify certificate issuance and HTTP->HTTPS redirection.
- [ ] Task: Update task status to 'review' in Archon.
- [ ] Task: Upon success, propagate changes to `test`, `staging`, and `prod` overlays (if applicable/safe).
- [ ] Task: Conductor - User Manual Verification 'Validation & Propagation' (Protocol in workflow.md)
