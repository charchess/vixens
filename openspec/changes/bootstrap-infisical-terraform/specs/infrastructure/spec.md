# Infrastructure Specification - Terraform Bootstrap

## ADDED Requirements

### Requirement: Terraform Module SHALL Exist for Infisical Bootstrap

A reusable Terraform module SHALL be created at `terraform/modules/infisical-bootstrap/` to deploy Infisical Machine Identity credentials from external files.

#### Scenario: Module structure and files
**GIVEN** the infisical-bootstrap module is implemented
**WHEN** examining module directory `terraform/modules/infisical-bootstrap/`
**THEN** file `main.tf` SHALL exist with kubectl_manifest resources
**AND** file `variables.tf` SHALL exist with inputs (environment, argocd_bootstrap_complete)
**AND** file `versions.tf` SHALL exist requiring kubectl provider ~> 2.0
**AND** file `outputs.tf` SHALL exist exposing secret metadata

#### Scenario: Module reads secrets from .secrets directory
**GIVEN** module is called with `environment = "dev"`
**WHEN** module executes `yamldecode(file(...))`
**THEN** it SHALL read file `.secrets/dev/infisical-machine-identity.yml`
**AND** it SHALL fail with clear error if file does not exist
**AND** error message SHALL indicate which file is missing

### Requirement: Environment Terraform SHALL Call Infisical Bootstrap Module

Each environment's Terraform configuration SHALL include a module call to `infisical-bootstrap` after core infrastructure is provisioned.

#### Scenario: Dev environment calls bootstrap module
**GIVEN** file `terraform/environments/dev/main.tf`
**WHEN** reviewing module calls
**THEN** it SHALL include module call to `infisical-bootstrap`
**AND** module SHALL receive input `environment = var.environment`
**AND** module SHALL receive input `argocd_bootstrap_complete = module.environment.argocd_bootstrap_complete`
**AND** module call SHALL come after `module.environment` (dependency ordering)

#### Scenario: Module depends on ArgoCD bootstrap
**GIVEN** Terraform is applying dev environment
**WHEN** infisical-bootstrap module executes
**THEN** it SHALL NOT run until ArgoCD has created target namespaces
**AND** it SHALL use `depends_on` or input variable to enforce ordering
**AND** Terraform plan SHALL show secrets created after ArgoCD resources
