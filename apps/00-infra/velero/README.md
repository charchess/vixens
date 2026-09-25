# Velero — Kubernetes backup

Velero manages Kubernetes backup/restore resources and persistent-volume backup workflows used by Vixens.

## Secrets

Velero credentials are projected from OpenBao through External Secrets Operator:

```text
OpenBao
  ↓
ClusterSecretStore/openbao
  ↓
ExternalSecret
  ↓
Secret/velero + Secret/velero-repo-credentials
  ↓
Velero
```

The GitOps manifests live under:

```text
apps/00-infra/velero/external-secrets/
```

Production reads the canonical environment path used by the current manifests:

```text
vixens/prod/apps/00-infra/velero
```

Do not put credentials in Git and do not recreate the retired Infisical integration.

## Deployment structure

- Velero itself is Helm-managed by ArgoCD.
- `velero-secrets` is a separate ArgoCD Application so credentials are materialized before Velero needs them.
- Environment-specific values and schedules live in the corresponding GitOps values/overlays; inspect current manifests rather than relying on copied values in this README.

## Operations

Examples of read/operational commands:

```bash
velero backup get
velero schedule get
velero backup describe <backup>
velero backup logs <backup>
velero restore get
```

Manual backup when operationally required:

```bash
velero backup create <name> --include-namespaces=<namespace>
```

Restore:

```bash
velero restore create --from-backup <backup>
```

A manual backup/restore is an operational action; persistent schedule/configuration changes still belong in Git.

## Validation

After GitOps changes, verify:

```bash
kubectl -n velero get externalsecret
kubectl -n velero get secret velero velero-repo-credentials
velero backup-location get
velero schedule get
```

Check readiness/status only; never print secret values into terminals, logs, PRs or documentation.

## References

- `docs/guides/secret-management.md`
- `docs/adr/018-openbao-external-secrets-and-nas-fqdn.md`
- https://velero.io/docs/
