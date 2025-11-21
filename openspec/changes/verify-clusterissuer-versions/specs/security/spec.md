# Security Specification - ClusterIssuer Configuration

## ADDED Requirements

### Requirement: ClusterIssuers SHALL exist in all environments

Both letsencrypt-staging and letsencrypt-prod ClusterIssuers SHALL be deployed in every Kubernetes cluster.

#### Scenario: ClusterIssuers in dev environment
- **WHEN** Querying ClusterIssuers in dev cluster
- **THEN** both `letsencrypt-staging` and `letsencrypt-prod` SHALL exist
- **AND** both SHALL show "Ready" status
- **AND** both SHALL reference Gandi webhook for DNS-01 challenges

#### Scenario: ClusterIssuers in prod environment
- **WHEN** Querying ClusterIssuers in prod cluster
- **THEN** both `letsencrypt-staging` and `letsencrypt-prod` SHALL exist
- **AND** both SHALL show "Ready" status
- **AND** configuration SHALL match dev environment (except environment-specific secrets)

### Requirement: ClusterIssuer ACME servers SHALL be correct

Staging ClusterIssuer SHALL use Let's Encrypt staging server, production ClusterIssuer SHALL use Let's Encrypt production server.

#### Scenario: Staging ClusterIssuer uses staging ACME server
- **WHEN** Inspecting letsencrypt-staging ClusterIssuer
- **THEN** `spec.acme.server` SHALL be `https://acme-staging-v02.api.letsencrypt.org/directory`
- **AND** certificates issued SHALL be untrusted (Fake LE Intermediate)
- **AND** rate limits SHALL be higher than production

#### Scenario: Production ClusterIssuer uses production ACME server
- **WHEN** Inspecting letsencrypt-prod ClusterIssuer
- **THEN** `spec.acme.server` SHALL be `https://acme-v02.api.letsencrypt.org/directory`
- **AND** certificates issued SHALL be trusted by browsers
- **AND** rate limits SHALL apply (50 certificates per domain per week)

### Requirement: ClusterIssuers SHALL use DNS-01 challenge with Gandi

All ClusterIssuers SHALL use DNS-01 challenge method with cert-manager-webhook-gandi for validation.

#### Scenario: ClusterIssuer configured for DNS-01 with Gandi
- **WHEN** Inspecting ClusterIssuer configuration
- **THEN** `spec.acme.solvers` SHALL include dns01 solver
- **AND** solver SHALL use webhook with groupName `acme.gandi.net`
- **AND** solver SHALL use solverName `gandi`
- **AND** solver SHALL reference secret `gandi-credentials` in namespace `cert-manager`

#### Scenario: Gandi API credentials accessible
- **WHEN** cert-manager processes DNS-01 challenge
- **THEN** secret `gandi-credentials` SHALL exist in namespace `cert-manager`
- **AND** secret SHALL contain key `api-token`
- **AND** API token SHALL have valid Gandi LiveDNS permissions

### Requirement: Production certificates SHALL only use letsencrypt-prod

Production environment SHALL NOT use letsencrypt-staging for any certificates.

#### Scenario: No staging certificates in production
- **WHEN** Listing all certificates in prod cluster
- **THEN** all certificates SHALL have issuerRef `letsencrypt-prod`
- **AND** NO certificates SHALL reference `letsencrypt-staging`
- **AND** all certificates SHALL be trusted by browsers

#### Scenario: Staging certificates allowed in dev/test
- **WHEN** Listing certificates in dev or test cluster
- **THEN** certificates MAY use `letsencrypt-staging` for testing
- **AND** certificates MAY use `letsencrypt-prod` for production-like validation
- **AND** both issuers SHALL be functional

### Requirement: ClusterIssuer email SHALL be consistent

Contact email for Let's Encrypt SHALL be consistent across all ClusterIssuers and environments.

#### Scenario: Email address consistency
- **WHEN** Inspecting all ClusterIssuers across all environments
- **THEN** `spec.acme.email` SHALL be identical for all
- **AND** email SHALL be a monitored address (e.g., admin@truxonline.com)
- **AND** email SHALL receive certificate expiration warnings

### Requirement: Certificate issuance SHALL be verified in each environment

Before marking an environment as operational, certificate issuance SHALL be tested.

#### Scenario: Test certificate issuance in dev
- **GIVEN** ClusterIssuers deployed in dev
- **WHEN** Creating test Certificate resource
- **THEN** DNS-01 challenge SHALL complete successfully
- **AND** certificate SHALL be issued within 5 minutes
- **AND** certificate SHALL be stored in specified secret
- **AND** certificate SHALL have correct dnsNames

#### Scenario: Test certificate issuance in prod
- **GIVEN** ClusterIssuers deployed in prod
- **WHEN** Creating test Certificate resource with letsencrypt-prod
- **THEN** DNS-01 challenge SHALL complete successfully
- **AND** certificate SHALL be trusted by browsers
- **AND** certificate SHALL NOT count against rate limits (use staging for testing)

### Requirement: ClusterIssuer status SHALL be monitored

ClusterIssuer readiness SHALL be checked regularly to ensure functionality.

#### Scenario: ClusterIssuer shows Ready status
- **WHEN** Checking ClusterIssuer status
- **THEN** status condition SHALL show type "Ready" with status "True"
- **AND** reason SHALL be empty or "ACMEAccountRegistered"
- **AND** message SHALL NOT contain errors

#### Scenario: ClusterIssuer shows error status
- **WHEN** ClusterIssuer encounters error (e.g., invalid ACME server)
- **THEN** status condition SHALL show type "Ready" with status "False"
- **AND** reason SHALL indicate error type
- **AND** message SHALL provide troubleshooting information
- **AND** operators SHALL be alerted (via monitoring in future)

### Requirement: ClusterIssuer configuration SHALL be documented

ClusterIssuer configuration and management procedures SHALL be documented in runbooks.

#### Scenario: Runbook documents ClusterIssuer configuration
- **WHEN** Operator needs to add new ClusterIssuer or troubleshoot
- **THEN** runbook SHALL provide standard configuration
- **AND** runbook SHALL explain staging vs prod usage
- **AND** runbook SHALL document rate limits
- **AND** runbook SHALL include troubleshooting procedures

#### Scenario: Certificate management guide exists
- **WHEN** Operator needs to request or manage certificates
- **THEN** guide SHALL explain how to create Certificate resources
- **AND** guide SHALL explain how to use Ingress annotations
- **AND** guide SHALL document common errors and resolutions
