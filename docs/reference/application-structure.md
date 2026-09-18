# Application structure

The normal native-manifest layout is:

```text
apps/<category>/<app>/
├── base/
│   ├── kustomization.yaml
│   └── ...
└── overlays/
    ├── dev/
    │   └── kustomization.yaml
    └── prod/
        └── kustomization.yaml
```

Not every workload needs every resource. Kustomize bases contain environment-independent desired state; overlays contain only environment differences.

ArgoCD Application definitions live separately under `argocd/overlays/<env>/apps/`.

Shared concerns belong in `apps/_shared/` where an existing component fits. Avoid copying common labels, policy, sizing or scheduling patterns into every application.
