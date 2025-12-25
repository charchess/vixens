# Prod Readiness and Portability Plan

## Goal
Verify the portability of the infrastructure and applications from `dev` to `prod` to ensure a smooth promotion process.

## Phase 1: Overlay Audit
- [ ] **Structure Comparison**: Systematically compare `argocd/overlays/dev` and `argocd/overlays/prod` to identify structural discrepancies (missing apps, extra files).
- [ ] **Content Diff**: check differences in `kustomization.yaml` and patch files between environments. Ensure `prod` specific values (replicas, resources, ingress hosts) are correctly set.

## Phase 2: Prerequisites Validation
- [ ] **Secret Availability**: Verify that all `InfisicalSecret` resources defined in `prod` overlays have the corresponding secrets available in the Infisical project (Prod environment).
- [ ] **Network & Storage**: Check that `prod` specific resources (like specific storage classes or ingress classes if different) are correctly referenced.

## Phase 3: Dry Run
- [ ] **Template Validation**: Run `kubectl kustomize argocd/overlays/prod` to ensure the overlays build without error.
