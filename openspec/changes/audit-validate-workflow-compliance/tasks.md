# Tasks for Audit & Fix Validate Workflow Compliance

## Phase 1: Extract yamllint Config (4 tasks)

### Task 1.1: Create .yamllint.yaml in repo root ✅
**Owner**: DevOps
**Status**: todo
**Description**: Extract inline yamllint config from workflow to dedicated file
**Acceptance Criteria**:
- File `.yamllint.yaml` exists at repo root
- Contains exact config from workflow (extends: default, rules section)
- Includes yaml-files section (*.yaml, *.yml, .yamllint)

### Task 1.2: Test yamllint with new config ✅
**Owner**: DevOps
**Status**: todo
**Description**: Verify new config file produces identical results to inline config
**Acceptance Criteria**:
- Run: `find apps argocd -name "*.yaml" -o -name "*.yml" | xargs yamllint -c .yamllint.yaml`
- Output matches previous test (30 warnings, 0 errors)
- No new errors introduced

### Task 1.3: Update workflow to use config file ✅
**Owner**: DevOps
**Status**: todo
**Description**: Modify `.github/workflows/validate.yaml` to reference `.yamllint.yaml`
**Acceptance Criteria**:
- Workflow line 26-73 replaced with simplified command
- Uses `-c .yamllint.yaml` flag
- No inline config remains in workflow

### Task 1.4: Commit yamllint changes ✅
**Owner**: DevOps
**Status**: todo
**Description**: Commit new config file and updated workflow
**Acceptance Criteria**:
- Git commit includes `.yamllint.yaml` and `.github/workflows/validate.yaml`
- Commit message: "refactor(ci): Extract yamllint config to .yamllint.yaml"

## Phase 2: Create Local Validation Script (3 tasks)

### Task 2.1: Create scripts/validate-local.sh ✅
**Owner**: DevOps
**Status**: todo
**Description**: Create local validation script mimicking GitHub Actions
**Acceptance Criteria**:
- File `scripts/validate-local.sh` exists
- Executable permissions (chmod +x)
- Contains 4 validation jobs (yaml-lint, kube-structure, argocd-validate, openspec)
- Handles missing kubeval gracefully

### Task 2.2: Test script on current codebase ✅
**Owner**: DevOps
**Status**: todo
**Description**: Execute script and verify all validations pass
**Acceptance Criteria**:
- Run: `./scripts/validate-local.sh`
- Exit code 0 (success)
- All 4 jobs complete successfully
- Output shows "✅ All validations passed!"

### Task 2.3: Commit validation script ✅
**Owner**: DevOps
**Status**: todo
**Description**: Add script to repository
**Acceptance Criteria**:
- Git commit includes `scripts/validate-local.sh`
- Script has executable permissions in Git
- Commit message: "feat(ci): Add local validation script"

## Phase 3: Validate GitHub Actions (2 tasks)

### Task 3.1: Push changes to dev branch ✅
**Owner**: DevOps
**Status**: todo
**Description**: Push commits and trigger workflow
**Acceptance Criteria**:
- Changes pushed to origin/dev
- GitHub Actions workflow triggered automatically
- Workflow run appears in Actions tab

### Task 3.2: Verify workflow passes ✅
**Owner**: DevOps
**Status**: todo
**Description**: Confirm workflow still passes with new config
**Acceptance Criteria**:
- Workflow completes with success status
- All 5 jobs pass (yaml-lint, kube-structure, argocd-validate, security-scan, branch-flow)
- No new errors or warnings

## Phase 4: Update Documentation (2 tasks)

### Task 4.1: Add validation section to CLAUDE.md ✅
**Owner**: DevOps
**Status**: todo
**Description**: Document validation rules and local testing
**Acceptance Criteria**:
- New section "## Code Validation" added
- Lists all 5 validation jobs with descriptions
- Documents local testing command (`./scripts/validate-local.sh`)
- Shows current compliance status (all files passing)

### Task 4.2: Commit documentation updates ✅
**Owner**: DevOps
**Status**: todo
**Description**: Finalize documentation changes
**Acceptance Criteria**:
- Git commit includes updated CLAUDE.md
- Commit message: "docs: Add code validation section to CLAUDE.md"

## Summary

- **Total Tasks**: 11
- **Phases**: 4 (extract config, create script, validate workflow, document)
- **Estimated Time**: 2-3 hours
- **Dependencies**: Phase 2 depends on Phase 1, Phase 3 depends on Phase 1+2
- **Risk**: Low (no breaking changes, pure refactoring)
