# Secrets Management Specification - Infisical Bootstrap

## MODIFIED Requirements

### Requirement: Infisical Machine Identity Credentials SHALL be Deployed via Terraform

Infisical Kubernetes Operator authentication credentials SHALL be deployed automatically by Terraform from `.secrets/<env>/` directory (gitignored), not hardcoded in Git manifests.

#### Scenario: Bootstrap Infisical authentication in dev environment
**GIVEN** a fresh dev cluster provisioned by Terraform
**WHEN** Terraform apply completes
**THEN** secret `infisical-universal-auth` SHALL exist in namespace `cert-manager`
**AND** secret `infisical-universal-auth` SHALL exist in namespace `synology-csi`
**AND** secrets SHALL contain keys `clientId` and `clientSecret`
**AND** secrets SHALL be labeled `managed-by=terraform`

#### Scenario: InfisicalSecret CRD authenticates with Terraform-deployed credentials
**GIVEN** InfisicalSecret CRD `gandi-credentials-sync` in namespace `cert-manager`
**WHEN** Infisical Operator reconciles the CRD
**THEN** it SHALL authenticate using secret `infisical-universal-auth` in same namespace
**AND** it SHALL successfully retrieve secrets from Infisical path `/cert-manager`
**AND** it SHALL NOT fail with authentication errors

#### Scenario: Different Machine Identity per environment
**GIVEN** four environments (dev, test, staging, prod)
**WHEN** deploying each environment with Terraform
**THEN** dev SHALL use Machine Identity `vixens-dev-k8s-operator`
**AND** test SHALL use Machine Identity `vixens-test-k8s-operator`
**AND** staging SHALL use Machine Identity `vixens-staging-k8s-operator`
**AND** prod SHALL use Machine Identity `vixens-prod-k8s-operator`
**AND** each identity SHALL have access ONLY to its corresponding Infisical environment

## ADDED Requirements

### Requirement: Machine Identity Credentials SHALL NOT be Committed to Git

Machine Identity credentials (clientId, clientSecret) SHALL be stored in `.secrets/<env>/infisical-machine-identity.yml` which is gitignored, never committed to repository.

#### Scenario: Checking Git repository for exposed credentials
**GIVEN** the bootstrap-infisical-terraform change is implemented
**WHEN** searching Git repository for Machine Identity credentials
**THEN** running `rg "clientId|clientSecret" apps/` SHALL return zero results
**AND** running `git log --all --full-history --source -- '*infisical-auth-secret.yaml'` SHALL show files deleted
**AND** `.secrets/` directory SHALL be listed in `.gitignore`

#### Scenario: Creating secrets file for new environment
**GIVEN** a new environment `test` being provisioned
**WHEN** operator creates `.secrets/test/infisical-machine-identity.yml`
**THEN** file SHALL contain YAML with keys `clientId` and `clientSecret`
**AND** file permissions SHALL be `600` (readable only by owner)
**AND** file SHALL NOT appear in `git status` output

### Requirement: Terraform Module SHALL Deploy Secrets Before Applications

The `infisical-bootstrap` Terraform module SHALL deploy Machine Identity secrets with proper dependency ordering to ensure availability before ArgoCD applications need them.

#### Scenario: Secrets available before ArgoCD app sync
**GIVEN** Terraform is applying infrastructure for dev environment
**WHEN** ArgoCD bootstraps and begins syncing applications
**THEN** secrets SHALL already exist in target namespaces
**AND** InfisicalSecret CRDs SHALL NOT fail due to missing authentication credentials
**AND** Terraform module SHALL use `depends_on` to wait for ArgoCD bootstrap completion

#### Scenario: Module deploys secrets to multiple namespaces
**GIVEN** Terraform module `infisical-bootstrap` with list of target namespaces
**WHEN** module is applied
**THEN** it SHALL create secret `infisical-universal-auth` in each namespace
**AND** all secrets SHALL contain identical credentials (same Machine Identity)
**AND** module SHALL use `for_each` to iterate over namespace list

### Requirement: Secrets File Format SHALL be Standard YAML

Machine Identity credentials file SHALL use simple YAML format with top-level keys `clientId` and `clientSecret` for easy manual editing.

#### Scenario: Parsing secrets file in Terraform
**GIVEN** file `.secrets/dev/infisical-machine-identity.yml` with content:
```yaml
clientId: "ee279e5e-82b6-476b-9643-093898807f35"
clientSecret: "ed8de635fd4b5818861e842fb1a03722bb7a35bda478b432d4d370609d12aefe"
```
**WHEN** Terraform reads file with `yamldecode(file(...))`
**THEN** it SHALL successfully parse to map with keys `clientId` and `clientSecret`
**AND** values SHALL be strings (not decoded/encrypted)
