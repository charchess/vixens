# Adding an application

This guide covers GitOps state only.

## 1. Create the application tree

Use the existing category that best matches the workload:

```text
apps/<category>/<app>/
├── base/
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    └── prod/
        └── kustomization.yaml
```

Prefer existing shared components under `apps/_shared/` rather than copying common policy/resources.

## 2. Define Kubernetes state

Add only resources required by the workload: Deployment/StatefulSet, Service, Ingress, PVC, policies and related configuration.

If secrets are required, use an `ExternalSecret` referencing the `openbao` ClusterSecretStore. See [Secret management](secret-management.md).

## 3. Add ArgoCD Applications

Development repository sources use:

```yaml
repoURL: https://github.com/charchess/vixens
targetRevision: main
```

Production repository sources use:

```yaml
repoURL: https://github.com/charchess/vixens
targetRevision: prod-stable
```

External Helm chart sources keep their own chart version; only the Vixens Git source follows the environment revision.

Register the Application in the appropriate `argocd/overlays/<env>/kustomization.yaml`.

## 4. Validate

```bash
kustomize build --enable-helm apps/<category>/<app>/overlays/dev >/dev/null
kustomize build --enable-helm apps/<category>/<app>/overlays/prod >/dev/null
just validate
```

## 5. Merge and observe development

Create a PR to `main`. Once merged, wait for ArgoCD development reconciliation and perform read-only runtime validation.

## 6. Promote

Promote only the validated immutable dev snapshot. See [Production promotion](promotion-workflow.md).
