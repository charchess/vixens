---
name: vixens-secrets
description: >-
  Vixens secrets-management guide for OpenBao and External Secrets Operator.
  Use when diagnosing ExternalSecret sync, ClusterSecretStore/OpenBao access,
  secret rotation, missing Kubernetes Secrets, or adding secret-backed apps.
  Trigger on: "secret", "OpenBao", "ExternalSecret", "External Secrets", "credentials", "rotation".
argument-hint: "[app-name or issue description]"
license: MIT
compatibility: opencode
metadata:
  domain: secrets-management
  audience: homelab-operators
---

# Vixens secrets management

Follow `WORKFLOW.md` and `AGENTS.md` first. This skill is an optional troubleshooting adapter; it does not redefine repository workflow.

**Focus:** $ARGUMENTS

## Canonical architecture

```text
OpenBao (secret values)
    ↓
ClusterSecretStore/openbao
    ↓
ExternalSecret
    ↓
Kubernetes Secret
    ↓
Pod / controller
```

The Git repository stores **references and desired state only**, never secret values.

Canonical resources live under:

- `apps/00-infra/openbao/` — `ClusterSecretStore` wiring;
- `argocd/.../external-secrets.yaml` — External Secrets Operator deployment;
- application manifests — `ExternalSecret` resources using `secretStoreRef.name: openbao`.

Read the current manifests before assuming endpoints, authentication method, paths, or namespaces: these details evolve and Git is the source of truth.

## Standard ExternalSecret pattern

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-secrets
  namespace: app
spec:
  refreshInterval: 60s
  secretStoreRef:
    name: openbao
    kind: ClusterSecretStore
  target:
    name: app-secrets
    creationPolicy: Owner
  dataFrom:
    - extract:
        key: vixens/prod/apps/category/app
```

Use the repository's existing path convention for the target environment rather than inventing a new hierarchy.

## Diagnose sync

Observation commands are allowed; persistent fixes belong in Git/OpenBao.

```bash
# Store health
kubectl get clustersecretstore openbao
kubectl describe clustersecretstore openbao

# ExternalSecret health
kubectl get externalsecret -A
kubectl -n <namespace> describe externalsecret <name>

# Resulting Secret metadata / key names
kubectl -n <namespace> get secret <name>
kubectl -n <namespace> get secret <name> -o jsonpath='{.data}' | jq 'keys'

# ESO controller logs
kubectl -n external-secrets get pods
kubectl -n external-secrets logs deploy/external-secrets --tail=200
```

Do **not** print or paste secret values unless strictly required for a controlled diagnosis. Prefer checking existence, key names, resource versions, checksums, or application behavior.

## Force a refresh

Prefer waiting for the configured `refreshInterval`. If an immediate reconciliation is required for a test, use the External Secrets reconciliation mechanism supported by the deployed ESO version and treat it as a runtime diagnostic action, not a replacement for Git/OpenBao state.

Always verify the currently deployed ESO documentation/version before relying on a specific annotation or CLI behavior.

## Rotation workflow

1. Create/rotate the value in OpenBao using an authorized operator path.
2. Confirm the relevant `ExternalSecret` becomes `Ready`.
3. Confirm the Kubernetes `Secret` resource changed without revealing the value.
4. Restart/reconcile only workloads that do not automatically consume updated Secret volumes/data.
5. Validate the application.
6. Revoke the old credential at its upstream provider/database/service.
7. Document unusual rotation steps in the application runbook.

Never commit temporary secret values to Git, issues, PR descriptions, comments, logs, skills, examples, or test fixtures.

## Adding a secret-backed application

Before creating a new pattern:

1. search for an existing `ExternalSecret` in the same application category;
2. reuse `ClusterSecretStore/openbao`;
3. keep environment-specific remote paths in overlays/patches only when necessary;
4. make the generated Kubernetes Secret name explicit;
5. ensure the application depends on the Secret without embedding its content;
6. validate Kustomize/CI before merge.

## Security rules

- No real tokens, passwords, client secrets, API keys, private keys, cookies, or bearer headers in Git.
- No credentials in agent memory/skills/examples.
- A credential ever committed to a public repository is considered compromised and must be revoked/rotated; deleting the current file is insufficient because Git history persists.
- Do not expose Secret `.data` values in GitHub Issues or PRs.
- Secret-store authentication should use least privilege and short-lived identities where the platform supports it.

## Historical Infisical material

Older ADRs, audits and reports may mention Infisical/`InfisicalSecret`. Treat those as historical unless the current manifests explicitly prove otherwise.

Do not copy legacy Infisical examples into new manifests. The active model is OpenBao + External Secrets Operator.

## References

- `WORKFLOW.md`
- `AGENTS.md`
- `docs/adr/018-openbao-external-secrets-and-nas-fqdn.md`
- `apps/00-infra/openbao/base/cluster-secret-store.yaml`
