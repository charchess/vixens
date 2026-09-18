# Vixens GitOps workflow

This file is the canonical workflow for this repository.

## Repository contract

Vixens is a **GitOps state repository**. Desired Kubernetes state lives in Git and is reconciled by ArgoCD.

- `main` is the only long-lived branch.
- Development ArgoCD Applications track `main`.
- Production ArgoCD Applications track the moving `prod-stable` tag.
- Feature/fix branches are short-lived and merge to `main` through pull requests.
- Terraform/Talos provisioning belongs in the separate `terravixens` repository.
- GitHub Issues are the only task tracker used by this repository.

## Sources of truth

1. Desired runtime state: `apps/` and `argocd/`
2. Workflow: this file
3. Current architecture: `docs/architecture.md`
4. Architecture decisions: `docs/adr/000-index.md`
5. Operational guides: `docs/guides/`
6. Application-specific notes: `docs/applications/`
7. Historical incidents: `docs/post-mortems/`

Agent-specific files must not redefine these rules.

## Change flow

```text
GitHub Issue
    ↓
feature/fix branch
    ↓
Pull Request to main
    ↓
required CI checks
    ↓
squash merge
    ↓
main
    ↓
ArgoCD dev reconciliation
    ↓
runtime validation
    ↓
manual production promotion
    ↓
prod-stable
    ↓
ArgoCD prod reconciliation
```

Typical commands:

```bash
just gh-resume
just gh-start <issue-number>

# work, commit and push
just validate

just gh-done <pr-number>
```

Direct pushes to `main` are not part of the normal workflow.

## Production promotion

Production promotion is performed **only** through `.github/workflows/promote-prod.yaml`.

```bash
just promote-prod v2026.09.3271
# equivalent:
gh workflow run promote-prod.yaml -f version=v2026.09.3271
```

The workflow consumes the corresponding immutable `dev-v...` snapshot and updates `prod-stable`.

Do not move `prod-stable` manually as a normal deployment mechanism.

`prod-working` is a manual rescue reference. **Never move it without explicit owner approval.**

## GitOps safety

Read-only Kubernetes operations are always appropriate for diagnosis:

```bash
kubectl get ...
kubectl describe ...
kubectl logs ...
```

Do not use `kubectl apply`, `kubectl edit`, or workload patches to create desired state. Fix Git and let ArgoCD reconcile it.

ArgoCD refresh/sync operations may be used to reconcile committed Git state, but they are not a substitute for changing Git.

## Validation

Before a normal PR:

```bash
just validate
```

GitHub Actions then validates the PR independently.

The temporary clean-room branch `refactor/gitops-cleanroom` is intentionally validated on **push** before any pull request is opened. It does not deploy to dev because ArgoCD dev tracks `main`, not this branch.

## Documentation rules

Documentation describes the repository; it must not become a second desired-state database.

- Current architecture belongs in `docs/architecture.md`.
- Do not maintain hand-written dashboards that duplicate live ArgoCD/Kubernetes state.
- Application docs are runbooks and design notes, not deployment truth.
- Historical post-mortems remain historical and are not rewritten to match current architecture.

### ADR immutability

An ADR records the decision that was made at that time.

Once an ADR is accepted, deprecated, or superseded, **do not rewrite its decision or rationale to match the present**. A changed decision requires a new ADR that supersedes the previous one. The ADR index may be updated to describe status and relationships without rewriting the historical record.

## Secrets

Never commit credentials, tokens, private keys, kubeconfigs, or copied secret values — including in documentation, examples, logs, agent instructions, and comments.

Secrets are materialized through External Secrets/OpenBao according to the current architecture. Gitleaks scans documentation as well as manifests.
