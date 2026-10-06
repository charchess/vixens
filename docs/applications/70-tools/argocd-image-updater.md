# ArgoCD Image Updater — retired documentation

> **Status: historical compatibility stub.**
>
> There is currently no `argocd-image-updater` application desired state under
> `apps/70-tools/`. This page described an older deployment and must not be used
> as an operational runbook or as evidence that registry credentials still require
> a dedicated secret integration.
>
> Historical content remains available in Git history.

Vixens currently relies on Renovate for dependency/image update discovery and
automation. See:

- `renovate.json`
- `docs/applications/70-tools/renovate.md`
- `WORKFLOW.md`

If ArgoCD Image Updater is reintroduced, document its current manifests, secret
source and write-back semantics from the executable Git state rather than reviving
this retired page.
