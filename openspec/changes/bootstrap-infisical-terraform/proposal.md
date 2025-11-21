# Bootstrap Infisical Machine Identity via Terraform

## Why

**Current State:**
- Infisical Machine Identity credentials (clientId/clientSecret) are **hardcoded in Git** in app manifests:
  - `apps/cert-manager-webhook-gandi/base/infisical-auth-secret.yaml`
  - `apps/synology-csi/infisical/base/infisical-auth-secret.yaml`
- Same credentials used across namespaces (not ideal but acceptable for homelab)
- Manual deployment required if secrets change

**Problems:**
- ❌ **GitHub exposure**: Machine Identity credentials visible in public repository
- ❌ **No rotation workflow**: Changing credentials requires Git commits + ArgoCD sync
- ❌ **Manual bootstrap**: Secrets must be applied after Terraform but before apps
- ❌ **Not DRY**: Same secret duplicated in multiple app directories

**Vision:**
Store Machine Identity credentials in `.secrets/<env>/infisical-machine-identity.yml` (gitignored) and deploy them automatically at the end of Terraform provisioning. This provides a pragmatic balance between security and simplicity for a homelab environment.

## What Changes

### Architecture

```
.secrets/
├── dev/
│   └── infisical-machine-identity.yml     # clientId + clientSecret
├── test/
│   └── infisical-machine-identity.yml
├── staging/
│   └── infisical-machine-identity.yml
└── prod/
    └── infisical-machine-identity.yml

terraform/modules/infisical-bootstrap/
├── main.tf           # kubectl_manifest resources
├── variables.tf      # environment input
└── versions.tf       # kubectl provider ~> 2.0

terraform/environments/dev/
└── main.tf           # Calls infisical-bootstrap module
```

### Secret Format

```yaml
# .secrets/dev/infisical-machine-identity.yml
clientId: "ee279e5e-82b6-476b-9643-093898807f35"
clientSecret: "ed8de635fd4b5818861e842fb1a03722bb7a35bda478b432d4d370609d12aefe"
```

### Terraform Module

```hcl
# terraform/modules/infisical-bootstrap/main.tf
locals {
  secrets_file = "${path.root}/../../../.secrets/${var.environment}/infisical-machine-identity.yml"
  secrets_data = yamldecode(file(local.secrets_file))

  # Namespaces requiring Infisical authentication
  target_namespaces = [
    "cert-manager",
    "synology-csi",
  ]
}

resource "kubectl_manifest" "infisical_universal_auth" {
  for_each = toset(local.target_namespaces)

  yaml_body = yamlencode({
    apiVersion = "v1"
    kind       = "Secret"
    metadata = {
      name      = "infisical-universal-auth"
      namespace = each.value
      labels = {
        "managed-by" = "terraform"
      }
    }
    type = "Opaque"
    stringData = {
      clientId     = local.secrets_data.clientId
      clientSecret = local.secrets_data.clientSecret
    }
  })

  depends_on = [
    # Wait for namespaces to exist
    var.argocd_bootstrap_complete
  ]
}
```

### Git Changes

**Remove hardcoded secrets from:**
- `apps/cert-manager-webhook-gandi/base/infisical-auth-secret.yaml` → DELETE
- `apps/synology-csi/infisical/base/infisical-auth-secret.yaml` → DELETE

**Update ArgoCD Applications:**
- Remove `infisical-auth-secret.yaml` from kustomization bases
- Keep only `InfisicalSecret` CRD (which references the Terraform-deployed secret)

### Workflow

```bash
# 1. Create secrets file (manual, one-time per environment)
mkdir -p .secrets/dev
cat > .secrets/dev/infisical-machine-identity.yml <<EOF
clientId: "ee279e5e-82b6-476b-9643-093898807f35"
clientSecret: "ed8de635fd4b5818861e842fb1a03722bb7a35bda478b432d4d370609d12aefe"
EOF

# 2. Deploy infrastructure (secrets deployed automatically)
cd terraform/environments/dev
terraform apply

# 3. ArgoCD syncs apps (InfisicalSecret CRDs use Terraform-deployed auth)
# No manual kubectl apply needed!
```

## Non-Goals

- **Not implementing external vaults** (1Password, HashiCorp Vault) - overkill for homelab
- **Not rotating secrets automatically** - manual process acceptable
- **Not using Sealed Secrets** - adds complexity without value here
- **Not deploying Infisical Operator via Terraform** - remains in ArgoCD (Phase 2)

## Testing Strategy

### Phase 1: Module Creation & Dev Validation
1. Create `terraform/modules/infisical-bootstrap` module
2. Add module call to `terraform/environments/dev/main.tf`
3. Move dev credentials to `.secrets/dev/infisical-machine-identity.yml`
4. Run `terraform plan` - should show 2 new secrets (cert-manager, synology-csi)
5. Run `terraform apply`
6. Validate secrets exist: `kubectl get secret -n cert-manager infisical-universal-auth`

### Phase 2: App Manifest Cleanup
1. Remove `infisical-auth-secret.yaml` from app bases
2. Update `kustomization.yaml` to remove deleted files
3. Commit and push to dev branch
4. Monitor ArgoCD sync - apps should remain Healthy
5. Validate InfisicalSecret CRDs still work (check synced app secrets)

### Phase 3: Multi-Environment Rollout
1. Create `.secrets/test/infisical-machine-identity.yml` (test Machine Identity)
2. Deploy test environment with Terraform
3. Validate test cluster apps functional
4. Repeat for staging and prod

### Phase 4: Validation
- ✅ No credentials in Git history (use `rg "clientId|clientSecret" apps/`)
- ✅ `.secrets/` directory gitignored
- ✅ Terraform state encrypted (S3 backend with encryption)
- ✅ Apps deploy successfully with Infisical secrets synced

## Success Criteria

- ✅ Infisical Machine Identity credentials stored in `.secrets/<env>/` (gitignored)
- ✅ Terraform module deploys secrets to required namespaces automatically
- ✅ No hardcoded credentials in Git repository
- ✅ ArgoCD applications remain Healthy after migration
- ✅ InfisicalSecret CRDs successfully authenticate and sync secrets
- ✅ All 4 environments (dev/test/staging/prod) use separate Machine Identities
- ✅ Secrets deployed before ArgoCD applications need them (dependency ordering)

## Rollback Plan

If secrets deployment fails:

1. **Immediate**: Restore hardcoded secrets in app bases
   ```bash
   git revert <commit-removing-hardcoded-secrets>
   git push origin dev --force
   ```

2. **ArgoCD**: Force sync applications to restore hardcoded secrets
   ```bash
   argocd app sync cert-manager-secrets --force
   argocd app sync synology-csi-secrets --force
   ```

3. **Terraform**: Remove infisical-bootstrap module call
   ```bash
   terraform state rm 'module.infisical_bootstrap.kubectl_manifest.infisical_universal_auth["cert-manager"]'
   terraform state rm 'module.infisical_bootstrap.kubectl_manifest.infisical_universal_auth["synology-csi"]'
   ```

Git history preserves original hardcoded approach, rollback is straightforward.
