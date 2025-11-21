# Promote Configurations to Test/Staging/Prod Environments

## Why

**Current State:**
- All configurations exist for 4 environments (dev/test/staging/prod) in Git
- Only **dev environment** has been deployed and validated
- Terraform for test/staging/prod exists but never applied
- ArgoCD apps configured for all envs but test/staging/prod clusters don't exist yet

**Problems:**
- ❌ **Single point of failure**: All apps run only in dev (Hyper-V VM)
- ❌ **No validation path**: Can't test changes before prod
- ❌ **Prod not operational**: Physical nodes exist but cluster not provisioned

**Vision:**
Follow infrastructure maturity path: **dev (working)** → **test (next)** → **staging** → **prod (bare metal)**. Each environment validates the next, building confidence before prod deployment.

## What Changes

### Environment Progression

**Phase 1: Test Environment (Priority 1)**
- Nodes: carny, celesty, citrine (Hyper-V VMs, VLAN 111 + 209)
- Purpose: Validate changes before staging
- Infrastructure: 3 control planes HA (like dev)
- Terraform: Apply `terraform/environments/test/`
- Secrets: Create `.secrets/test/infisical-machine-identity.yml`

**Phase 2: Staging Environment (Priority 2)**
- Nodes: TBD (3 Hyper-V VMs, VLAN 111 + 210)
- Purpose: Pre-production validation with prod-like config
- Infrastructure: 3 control planes HA
- Terraform: Apply `terraform/environments/staging/`

**Phase 3: Prod Environment (Priority 3)**
- Nodes: Physical mini PCs (3 nodes, VLAN 111 + 201)
- Purpose: Production workloads
- Infrastructure: 3 control planes HA, production-grade resources
- Terraform: Apply `terraform/environments/prod/`

### Git Workflow Integration

```
dev (fast iteration)
  ↓ PR + validation
test (integration testing)
  ↓ PR + validation
staging (pre-prod)
  ↓ PR + validation + manual approval
main/prod (production)
```

### Validation Checklist per Environment

**Before promoting dev → test:**
- ✅ All ArgoCD apps Healthy in dev
- ✅ All ingresses accessible (https://*.dev.truxonline.com)
- ✅ Certificates valid (Let's Encrypt)
- ✅ PVCs bound and data persistent
- ✅ No manual kubectl commands required (GitOps only)

**Before promoting test → staging:**
- ✅ Same checks as dev, but on test environment
- ✅ Smoke tests pass (curl all ingresses)
- ✅ Applications functional (Home Assistant login works, etc.)

**Before promoting staging → prod:**
- ✅ Same checks as test
- ✅ Production certificates configured (letsencrypt-prod issuer)
- ✅ Backup plan documented
- ✅ Rollback plan tested

## Non-Goals

- **Not automating promotion**: Keep manual PR workflow (visibility + control)
- **Not deploying all apps to all envs immediately**: Start with infra apps only
- **Not changing test/staging to match dev exactly**: Each env can have different resource limits
- **Not replicating data**: Each env has its own databases/configs

## Testing Strategy

### Phase 1: Test Environment Bootstrap
1. Apply Terraform: `cd terraform/environments/test && terraform apply`
2. Wait for cluster ready (~30 min for full bootstrap)
3. Verify kubeconfig access
4. Check ArgoCD auto-deployed and accessible
5. Verify Cilium, Traefik, cert-manager operational

### Phase 2: Application Rollout to Test
1. Create PR: dev → test (all application changes)
2. Wait for GitHub Actions validation
3. Merge PR
4. Monitor ArgoCD sync in test cluster
5. Validate applications (smoke tests)

### Phase 3: Infisical Rollout
1. Create test Machine Identity in Infisical
2. Deploy via Terraform (bootstrap-infisical-terraform proposal)
3. Verify InfisicalSecret CRDs sync in test

### Phase 4: Staging Bootstrap & Rollout
1. Provision staging VMs (if not exists)
2. Apply Terraform for staging
3. Create PR: test → staging
4. Validate and merge
5. Monitor ArgoCD, verify apps

### Phase 5: Prod Bootstrap & Rollout
1. Ensure physical nodes ready (network, power, BIOS)
2. Apply Terraform for prod
3. Create PR: staging → main (prod)
4. **Manual approval required**
5. Merge and monitor carefully
6. Validate production ingresses (*.truxonline.com)

## Success Criteria

- ✅ Test environment fully operational (all apps Healthy)
- ✅ Staging environment operational
- ✅ Prod environment operational (bare metal)
- ✅ Git workflow enforced (dev → test → staging → main)
- ✅ Each environment has separate Infisical Machine Identity
- ✅ Certificates valid for each environment domain
- ✅ No manual kubectl commands needed after Terraform bootstrap
- ✅ Rollback tested in test/staging before prod deployment

## Rollback Plan

**Per environment:**
1. **Terraform destroy**: Tears down entire cluster
2. **Git revert + force push**: Reverts configuration to previous state
3. **ArgoCD rollback**: Sync to previous commit (git ref change)

**Important:** Destroying test/staging has no impact on dev/prod. Each environment is fully isolated.
