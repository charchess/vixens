# Plan: Global Application Migration & Infisical Hierarchy Refactor

## Phase 1: Audit & Mapping
- [ ] Task: Audit all `InfisicalSecret` resources in the codebase.
- [ ] Task: Generate a comprehensive mapping table (`Old Path` -> `New Path`) following the strict mirror rule.
- [ ] Task: Identify all shared secrets (DB, S3, etc.) and map their duplication paths.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Audit & Mapping' (Protocol in workflow.md)

## Phase 2: Infrastructure & Storage Migration (00-infra, 01-storage)
- [ ] Task: User Action: Create Infisical secrets for `00-infra` and `01-storage` at new mirrored paths.
- [ ] Task: Refactor `InfisicalSecret` manifests for `argocd`, `cert-manager`, `synology-csi`, etc.
- [ ] Task: Verify secret synchronization for the infrastructure layer.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Infrastructure & Storage' (Protocol in workflow.md)

## Phase 3: Database & Media Migration (04-databases, 20-media)
- [ ] Task: User Action: Create Infisical secrets for `04-databases` and `20-media` at new mirrored paths.
- [ ] Task: Refactor `InfisicalSecret` manifests for `postgresql-shared`, `sabnzbd`, `jellyfin`, etc.
- [ ] Task: Verify secret synchronization for the database and media layers.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Database & Media' (Protocol in workflow.md)

## Phase 4: Services, Tools & Home Migration (10-home, 60-services, 70-tools)
- [ ] Task: User Action: Create Infisical secrets for remaining namespaces.
- [ ] Task: Refactor `InfisicalSecret` manifests for `homeassistant`, `vaultwarden`, `netbox`, etc.
- [ ] Task: Verify secret synchronization for the remaining applications.
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Services & Tools' (Protocol in workflow.md)

## Phase 5: Verification & Cleanup
- [ ] Task: Functional validation: Ensure all applications are healthy and using new secrets.
- [ ] Task: User Action: Delete old/deprecated secret paths in Infisical.
- [ ] Task: Update project documentation to reflect the new secret management standards.
- [ ] Task: Conductor - User Manual Verification 'Phase 5: Verification & Cleanup' (Protocol in workflow.md)
