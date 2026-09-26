# Dependency management

Renovate is the single dependency-update manager for Vixens.

## Ownership

- `renovate.json` is the repository policy source of truth.
- The self-hosted Renovate ConfigMap only defines runtime/platform settings and must not override repository update or automerge policy.
- Renovate manages GitHub Actions in addition to the repository's Kubernetes, Helm values, Terraform, Dockerfile and custom-regex dependencies.
- Dependabot version-update configuration is intentionally absent to avoid duplicate pull requests and split ownership.

## Merge policy

- `minor`, `patch`, `pin` and `digest` updates may use platform auto-merge after required repository checks pass.
- `major` updates require human review and must not be auto-merged by Renovate.
- Package-specific exceptions remain explicit `packageRules` in `renovate.json`.

GitHub branch protection and required checks remain authoritative gates: Renovate auto-merge does not bypass them.

## Verification

When changing dependency policy, verify all three layers together:

1. `renovate.json` contains the intended managers and package rules.
2. `apps/70-tools/renovate/base/configmap.yaml` contains no competing `packageRules` or global `automerge` policy.
3. `.github/dependabot.yml` is absent unless responsibility is deliberately split again and documented here.
