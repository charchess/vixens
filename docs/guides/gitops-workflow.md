# GitOps workflow

[WORKFLOW.md](../../WORKFLOW.md) is authoritative. This guide explains the normal operator flow.

## Start work

Create or select a GitHub Issue, then create a short-lived branch.

```bash
just gh-start <issue-number>
```

Make the desired-state change under `apps/` or `argocd/`, together with the minimum documentation needed to explain it.

## Validate locally

```bash
just validate
```

Local validation is a convenience. GitHub Actions repeats validation independently.

## Merge to development

Push the branch and complete its PR to `main`.

After merge, ArgoCD development Applications reconcile `main`. Diagnose the resulting runtime with read-only commands and fix any problem through another Git change.

## Production

After the exact merged state has been validated in development, promote its automatically generated `dev-v...` tag using the production workflow.

See [Production promotion](promotion-workflow.md).

## What not to do

Do not use direct workload `kubectl apply/edit/patch` as permanent configuration.
Do not manually move `prod-stable` during the normal release flow.
