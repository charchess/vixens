# Migrate Synology CSI Secrets to Isolated Infisical Path

## Why

Currently, Synology CSI secrets are stored at the root path `/` in Infisical, which:
- Causes potential naming conflicts with other applications
- Violates the isolation principle established for cert-manager (which uses `/cert-manager`)
- Makes permission management less granular (Machine Identity has access to all root secrets)
- Reduces clarity in secret organization

Following the pattern established with cert-manager, each application should have its own isolated path in Infisical.

## What Changes

- Create `/synology-csi` path in Infisical UI for dev environment
- Move `synology-csi-client-info` secret from `/` to `/synology-csi`
- Update InfisicalSecret CRD `secretsPath` from `/` to `/synology-csi`
- Test Synology CSI functionality with the new path
- Document the migration pattern for future applications
- Update specs to reflect implemented path isolation

## Impact

- **Breaking (temporary)**: During migration, Synology CSI pods may need restart to pick up secret from new path
- **Security**: Improved isolation and permission granularity
- **Consistency**: Aligns with established cert-manager pattern
- **Future**: Sets clear precedent for all new applications to use isolated paths

## Non-Goals

- Migrating to test/staging/prod environments (dev only for now)
- Changing secret content or format
- Modifying Synology CSI application code
