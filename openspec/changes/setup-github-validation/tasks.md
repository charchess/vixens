# Tasks - Setup GitHub Actions Validation

## Phase 1: Create Workflow Files
- [ ] Create `.github/workflows/validate.yml` (yaml-lint, terraform-validate, kustomize-build, openspec-validate jobs)
- [ ] Create `.yamllint.yaml` configuration
- [ ] Commit and push to dev branch

## Phase 2: Verify Workflow Runs
- [ ] Check GitHub Actions tab for workflow run
- [ ] Verify all 4 jobs pass successfully
- [ ] Check execution time (< 3 minutes target)

## Phase 3: Test Validation Failures
- [ ] Introduce YAML syntax error in apps/whoami/base/deployment.yaml
- [ ] Push to dev, verify yamllint job fails
- [ ] Fix error, verify job passes
- [ ] Repeat for terraform-validate (invalid HCL)
- [ ] Repeat for kustomize-build (invalid reference)
- [ ] Repeat for openspec-validate (invalid spec)

## Phase 4: Configure Branch Protection
- [ ] GitHub Settings → Branches → Add rule for `test`
- [ ] Enable "Require status checks" with: yaml-lint, terraform-validate, kustomize-build, openspec-validate
- [ ] Enable "Require branches to be up to date"
- [ ] Repeat for `staging` and `main` branches
- [ ] Keep `dev` branch unprotected (force push allowed)

## Phase 5: Test Branch Protection
- [ ] Create PR dev → test with failing validation
- [ ] Verify "Merge" button disabled
- [ ] Fix validation errors
- [ ] Verify "Merge" button enabled after checks pass

## Phase 6: Documentation
- [ ] Update CLAUDE.md with CI/CD section
- [ ] Document GitHub Actions workflow
- [ ] Document branch protection rules
- [ ] Add "Status Badge" to README.md

---
**Estimated Time:** 1-2 hours
**Dependencies:** GitHub repository with Actions enabled
