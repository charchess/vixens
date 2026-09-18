# Secret management

Vixens uses **External Secrets Operator** with an OpenBao-backed `ClusterSecretStore` named `openbao`.

Secret values are never committed to Git.

## Application pattern

A typical application resource is:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: example-secrets
  namespace: example
spec:
  refreshInterval: 60s
  secretStoreRef:
    name: openbao
    kind: ClusterSecretStore
  target:
    name: example-secrets
    creationPolicy: Owner
    deletionPolicy: Retain
  dataFrom:
    - extract:
        key: vixens/dev/apps/60-services/example
```

Production overlays must reference the corresponding production path.

Use existing current manifests as the implementation pattern; for example,
`apps/10-home/homeassistant/base/infisical-secret.yaml` has a legacy filename but contains a current `ExternalSecret`.

## Bootstrap boundary

Credentials required to let External Secrets authenticate to OpenBao are bootstrap material and are not stored in this repository.

## Safety

Never paste secret values into:

- manifests
- Markdown
- GitHub Issues or PR descriptions
- logs committed for troubleshooting
- agent instructions or examples

Gitleaks scans documentation as well as manifests.

The historical Infisical architecture remains documented in its original ADR for audit purposes; ADR-029 is the current decision.
