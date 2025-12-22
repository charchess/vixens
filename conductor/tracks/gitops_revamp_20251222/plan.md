# Plan: GitOps Structure Revamp & Infisical Configuration

## Phase 1: Infrastructure & Controller Setup
- [x] Task: Deploy `stakater/Reloader` to the `tools` (or system) namespace.
- [x] Task: Create a new custom controller (e.g., `gitops-revision-controller`) using Python/Kopf.
- [x] Task: Implement controller logic to watch Secrets with label `argocd.argoproj.io/target-revision-secret=true`.
- [x] Task: Implement controller logic to patch ArgoCD Application `spec.source.targetRevision`.
- [x] Task: Deploy the controller to the cluster (Deployment + RBAC).

## Phase 2: Home Assistant Pilot (Config Migration)
- [x] Task: Create Infisical Secret `homeassistant-config` containing the full `configuration.yaml`.
- [x] Task: Update Home Assistant `InfisicalSecret` manifest to sync `homeassistant-config` to a K8s Secret.
- [x] Task: Refactor Home Assistant `kustomization.yaml` to remove local `configmap.yaml`.
- [x] Task: Update Home Assistant `deployment.yaml` to mount the new Secret as a volume at `/config/configuration.yaml`.
- [x] Task: Add `reloader.stakater.com/auto: "true"` annotation to Home Assistant Deployment.
- [ ] Task: Verify Home Assistant starts correctly with config from Infisical.

## Phase 3: Dynamic Revision Verification
- [x] Task: Create an Infisical Secret for Home Assistant revision (e.g., `HASS_TARGET_REVISION`).
- [x] Task: Configure the `gitops-revision-controller` to watch this secret.
- [ ] Task: Functional Test: Change revision in Infisical and observe ArgoCD Application update.

## Phase 4: Follow-up & Documentation
- [x] Task: Document the new Configuration Management pattern in `conductor/product-guidelines.md` or `tech-stack.md`.
- [x] Task: Create a new Track for "Global Application Migration & Infisical Hierarchy Refactor".
