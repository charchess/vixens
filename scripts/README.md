# Validation scripts

Only static GitOps validation helpers live here.

- `validate-helm-values.py`: validates YAML embedded in ArgoCD Helm values.
- `validate-pdb-selectors.py`: checks PDB selectors against rendered workloads.
- `validate-kustomization-refs.sh`: checks local Kustomize resource/patch references.
- `validate-repository.py`: checks repository-level workflow, documentation and migration invariants.

Run the supported local validation entry point with:

```bash
just validate
```

Runtime cluster diagnosis is intentionally not encoded as a second desired-state
system in this directory.
