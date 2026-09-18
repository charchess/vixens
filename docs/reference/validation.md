# Validation reference

## Local

```bash
just validate
```

This runs YAML style checks, embedded Helm values validation, Kustomize reference validation and repository-contract checks.

## GitHub Actions

`Validate & Security` is the authoritative merge gate. Its final `Validation Summary` fails when a required upstream validation fails.

The temporary `refactor/gitops-cleanroom` branch is included in the push trigger so the clean-room work can be validated before opening a PR.

## Render validation

PR and merge-queue workflows build Kustomize overlays with Helm enabled. Changes under an application base validate that application's environment overlays; changes to shared components trigger broad validation.

## Secret scanning

Gitleaks scans current repository content including documentation. Documentation and agent-oriented text are not exempt from secret detection.
