---
name: vixens-gitops
description: >-
  Vixens GitOps workflow adapter. Use for PRs, merges, releases, production
  promotion and Git workflow questions. This skill delegates policy to
  WORKFLOW.md and AGENTS.md and must never redefine them.
disable-model-invocation: true
license: MIT
compatibility: opencode
metadata:
  domain: gitops
  audience: homelab-operators
---

# Vixens GitOps adapter

Read `WORKFLOW.md` and `AGENTS.md` first. They are authoritative.

This skill is only a convenience layer for the current GitHub/GitOps workflow. If
anything here appears to contradict the canonical files or executable workflows,
stop and follow the canonical/executable state instead.

## Before changing anything

1. Read the issue and acceptance criteria.
2. Refresh the current GitHub `main`.
3. Inspect open PRs for overlapping work.
4. Check the issue in `vixens roadmap` when it is planned.
5. Create a short-lived branch from current `main`.
6. Run applicable local/pre-commit validation before pushing.

## Pull-request flow

```bash
git fetch origin
git switch main
git pull --ff-only
git switch -c feat/<issue>-<slug>

# edit + applicable local validation
pre-commit run

git push -u origin HEAD
gh pr create --base main --fill
```

CI is the authoritative merge gate. Do not bypass a failing check with a live
cluster mutation.

## Production promotion

Production promotion is **never** performed by moving tags manually.

After the chosen `dev-v*` snapshot has been validated and the user/operator has
explicitly authorized that exact candidate:

```bash
gh workflow run promote-prod.yaml -f version=vYYYY.MM.<PR>
```

The workflow owns creation/verification of `prod-v*` and movement of
`prod-stable`.

Do not run the promotion merely because a PR merged, CI is green or a dev tag
exists. Automated agents require explicit human authorization for the exact
candidate.

## Known-good marker

`prod-working` is optional and manual. Move it only through:

```bash
gh workflow run mark-prod-working.yaml -f version=vYYYY.MM.<PR>
```

and only after explicit authorization for the already validated production
release.

Never use `git tag -f prod-stable`, `git tag -f prod-working`, or force-push
those tags as a normal workflow.

## ArgoCD

Read-only observation is appropriate for normal verification.

Before any invasive ArgoCD recovery/sync action, load
`vixens-argocd-safety` and diagnose the failing layer first. An ArgoCD patch is
not automatically safe merely because ArgoCD is a GitOps controller.

Persistent desired-state fixes belong in Git.

## Rollback

Normal rollback is declarative: revert/fix in Git through a PR, validate dev, then
promote the resulting immutable snapshot.

Emergency recovery is exceptional. Follow `WORKFLOW.md`, the production
promotion/rollback runbook and explicit operator instructions. Do not improvise by
force-moving `prod-stable`.

## Useful read-only commands

```bash
gh pr list --state open
gh pr checks <PR_NUMBER>
gh run list --limit 10
kubectl -n argocd get applications
```

## References

- `WORKFLOW.md`
- `AGENTS.md`
- `docs/guides/promotion-workflow.md`
- `.github/workflows/promote-prod.yaml`
- `.github/workflows/mark-prod-working.yaml`
