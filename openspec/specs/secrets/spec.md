# Secrets Management Specification - Vixens

## Purpose
Définit comment les secrets sensibles sont gérés via Infisical Operator.
## Requirements
### Requirement: All Secrets SHALL be Managed via Infisical
Aucun secret SHALL exister en clair dans les fichiers YAML.

#### Scenario: CSI driver credentials
- **WHEN** synology-csi is deployed
- **THEN** it SHALL retrieve credentials from Infisical path `/vixens/{env}/synology-csi`
- **AND** it SHALL NOT contain hardcoded username/password

#### Scenario: Cert-manager credentials
- **WHEN** cert-manager needs Gandi API key
- **THEN** it SHALL retrieve it from Infisical path `/vixens/{env}/cert-manager/gandi`

### Requirement: Infisical Operator SHALL be Deployed via ArgoCD
L'operator SHALL être déployé dans chaque environnement.

#### Scenario: Operator deployment
- **WHEN** setting up a new cluster
- **THEN** Infisical Operator SHALL be deployed via ArgoCD
- **AND** it SHALL run in namespace `infisical-system`

### Requirement: Infisical Integration for Secrets Management
The system SHALL integrate with Infisical for centralized and externalized secrets management.

#### Scenario: Application retrieves a secret
- **WHEN** an application starts or requires a secret
- **THEN** the application SHALL securely retrieve the necessary secret from Infisical.

#### Scenario: Secure storage of secrets
- **WHEN** a new secret is created or an existing secret is updated
- **THEN** the secret SHALL be stored in Infisical with appropriate encryption and access controls.

