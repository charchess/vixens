# Secrets Spec Delta - Synology CSI Path Isolation

## MODIFIED Requirements

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
