# cert-manager-webhook-gandi - Infisical Integration

This directory contains the Infisical integration for cert-manager-webhook-gandi secrets.

## Architecture

Secrets are automatically synchronized from Infisical (self-hosted) to Kubernetes using the Infisical Operator.

### Infisical Configuration

**Instance:** `http://192.168.111.69:8085` (self-hosted)

**Project Structure:**
```
Project: vixens
└── Environment: dev
    └── Path: /cert-manager
        └── api-token: <Gandi LiveDNS API Token>
```

**Machine Identity:**
- Name: `vixens-dev-k8s-operator`
- Auth Method: Universal Auth
- Permissions: Read access to `/cert-manager` path in `dev` environment

### Kubernetes Resources

**1. infisical-auth-secret.yaml**
- Contains Machine Identity credentials (Client ID + Client Secret)
- Used by InfisicalSecret to authenticate with Infisical API
- Namespace: `cert-manager`

**2. gandi-infisical-secret.yaml**
- InfisicalSecret CRD that syncs secrets from Infisical to Kubernetes
- Creates `gandi-credentials` secret in `cert-manager` namespace
- Auto-sync every 60 seconds
- Syncs from path: `/cert-manager`

**Generated Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gandi-credentials
  namespace: cert-manager
type: Opaque
data:
  api-token: <base64-encoded Gandi API token>
```

### How It Works

1. **Infisical Operator** runs in `infisical-operator-system` namespace
2. **InfisicalSecret CRD** (`gandi-credentials-sync`) defines the sync configuration
3. **Operator authenticates** to Infisical using Machine Identity credentials
4. **Secrets are fetched** from Infisical project `vixens`, environment `dev`, path `/cert-manager`
5. **Kubernetes Secret** (`gandi-credentials`) is automatically created/updated
6. **cert-manager ClusterIssuers** reference this secret for DNS-01 challenges

### Secret Rotation

To rotate the Gandi API token:

1. Update the secret in Infisical UI:
   - Project: `vixens`
   - Environment: `dev`
   - Path: `/cert-manager`
   - Secret: `api-token`

2. Wait up to 60 seconds for automatic sync (or force reconciliation):
   ```bash
   kubectl annotate infisicalsecret gandi-credentials-sync \
     -n cert-manager \
     --overwrite \
     reconcile="$(date +%s)"
   ```

3. Verify the secret was updated:
   ```bash
   kubectl get secret gandi-credentials -n cert-manager -o yaml
   ```

### Troubleshooting

**Check InfisicalSecret status:**
```bash
kubectl get infisicalsecret -n cert-manager gandi-credentials-sync -o yaml
```

**Check sync conditions:**
```bash
kubectl get infisicalsecret -n cert-manager gandi-credentials-sync \
  -o jsonpath='{.status.conditions[?(@.type=="secrets.infisical.com/ReadyToSyncSecrets")]}'
```

**Check Infisical Operator logs:**
```bash
kubectl logs -n infisical-operator-system \
  deployment/infisical-opera-controller-manager \
  --tail=50
```

**Common Issues:**

1. **Authentication failed (401):**
   - Verify Machine Identity credentials in `infisical-auth-secret`
   - Check Machine Identity is not disabled in Infisical UI

2. **Project not found (404):**
   - Verify `projectSlug` is correct (`vixens`)
   - Check Machine Identity has access to the project

3. **Secret not syncing:**
   - Verify secret exists in Infisical at path `/cert-manager`
   - Check `resyncInterval` setting (default: 60s)
   - Force reconciliation with annotation

### Security Notes

- ✅ No secrets stored in Git
- ✅ Centralized secret management via Infisical UI
- ✅ Machine Identity uses Universal Auth (not static tokens)
- ✅ Secrets isolated in dedicated path `/cert-manager`
- ✅ Automatic rotation support
- ✅ Audit trail in Infisical

### References

- [Infisical Kubernetes Operator Docs](https://infisical.com/docs/integrations/platforms/kubernetes)
- [cert-manager Webhook Gandi](https://github.com/bwolf/cert-manager-webhook-gandi)
- [ADR 007: Infisical Secrets Management](../../../docs/adr/007-infisical-secrets-management.md)
