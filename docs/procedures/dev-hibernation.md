# Dev Environment Hibernation

## Principle

Dev hibernation is represented in Git. Applications that are not needed should normally stay present in ArgoCD and declare `replicas: 0` in their dev overlay.

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base

patches:
  - patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: <app-name>
      spec:
        replicas: 0
```

This keeps the application visible, reconciled and reproducible while releasing compute resources.

## Permanent wake or sleep

A lasting change must follow the normal GitOps workflow:

1. create/update the replicas patch in the dev overlay;
2. open a PR to `main`;
3. let CI validate it;
4. merge;
5. let ArgoCD reconcile dev.

To wake an application permanently, set the desired replica count or remove the hibernation patch when the base already expresses the intended count.

## Temporary testing

A temporary runtime scale is acceptable for short diagnostic/testing sessions, but it is **not** persistent desired state.

Example:

```bash
kubectl -n <namespace> scale deployment/<app-name> --replicas=1
kubectl -n <namespace> wait --for=condition=available deployment/<app-name> --timeout=300s
```

ArgoCD may reconcile the deployment back to the Git value. Do not disable self-heal or mutate ArgoCD configuration as a routine testing workflow.

If an application needs to remain active after testing, represent that decision in Git through a PR.

## Verification

```bash
kubectl get deployments -A
kubectl -n argocd get applications
```

For a specific application:

```bash
kubectl -n <namespace> get deployment/<app-name>
kubectl -n <namespace> get pods -l app=<app-name>
```

## Tracking

GitHub Issues are the canonical place for temporary operational follow-up, incidents and planned hibernation changes. Do not use Beads/Archon/Just state as repository truth.

## Rules

- Production hibernation changes always go through Git/PR.
- Infrastructure components should not be hibernated casually.
- Scaling to zero does not remove PVCs or persisted data.
- Never delete storage as part of a hibernation operation unless that deletion is a separate, explicit change.
- Direct runtime changes are diagnostic/temporary only; Git remains authoritative.

## References

- [`WORKFLOW.md`](../../WORKFLOW.md)
- [`AGENTS.md`](../../AGENTS.md)
- [`docs/guides/gitops-workflow.md`](../guides/gitops-workflow.md)
