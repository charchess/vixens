# AdGuard Home - Infisical Credentials Update

## Context

AdGuard Home was stuck in CrashLoopBackOff due to:
1. **DNS dependency cycle**: AdGuard provides DNS but needs DNS to restore from S3
2. **Invalid S3 credentials**: Old MinIO credentials were invalid

## Fix Applied

1. Changed `LITESTREAM_ENDPOINT` from DNS to IP
2. Created new MinIO service account
3. Temporarily disabled InfisicalSecret (using static Secret instead)

## TODO: Update Infisical Vault

Access Infisical UI: http://192.168.111.69:8085

**Login**: admin@truxonline.com / S0d0m!e69

**Update these secrets:**
- Project: `vixens`
- Environment: `prod`
- Path: `/apps/40-network/adguard-home`

**Get new credentials from Kubernetes Secret:**
```bash
kubectl -n networking get secret adguard-home-secrets -o yaml
```

Update in Infisical:
- `LITESTREAM_ENDPOINT` (use IP instead of DNS)
- `LITESTREAM_ACCESS_KEY_ID` (new MinIO service account)
- `LITESTREAM_SECRET_ACCESS_KEY` (new MinIO service account)

## After Infisical Update

Re-enable InfisicalSecret:
```bash
mv apps/40-network/adguard-home/base/infisical-secret.yaml.disabled \
   apps/40-network/adguard-home/base/infisical-secret.yaml

# Edit base/kustomization.yaml - uncomment infisical-secret.yaml
git add apps/40-network/adguard-home/base/
git commit -m "fix(adguard-home): re-enable InfisicalSecret with updated credentials"
git push origin main

# Delete static Secret (InfisicalSecret will recreate it)
kubectl -n networking delete secret adguard-home-secrets
```
