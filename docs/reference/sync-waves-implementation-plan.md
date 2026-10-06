# ArgoCD sync-waves implementation plan — historical

> **Status: historical compatibility stub.**
>
> The former contents of this file were a 2024 point-in-time implementation plan.
> They referenced retired branches/tools/services, direct pushes to `main`,
> destructive dev-cluster recreation steps and a pre-current ArgoCD application
> inventory. They are not safe current instructions.
>
> The original plan remains available in Git history.

## Current sources

For current sync-wave behavior use:

- [argocd-sync-waves.md](argocd-sync-waves.md) — current reference;
- current `argocd/` Applications and `apps/_shared/components/sync-wave/`;
- root [WORKFLOW.md](../../WORKFLOW.md) / [AGENTS.md](../../AGENTS.md) for the
  branch/PR/CI/reconciliation lifecycle.

Any new sync-wave change must start from the current manifests and dependency
graph. Do not recreate applications, wave numbers, Infisical dependencies or
`dev -> main` procedures from the historical plan.

Persistent changes use a short-lived branch and Pull Request. Do not push directly
to `main` or destroy/recreate a cluster merely to validate a sync-wave edit unless
that destructive test is separately justified and explicitly authorized.
