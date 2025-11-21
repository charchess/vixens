# Operations Specification - Multi-Environment Promotion

## ADDED Requirements

### Requirement: Configuration Changes SHALL Progress Through Environments Sequentially

All configuration changes SHALL be promoted through environments in order: dev → test → staging → prod, with validation at each stage.

#### Scenario: Promoting change from dev to test
**GIVEN** a configuration change validated in dev environment
**WHEN** creating PR from dev branch to test branch
**THEN** GitHub Actions SHALL validate the change
**AND** merge SHALL be blocked until validation passes
**AND** after merge, ArgoCD in test cluster SHALL sync automatically
**AND** operator SHALL validate test environment before promoting to staging

#### Scenario: Skipping environment is prevented
**GIVEN** a change merged to dev branch
**WHEN** attempting to create PR from dev directly to staging (skipping test)
**THEN** PR SHALL be rejected (branch protection rules)
**AND** change MUST go through test environment first

### Requirement: Each Environment SHALL Have Isolated Infrastructure

Each environment SHALL run on dedicated nodes with separate Kubernetes clusters, preventing impact between environments.

#### Scenario: Test cluster failure does not affect dev
**GIVEN** dev and test clusters are both operational
**WHEN** test cluster experiences node failure
**THEN** dev cluster SHALL continue operating normally
**AND** applications in dev SHALL remain accessible
**AND** test cluster recovery SHALL not require dev cluster changes

#### Scenario: Each environment has separate secrets
**GIVEN** four environments (dev/test/staging/prod)
**WHEN** deploying Infisical Machine Identity credentials
**THEN** each environment SHALL use different Machine Identity
**AND** test identity SHALL NOT have access to prod Infisical environment
**AND** compromise of test credentials SHALL NOT expose prod secrets

### Requirement: Production Deployment SHALL Require Manual Approval

Merging changes to main branch (prod) SHALL require manual approval to prevent accidental production deployments.

#### Scenario: PR to prod requires approval
**GIVEN** a PR from staging to main branch
**WHEN** all validation checks pass
**THEN** merge button SHALL remain disabled until approval
**AND** operator SHALL manually review changes
**AND** operator SHALL approve explicitly before merge

#### Scenario: Prod deployment monitored carefully
**GIVEN** PR to prod is approved and merged
**WHEN** ArgoCD syncs applications in prod cluster
**THEN** operator SHALL monitor sync progress actively
**AND** operator SHALL validate each application after sync
**AND** operator SHALL NOT deploy multiple changes simultaneously to prod
