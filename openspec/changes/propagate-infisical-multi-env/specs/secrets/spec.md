# Secrets Management Specification - Multi-Environment Propagation

## MODIFIED Requirements

### Requirement: Infisical Operator SHALL be Deployed via ArgoCD

L'operator SHALL être déployé dans chaque environnement (dev, test, staging, prod).

#### Scenario: Operator deployment in test environment
- **WHEN** setting up test cluster
- **THEN** Infisical Operator SHALL be deployed via Terraform
- **AND** it SHALL run in namespace `infisical-system`
- **AND** it SHALL use Machine Identity `vixens-test-k8s-operator`

#### Scenario: Operator deployment in staging environment
- **WHEN** setting up staging cluster
- **THEN** Infisical Operator SHALL be deployed via Terraform
- **AND** it SHALL run in namespace `infisical-system`
- **AND** it SHALL use Machine Identity `vixens-staging-k8s-operator`

#### Scenario: Operator deployment in prod environment
- **WHEN** setting up prod cluster
- **THEN** Infisical Operator SHALL be deployed via Terraform
- **AND** it SHALL run in namespace `infisical-system`
- **AND** it SHALL use Machine Identity `vixens-prod-k8s-operator`

### Requirement: Applications SHALL use isolated Infisical paths

Each application SHALL store its secrets in a dedicated Infisical path to prevent naming conflicts and enable granular permissions. Paths SHALL be consistent across all environments.

#### Scenario: cert-manager retrieves secrets from isolated path (test)
- **GIVEN** cert-manager-webhook-gandi is deployed in test environment
- **WHEN** InfisicalSecret reconciles
- **THEN** it SHALL retrieve credentials from Infisical path `/cert-manager` with envSlug `test`
- **AND** the secret SHALL contain `api-token` key
- **AND** the secret SHALL be synchronized to Kubernetes namespace `cert-manager`

#### Scenario: Synology CSI retrieves secrets from isolated path (staging)
- **GIVEN** Synology CSI is deployed in staging environment
- **WHEN** Synology CSI InfisicalSecret reconciles
- **THEN** it SHALL retrieve credentials from Infisical path `/synology-csi` with envSlug `staging`
- **AND** the secret SHALL contain `synology-csi-client-info` key
- **AND** the secret SHALL be synchronized to Kubernetes namespace `synology-csi`

#### Scenario: Synology CSI retrieves secrets from isolated path (prod)
- **GIVEN** Synology CSI is deployed in prod environment
- **WHEN** Synology CSI InfisicalSecret reconciles
- **THEN** it SHALL retrieve credentials from Infisical path `/synology-csi` with envSlug `prod`
- **AND** the secret SHALL contain `synology-csi-client-info` key
- **AND** the secret SHALL be synchronized to Kubernetes namespace `synology-csi`
- **AND** it SHALL NOT contain test or staging secrets

## ADDED Requirements

### Requirement: Environment-specific secrets SHALL be isolated via envSlug

Each Kubernetes cluster SHALL access only secrets from its corresponding Infisical environment using the `envSlug` parameter in InfisicalSecret CRD.

#### Scenario: Test cluster accesses only test secrets
- **GIVEN** InfisicalSecret in test cluster with envSlug `test`
- **WHEN** Infisical Operator synchronizes secrets
- **THEN** it SHALL retrieve secrets from test environment only
- **AND** it SHALL NOT have access to dev, staging, or prod secrets

#### Scenario: Prod cluster accesses only prod secrets
- **GIVEN** InfisicalSecret in prod cluster with envSlug `prod`
- **WHEN** Infisical Operator synchronizes secrets
- **THEN** it SHALL retrieve secrets from prod environment only
- **AND** it SHALL NOT have access to dev, test, or staging secrets

### Requirement: Machine Identity credentials SHALL be environment-specific

Each Kubernetes cluster SHALL use a dedicated Machine Identity with access restricted to its environment.

#### Scenario: Test cluster uses test Machine Identity
- **GIVEN** test cluster with Infisical integration
- **WHEN** InfisicalSecret authenticates to Infisical API
- **THEN** it SHALL use Machine Identity `vixens-test-k8s-operator`
- **AND** credentials SHALL be stored in secret `infisical-universal-auth` in target namespace
- **AND** Machine Identity SHALL have access to test environment only

#### Scenario: Prod cluster uses prod Machine Identity
- **GIVEN** prod cluster with Infisical integration
- **WHEN** InfisicalSecret authenticates to Infisical API
- **THEN** it SHALL use Machine Identity `vixens-prod-k8s-operator`
- **AND** credentials SHALL be stored in secret `infisical-universal-auth` in target namespace
- **AND** Machine Identity SHALL have access to prod environment only

### Requirement: Plaintext secrets SHALL NOT exist in Git

After Infisical propagation, no plaintext secrets SHALL exist in `.secrets/` directories for any environment.

#### Scenario: No plaintext secrets in test directory
- **WHEN** Infisical integration is complete for test
- **THEN** directory `.secrets/test/` SHALL be deleted
- **AND** Git history SHALL NOT contain plaintext secrets (use git-filter-repo if needed)

#### Scenario: No plaintext secrets in prod directory
- **WHEN** Infisical integration is complete for prod
- **THEN** directory `.secrets/prod/` SHALL be deleted
- **AND** `.gitignore` SHALL ensure `.secrets/` is excluded
