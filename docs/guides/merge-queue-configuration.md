# GitHub merge queue — current status and future enablement

**Status:** Not active / not enforced
**Last verified:** 2026-10-06

## Current repository state

`charchess/vixens` is currently owned by the personal GitHub account `charchess`.
The active `main` ruleset requires:

- changes through a Pull Request;
- the `Validation Summary` required status check;
- strict/up-to-date status checks;
- non-fast-forward/deletion protection.

The active ruleset does **not** require a GitHub Merge Queue.

`.github/workflows/merge-queue.yaml` exists and listens for the `merge_group` event.
It is future-ready validation logic, but its presence does not mean a queue is
enabled. Without GitHub emitting `merge_group`, that workflow is dormant.

Historical issue #2233 records the earlier investigation/migration idea and the
account/feature constraint. Do not treat its migration plan as completed merely
because the issue is closed; verify repository ownership/rulesets before changing
the workflow contract.

## Normal merge workflow today

```text
short-lived branch
  → Pull Request
  → applicable CI
  → required Validation Summary
  → merge/auto-merge allowed by repository rules
  → main
```

Do not instruct contributors to click "Merge when ready" or wait for merge-group
checks as if that were mandatory today.

## Future enablement

If merge queue becomes desirable, treat it as an explicit repository-governance
change:

1. confirm the GitHub account/plan/repository ownership supports the feature;
2. update the `main` ruleset to require merge queue;
3. verify the existing `merge_group` workflow against current CI semantics;
4. test one real PR through the queue;
5. update `WORKFLOW.md`, this guide and agent adapters only after the feature is
   actually active;
6. document any repository ownership migration separately.

Do not add an emergency bypass recipe to this guide. Exceptional repository-rule
changes require explicit operator intent and should remain auditable.

## Source of truth

- root `WORKFLOW.md` — current contribution/merge lifecycle;
- active GitHub repository rulesets — enforced merge policy;
- `.github/workflows/validate.yaml` — authoritative PR validation orchestration;
- `.github/workflows/merge-queue.yaml` — dormant/future merge-group validation;
- #2233 — historical context for the ownership/feature investigation.
