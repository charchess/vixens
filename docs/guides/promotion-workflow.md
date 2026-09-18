# Production promotion

Production tracks `prod-stable`; development tracks `main`.

Every squash merge to `main` is automatically snapshotted by `auto-tag-dev.yaml` as:

```text
dev-vYYYY.MM.<PR-number>
```

## Promote a validated snapshot

After validating that exact state in development:

```bash
just promote-prod v2026.09.3271
```

Equivalent command:

```bash
gh workflow run promote-prod.yaml -f version=v2026.09.3271
```

The workflow:

1. verifies that `dev-v2026.09.3271` exists;
2. resolves the annotated dev tag to its underlying commit;
3. creates `prod-v2026.09.3271`;
4. moves `prod-stable` to the promoted commit;
5. publishes the production release/SBOM.

ArgoCD production then reconciles `prod-stable`.

## Validate production

Use ArgoCD and read-only Kubernetes checks to confirm the relevant applications are Synced/Healthy and their workloads are healthy.

## Rollback

Rollback is a production decision, not an automatic side effect. Select a known production release and use the repository's controlled recovery procedure.

`prod-working` is a manually managed rescue reference. Never move it without explicit owner approval.

Manual force-moving of `prod-stable` is reserved for an explicitly chosen recovery action, not the normal promotion workflow.
