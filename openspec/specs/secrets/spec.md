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

### Requirement: Applications SHALL use isolated Infisical paths

Each application SHALL store its secrets in a dedicated Infisical path to prevent naming conflicts and enable granular permissions.

#### Scenario: Synology CSI retrieves secrets from isolated path (dev)
- **GIVEN** Synology CSI is deployed in dev environment
- **WHEN** Synology CSI InfisicalSecret reconciles
- **THEN** it SHALL retrieve credentials from Infisical path `/synology-csi` (NOT root `/`)
- **AND** the secret SHALL contain `synology-csi-client-info` key
- **AND** the secret SHALL be synchronized to Kubernetes namespace `synology-csi`

#### Scenario: Path isolation prevents cross-application contamination
- **GIVEN** Multiple applications use Infisical (cert-manager, synology-csi)
- **WHEN** An InfisicalSecret syncs from path `/cert-manager`
- **THEN** it SHALL NOT include secrets from path `/synology-csi`
- **AND** each application SHALL only access secrets from its dedicated path

### Requirement: Infisical paths SHALL follow naming convention

Application paths SHALL use lowercase kebab-case matching the Kubernetes namespace or application name.

#### Scenario: Consistent path naming
- **GIVEN** An application named `synology-csi` in namespace `synology-csi`
- **WHEN** Creating Infisical path for this application
- **THEN** the path SHALL be `/synology-csi`
- **AND** it SHALL match the application/namespace name

Examples:
- `/cert-manager` for cert-manager application
- `/synology-csi` for Synology CSI application
- `/authentik` for Authentik application (future)

