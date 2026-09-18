# Application runbooks

This directory contains per-application operational notes.

The corresponding manifests under `apps/<category>/<application>/` and ArgoCD definitions under
`argocd/` are the deployment source of truth. A runbook may contain historical troubleshooting
context; do not infer desired Kubernetes state from prose when the manifests say otherwise.

## Expected content

Useful application documentation may cover:

- purpose and dependencies
- persistent data and backup expectations
- secret keys required by the workload, without secret values
- ingress/service behaviour
- safe runtime checks
- known application-specific failure modes
- restore or migration notes that are still applicable

Do not maintain deployment-status checkboxes or copied resource inventories here. Those drift from the cluster and Git.

To add a workload, start with [Adding an application](../guides/adding-new-application.md).
