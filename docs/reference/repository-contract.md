# Repository contract

Vixens is a GitOps desired-state repository.

## Allowed first-class content

- Kubernetes/Kustomize desired state under `apps/`
- ArgoCD desired state under `argocd/`
- GitHub workflows required to validate/update/promote that state
- static validation helpers
- concise current documentation
- immutable ADR and incident history
- build context required by a workload image maintained by this repository

## Content that belongs elsewhere

- Terraform/Talos provisioning → `terravixens`
- task databases → GitHub Issues
- third-party agent frameworks and generic skill libraries
- copied secret values or local cluster credentials
- hand-maintained live status/conformity reports
- generated caches and local tool state

The purpose of this boundary is to make a repository clone understandable from Git itself and to prevent obsolete tooling from becoming accidental architecture.
