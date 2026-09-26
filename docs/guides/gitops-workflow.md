# GitOps Workflow

Vixens follows a pure trunk-based GitOps workflow.

## Canonical flow

```text
GitHub Issue
  ↓
feature/fix branch
  ↓
Pull Request + CI
  ↓
main
  ↓
ArgoCD dev
  ↓
dev-vYYYY.MM.PR
  ↓
promote-prod.yaml
  ↓
prod-vYYYY.MM.PR + prod-stable
  ↓
ArgoCD prod
```

`main` is the source of truth for development. Production tracks the mutable deployment alias `prod-stable`; every promotion also creates an immutable `prod-vYYYY.MM.PR` release tag.

`prod-working` is reserved as a manually chosen last-known-good recovery marker. Do not move it as part of routine promotion.

## Standard change

1. Start from the current `main`.
2. Check open PRs for overlapping work.
3. Create a feature or fix branch.
4. Make the smallest coherent declarative change.
5. Open a PR to `main`.
6. Let the repository CI validate YAML, Kustomize, ArgoCD structure, security and repository policy.
7. Merge only when required checks pass.
8. Let ArgoCD reconcile dev from `main`.
9. Validate the result in dev when runtime validation is relevant.
10. Promote the generated dev release through the GitHub Actions production workflow.

Example promotion:

```bash
gh workflow run promote-prod.yaml -f version=v2026.09.1234
```

The promotion workflow owns `prod-stable`. Do **not** create or force-move `prod-stable` manually during normal operation.

## Secrets

The canonical secret path is:

```text
OpenBao → ClusterSecretStore/openbao → ExternalSecret → Secret → workload
```

Do not add `InfisicalSecret` resources and never commit secret values to Git.

## Persistent cluster changes

Persistent configuration belongs in Git. `kubectl edit`, `kubectl apply` or direct ArgoCD mutations are not substitutes for a GitOps change.

Runtime commands are acceptable for inspection, diagnosis and temporary testing, provided the lasting desired state is represented in Git afterwards.

## Rollback

Preferred rollback is declarative:

1. identify the bad change;
2. revert it through a PR;
3. validate dev;
4. promote the resulting immutable release normally.

For a production emergency, `prod-working` exists as the explicit known-good reference. Emergency recovery must not silently replace the normal release history.

## Task tracking

GitHub Issues are the canonical backlog and incident/task tracker. PRs should reference the relevant issue when one exists.

## References

- [`WORKFLOW.md`](../../WORKFLOW.md) — canonical contribution workflow
- [`AGENTS.md`](../../AGENTS.md) — agent contract
- [`docs/guides/secret-management.md`](secret-management.md) — secret architecture
- [`docs/adr/017-pure-trunk-based-single-branch.md`](../adr/017-pure-trunk-based-single-branch.md) — trunk-based decision

Historical documents may mention Beads, Archon, Serena, Just or Infisical. Those references are not current workflow instructions.
