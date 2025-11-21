# Setup GitHub Actions Validation

## Why

**Current State:**
- No CI/CD validation on pull requests
- Manual validation required (yamllint, terraform validate, kustomize build)
- Risk of broken configs merged to test/staging/prod

**Problems:**
- ❌ **No automated checks**: Syntax errors can reach non-dev environments
- ❌ **Inconsistent validation**: Manual checks may be skipped
- ❌ **Slow feedback loop**: Errors discovered after merge, not before

**Vision:**
GitHub Actions workflow runs automatically on all PRs, validating YAML syntax, Terraform, Kustomize, and OpenSpec before merge. Fast feedback (< 2 minutes), block merge on failure.

## What Changes

### GitHub Actions Workflow

Create `.github/workflows/validate.yml`:

```yaml
name: Validate

on:
  pull_request:
    branches: [test, staging, main]
  push:
    branches: [dev]

jobs:
  yaml-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
      - run: pip install yamllint
      - run: yamllint -c .yamllint.yaml apps/ argocd/ terraform/

  terraform-validate:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        environment: [dev, test, staging, prod]
    steps:
      - uses: actions/checkout@v4
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: '1.5.0'
      - run: |
          cd terraform/environments/${{ matrix.environment }}
          terraform init -backend=false
          terraform validate
          terraform fmt -check -recursive

  kustomize-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: azure/setup-kubectl@v4
      - run: kubectl version --client
      - run: |
          for app in apps/*/overlays/dev; do
            echo "Building $app"
            kustomize build $app
          done

  openspec-validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: |
          curl -sSL https://github.com/openspec-dev/openspec/releases/latest/download/openspec-linux-amd64 -o /usr/local/bin/openspec
          chmod +x /usr/local/bin/openspec
      - run: openspec validate --all --strict
```

### YAMLlint Configuration

Create `.yamllint.yaml`:

```yaml
extends: default

rules:
  line-length:
    max: 120
    level: warning
  indentation:
    spaces: 2
  comments:
    min-spaces-from-content: 1
```

### Branch Protection Rules

Configure in GitHub UI:
- **test/staging/main**: Require status checks before merge
  - yamllint
  - terraform-validate (all 4 matrix jobs)
  - kustomize-build
  - openspec-validate
- **dev**: No restrictions (force push allowed)

## Non-Goals

- **Not deploying automatically**: Validation only, ArgoCD handles deployment
- **Not testing applications**: No integration tests (manual for now)
- **Not validating secrets**: .secrets/ is gitignored, can't validate
- **Not running on dev branch PRs**: dev is fast iteration, validate on push only

## Testing Strategy

### Phase 1: Create Workflow
1. Create `.github/workflows/validate.yml`
2. Create `.yamllint.yaml`
3. Push to dev branch
4. Verify workflow runs successfully

### Phase 2: Test Failure Scenarios
1. Introduce YAML syntax error → yamllint should fail
2. Introduce Terraform syntax error → terraform-validate should fail
3. Introduce invalid Kustomize reference → kustomize-build should fail
4. Create invalid OpenSpec → openspec-validate should fail

### Phase 3: Branch Protection
1. Configure branch protection rules in GitHub
2. Create PR dev → test with failing validation
3. Verify merge blocked
4. Fix validation, verify merge allowed

## Success Criteria

- ✅ GitHub Actions workflow runs on all PRs
- ✅ yamllint validates all YAML files
- ✅ Terraform validates all 4 environments
- ✅ Kustomize builds all applications successfully
- ✅ OpenSpec validation passes
- ✅ Failed validation blocks PR merge
- ✅ Workflow completes in < 3 minutes
- ✅ Branch protection enforced on test/staging/main

## Rollback Plan

1. Remove branch protection rules (allow bypass)
2. Merge failing PR if critical
3. Fix validation issues
4. Re-enable branch protection
