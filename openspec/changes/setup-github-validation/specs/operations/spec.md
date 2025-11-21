# Operations Specification - GitHub Validation

## ADDED Requirements

### Requirement: Pull Requests SHALL Be Validated Before Merge

All pull requests to test/staging/main branches SHALL pass automated validation before merge is allowed.

#### Scenario: PR with valid changes passes checks
**GIVEN** a PR from dev to test with all valid configs
**WHEN** GitHub Actions runs validation workflow
**THEN** yaml-lint job SHALL pass
**AND** terraform-validate job SHALL pass for all environments
**AND** kustomize-build job SHALL pass
**AND** openspec-validate job SHALL pass
**AND** "Merge" button SHALL be enabled

#### Scenario: PR with syntax error blocked
**GIVEN** a PR with YAML syntax error in apps/homeassistant/base/deployment.yaml
**WHEN** GitHub Actions runs yamllint job
**THEN** job SHALL fail with error message indicating file and line
**AND** "Merge" button SHALL be disabled
**AND** PR shows "Some checks were not successful"

### Requirement: Validation Workflow SHALL Complete Quickly

GitHub Actions validation workflow SHALL complete in under 3 minutes to provide fast feedback.

#### Scenario: Workflow execution time
**GIVEN** a PR with typical changes (2-5 files modified)
**WHEN** GitHub Actions runs validation workflow
**THEN** total execution time SHALL be less than 180 seconds
**AND** yaml-lint job SHALL complete in < 30 seconds
**AND** terraform-validate SHALL complete in < 60 seconds per environment
**AND** kustomize-build SHALL complete in < 45 seconds

### Requirement: Dev Branch SHALL Allow Fast Iteration

The dev branch SHALL not have validation requirements, allowing force pushes and rapid iteration.

#### Scenario: Pushing to dev branch
**GIVEN** developer working on dev branch
**WHEN** pushing commits with force push
**THEN** push SHALL be allowed without validation
**AND** validation workflow MAY run (informational only)
**AND** failed validation SHALL NOT block push
